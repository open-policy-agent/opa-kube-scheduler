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

func loadPolicy(policyFile string) *ast.Compiler {
	f, err := os.Open(policyFile)
	if err != nil {
		glog.Fatalf("Failed to open policy file: %v", err)
	}
	defer f.Close()

	bs, err := ioutil.ReadAll(f)
	if err != nil {
		glog.Fatalf("Failed to read policy file: %v", err)
	}

	mod, err := ast.ParseModule(policyFile, string(bs))
	if err != nil {
		glog.Fatalf("Failed to parse policy file: %v", err)
	}

	mods := map[string]*ast.Module{
		policyFile: mod,
	}

	c := ast.NewCompiler()
	if c.Compile(mods); c.Failed() {
		glog.Fatalf("Failed to compile policy file: %v", c.FlattenErrors())
	}

	return c
}

func storePolicy(store *storage.Storage, modules map[string]*ast.Module) {
	txn, err := store.NewTransaction()
	if err != nil {
		glog.Fatalf("Failed to open transaction: %v", err)
	}
	defer store.Close(txn)
	for id, mod := range modules {
		if err := store.InsertPolicy(txn, id, mod, nil, false); err != nil {
			glog.Fatalf("Failed to store policy: %v", err)
		}
	}
}

func setPackage(r *repl.REPL, fitDoc string) {
	path := strings.Split(fitDoc[1:], "/")
	path = path[:len(path)-1]
	r.OneShot(fmt.Sprintf("package %v", strings.Join(path, ".")))
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

	store := storage.New(storage.InMemoryConfig())
	compiler := loadPolicy(c.policyFile)
	storePolicy(store, compiler.Modules)

	fit := []interface{}{}
	for _, x := range strings.Split(c.fitDoc[1:], "/") {
		fit = append(fit, x)
	}

	scheduler := pkg.New(compiler, store, fit, c.clusterURL)
	glog.Info("Starting scheduler...")
	scheduler.Start()

	defer glog.Flush()
	go func() {
		for _ = range time.Tick(1 * time.Second) {
			glog.Flush()
		}
	}()

	r := repl.New(store, "", os.Stdout, "json", "")
	setPackage(r, c.fitDoc)
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
