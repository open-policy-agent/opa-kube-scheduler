// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/repl"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/rego-scheduler/pkg"
)

type config struct {
	showVersion bool
	clusterURL  string
	policyFile  string
	fitDoc      string
}

func loadDataStore() *storage.DataStore {
	ds := storage.NewDataStore()
	return ds
}

func loadPolicyFile(ps *storage.PolicyStore, policyFile string) error {
	f, err := os.Open(policyFile)
	if err != nil {
		return err
	}
	defer f.Close()

	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	mod, err := ast.ParseModule(policyFile, string(bs))
	if err != nil {
		return err
	}

	mods := map[string]*ast.Module{
		policyFile: mod,
	}

	c := ast.NewCompiler()
	if c.Compile(mods); c.Failed() {
		return c.Errors[0]
	}

	err = ps.Add(policyFile, mods[policyFile], nil, false)
	if err != nil {
		return err
	}

	return nil
}

func setPackage(r *repl.REPL, fitDoc []interface{}) {
	path := []string{}
	for _, x := range fitDoc[:len(fitDoc)-1] {
		path = append(path, x.(string))
	}
	r.OneShot(fmt.Sprintf("package %v", strings.Join(path, ".")))
}

func watchPolicyFile(policyFile string, f func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case evt := <-w.Events:
				if evt.Op&fsnotify.Write != 0 {
					f()
				}
			}
		}
	}()
	return nil
}

func loadPolicyStore(ds *storage.DataStore) *storage.PolicyStore {
	ps := storage.NewPolicyStore(ds, "")
	return ps
}

func parseArgs() *config {
	c := config{}
	flag.StringVar(&c.clusterURL, "cluster_url", "http://localhost:8080/api/v1", "set the Kubernetes API URL")
	flag.StringVar(&c.policyFile, "policy", "etc/policy.rego", "set the path of the policy definition")
	flag.StringVar(&c.fitDoc, "fit", "/io/k8s/scheduler/fit", "set the path of the fit document")
	flag.Parse()
	return &c
}

func printVersion() {
	fmt.Println(pkg.Version)
}

func runREPL(c *config) {
	ds := loadDataStore()
	ps := loadPolicyStore(ds)

	err := loadPolicyFile(ps, c.policyFile)
	if err != nil {
		glog.Errorf("error loading policy: %v", err)
		os.Exit(1)
	}

	err = watchPolicyFile(c.policyFile, func() {
		err := loadPolicyFile(ps, c.policyFile)
		if err != nil {
			glog.Errorf("error reloading policy: %v", err)
		}
	})
	if err != nil {
		glog.Errorf("error watching policy: %v", err)
		os.Exit(1)
	}

	fit := []interface{}{}
	for _, x := range strings.Split(c.fitDoc[1:], "/") {
		fit = append(fit, x)
	}

	scheduler := pkg.New(ds, fit, c.clusterURL)
	glog.Info("Starting scheduler...")
	scheduler.Start()

	defer glog.Flush()
	go func() {
		for _ = range time.Tick(1 * time.Second) {
			glog.Flush()
		}
	}()

	r := repl.New(ds, ps, "", os.Stdout, "json", "")
	setPackage(r, fit)
	r.Loop()
}

func main() {
	c := parseArgs()
	if c.showVersion {
		printVersion()
	} else {
		runREPL(c)
	}
}
