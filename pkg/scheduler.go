// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/topdown"
)

// Scheduler implements the business logic that integrates Kubernetes with OPA.
type Scheduler struct {
	ds         *storage.DataStore
	fit        []interface{}
	clusterURL string
}

// New returns a new Scheduler object that can be started.
func New(ds *storage.DataStore, fit []interface{}, clusterURL string) *Scheduler {
	return &Scheduler{
		ds:         ds,
		fit:        fit,
		clusterURL: clusterURL,
	}
}

// Start causes the scheduler to begin scheduling pods.
func (s *Scheduler) Start() {
	s.init()
	s.run()
}

func (s *Scheduler) init() {
	baseDocs := []interface{}{
		"pods", "nodes", "replicationcontrollers", "services",
	}
	for _, x := range baseDocs {
		if err := s.ds.Patch(storage.AddOp, []interface{}{x}, map[string]interface{}{}); err != nil {
			glog.Errorf("Error creating %v document: %v", x, err)
			return
		}
	}
}

func (s *Scheduler) run() {

	// This table defines the reflectors that will be started. The action
	// is the function that will be called when a message from the reflector
	// is received.
	//
	// TODO(tsandall): implement barrier so that the unscheduled pod reflector
	// does not start until all of the other reflectors have sent resync messages.
	// currently if scheduler is started while there are unscheduled pods, they
	// will fail to schedule (because no nodes have synched).
	spec := []struct {
		action        action
		resourceType  string
		fieldSelector string
	}{
		{s.schedule, "pods", "spec.nodeName==,status.phase!=Succeeded,status.phase!=Failed"},
		{s.patch, "pods", "spec.nodeName!=,status.phase!=Succeeded,status.phase!=Failed"},
		{s.patch, "nodes", "spec.unschedulable=false"},
		{s.patch, "services", ""},
		{s.patch, "replicationcontrollers", ""},
	}

	mux := make(chan *msg)

	// Start the reflectors.
	for _, sp := range spec {
		r, err := newReflector(s.clusterURL, sp.resourceType, sp.fieldSelector)
		if err != nil {
			return
		}
		r.Start()
		sp := sp
		go func() {
			for x := range r.Rx {
				mux <- &msg{
					action:       sp.action,
					resourceType: sp.resourceType,
					payload:      x,
				}
			}
		}()
	}

	// Process updates from the reflectors.
	go func() {
		for msg := range mux {
			if err := msg.action(msg.resourceType, msg.payload); err != nil {
				glog.Errorf("Error handling update (%v) for %v: %v", msg.kind(), msg.resourceType, err)
			}
		}
	}()
}

func (s *Scheduler) schedule(resourceType string, payload interface{}) error {
	switch payload := payload.(type) {
	case *resync:
		for _, item := range payload.Items {
			if err := s.schedulePod(item.(map[string]interface{})); err != nil {
				return err
			}
		}
	case *sync:
		if payload.Type == added {
			return s.schedulePod(payload.Object)
		}
	case error:
		return payload
	}
	return nil
}

func (s *Scheduler) schedulePod(pod map[string]interface{}) error {

	uid, err := s.getUID(pod)
	if err != nil {
		return err
	}

	val, err := ast.InterfaceToValue(pod)
	if err != nil {
		return err
	}

	globals := storage.NewBindings()
	globals.Put(ast.Var("requested_pod"), val)

	t0 := time.Now()

	params := topdown.NewQueryParams(s.ds, globals, s.fit)
	params.Tracer = &glogtracer{}

	podName := pod["metadata"].(map[string]interface{})["name"]

	results, err := topdown.Query(params)
	if err != nil {
		return err
	}

	queryTime := time.Since(t0)

	rankings := rankings{}

	for k, v := range results.(map[string]interface{}) {
		w := v.(float64)
		rankings = append(rankings, ranking{k, w})
	}

	sort.Sort(rankings)

	if len(rankings) == 0 {
		glog.Infof("Unable to schedule pod: %v: no nodes are available (took query:%v)", podName, queryTime)
		return nil
	}

	nodeName := rankings[len(rankings)-1].nodeName

	spec := pod["spec"].(map[string]interface{})
	spec["nodeName"] = nodeName
	path := []interface{}{"pods", uid}

	t0 = time.Now()

	if err := s.ds.Patch(storage.AddOp, path, pod); err != nil {
		return err
	}

	storageTime := time.Since(t0)

	t0 = time.Now()

	if err := s.bindPod(pod); err != nil {
		glog.Errorf("Failed to bind pod %v: %v", podName, err)
		if err2 := s.ds.Patch(storage.RemoveOp, path, nil); err2 != nil {
			return err2
		}
		return err
	}

	bindTime := time.Since(t0)

	glog.Infof("Scheduling pod %v to %v (took query:%v storage:%v bind:%v)", podName, nodeName, queryTime, storageTime, bindTime)

	return nil
}

