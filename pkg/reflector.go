// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"k8s.io/kubernetes/pkg/client/restclient"

	"github.com/golang/glog"
)

type resync struct {
	Items []interface{} `json:"items"`
}

type sync struct {
	Type   string                 `json:"type"`
	Object map[string]interface{} `json:"object"`
}

const (
	added    = "ADDED"
	modified = "MODIFIED"
	deleted  = "DELETED"
)

const (
	backoffDelay = 5 * time.Second
)

type reflector struct {
	Rx     chan interface{}
	config *restclient.Config
	URL    *url.URL
}

func newReflector(config *restclient.Config, resourceType string, fieldSelector string) (*reflector, error) {

	u, err := url.Parse(baseURLFor(config) + "/" + resourceType)

	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Add("fieldSelector", fieldSelector)
	u.RawQuery = q.Encode()

	r := &reflector{
		Rx:     make(chan interface{}),
		URL:    u,
		config: config,
	}

	return r, nil
}

func (r *reflector) Start() {
	go func() {
		for {
			glog.V(2).Infof("Reflector restarting: %v", r.URL)
			items, version, err := r.list()
			if err != nil {
				r.Rx <- err
				time.Sleep(backoffDelay)
				continue
			}
			r.Rx <- &resync{items}
			if err := r.watch(version); err != nil {
				if err != io.EOF {
					r.Rx <- err
				}
			}
		}
	}()
}

func (r *reflector) list() ([]interface{}, string, error) {

	req, err := http.NewRequest("GET", r.URL.String(), nil)
	if err != nil {
		return nil, "", err
	}

	client, err := clientFor(r.config)
	if err != nil {
		return nil, "", err
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", httpErr(req, resp)
	}

	decoder := json.NewDecoder(resp.Body)

	var v map[string]interface{}
	if err := decoder.Decode(&v); err != nil {
		return nil, "", err
	}

	if m, ok := v["metadata"].(map[string]interface{}); ok {
		if version, ok := m["resourceVersion"].(string); ok {
			if items, ok := v["items"].([]interface{}); ok {
				return items, version, nil
			}
		}
	}

	return nil, "", fmt.Errorf("malformed response: %v", v)
}

func (r *reflector) watch(version string) error {

	u := *r.URL
	q := u.Query()
	q.Add("watch", "true")
	q.Add("resourceVersion", version)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	client, err := clientFor(r.config)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return httpErr(req, resp)
	}

	decoder := json.NewDecoder(resp.Body)

	for {
		v := &sync{}
		if err := decoder.Decode(v); err != nil {
			return err
		}
		r.Rx <- v
	}
}