func (s *Scheduler) bindPod(pod map[string]interface{}) error {

	podName, err := s.getMetadata("name", pod)
	if err != nil {
		return err
	}

	nodeName, err := s.getNodeName(pod)
	if err != nil {
		return err
	}

	namespace, err := s.getMetadata("namespace", pod)
	if err != nil {
		return err
	}

	b := binding{
		APIVersion: "v1",
		Kind:       "Binding",
		Metadata: metadata{
			Name:      podName,
			Namespace: namespace,
		},
		Target: target{
			APIVersion: "v1",
			Kind:       "Node",
			Name:       nodeName,
		},
	}

	buf := bytes.NewBuffer([]byte{})
	if err := json.NewEncoder(buf).Encode(b); err != nil {
		return err
	}

	path := fmt.Sprintf("%v/namespaces/%v/bindings", s.clusterURL, namespace)
	req, err := http.NewRequest("POST", path, buf)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode > http.StatusCreated {
		return httpErr(req, resp)
	}

	return nil
}

func (s *Scheduler) patch(resourceType string, payload interface{}) error {
	switch payload := payload.(type) {
	case *resync:
		// TODO(tsandall): handle stale objects
		for _, item := range payload.Items {
			if err := s.patchOp(resourceType, storage.AddOp, item); err != nil {
				return err
			}
		}
	case *sync:
		switch payload.Type {
		case added:
			return s.patchOp(resourceType, storage.AddOp, payload.Object)
		case modified:
			return s.patchOp(resourceType, storage.ReplaceOp, payload.Object)
		case deleted:
			return s.patchOp(resourceType, storage.RemoveOp, payload.Object)
		}
	case error:
		return payload
	}
	return nil
}

func (s *Scheduler) patchOp(resourceType string, op storage.PatchOp, obj interface{}) error {
	uid, err := s.getUID(obj)
	if err != nil {
		return err
	}
	path := []interface{}{resourceType, uid}
	return s.ds.Patch(op, path, obj)
}

func (s *Scheduler) getNodeName(pod map[string]interface{}) (string, error) {
	if m, ok := pod["spec"].(map[string]interface{}); ok {
		if v, ok := m["nodeName"].(string); ok {
			return v, nil
		}
	}
	return "", fmt.Errorf("malformed pod: %v", pod)
}

func (s *Scheduler) getUID(obj interface{}) (string, error) {
	return s.getMetadata("uid", obj)
}

func (s *Scheduler) getMetadata(key string, obj interface{}) (string, error) {
	if obj, ok := obj.(map[string]interface{}); ok {
		if m, ok := obj["metadata"].(map[string]interface{}); ok {
			if u, ok := m[key].(string); ok {
				return u, nil
			}
		}
	}
	return "", fmt.Errorf("malformed object: %v", obj)
}

type action func(string, interface{}) error

type msg struct {
	action       action
	resourceType string
	payload      interface{}
}

func (m *msg) kind() string {
	switch p := m.payload.(type) {
	case *sync:
		return p.Type
	case *resync:
		return "RESYNC"
	case error:
		return "ERROR"
	}
	return "unknown payload type"
}

type ranking struct {
	nodeName string
	weight   float64
}

type rankings []ranking

func (r rankings) Len() int {
	return len(r)
}

func (r rankings) Less(i, j int) bool {
	return r[i].weight < r[j].weight
}

func (r rankings) Swap(i, j int) {
	tmp := r[i]
	r[i] = r[j]
	r[j] = tmp
}

type glogtracer struct{}

func (t *glogtracer) Enabled() bool {
	return bool(glog.V(3))
}

func (t *glogtracer) Trace(ctx *topdown.Context, f string, a ...interface{}) {
	var padding string
	i := 0
	for ; ctx != nil; ctx = ctx.Previous {
		padding += strings.Repeat(" ", ctx.Index+i)
		i++
	}
	f = padding + f + "\n"
	glog.V(3).Infof(f, a...)
}
