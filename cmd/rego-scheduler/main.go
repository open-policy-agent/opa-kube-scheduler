// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/open-policy-agent/opa/server"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/rego-scheduler/pkg"
)

func cmdServer(c *config) {

	if err := os.MkdirAll(c.policyDir, 0755); err != nil {
		glog.Fatalf("Unable to create policy directory: %v.", err)
	}

	store := storage.New(storage.InMemoryConfig().WithPolicyDir(c.policyDir))

	if err := store.Open(); err != nil {
		glog.Fatalf("Unable open storage: %v.", err)
	}

	server := server.New(store, c.listenAddr, true)
	scheduler := pkg.New(server, store, parsePath(c.fitDoc), c.clusterURL)

	if err := scheduler.Start(); err != nil {
		glog.Fatalf("Unable to start scheduler: %v.", err)
	}

	if err := server.Loop(); err != nil {
		glog.Fatalf("Server exited: %v.", err)
	}
}

func cmdPrintVersion() {
	fmt.Println(pkg.Version)
}

type config struct {
	showVersion bool
	listenAddr  string
	policyDir   string
	clusterURL  string
	fitDoc      string
}

func parseArgs() *config {
	c := config{}
	flag.BoolVar(&c.showVersion, "version", false, "print the scheduler version and exit")
	flag.StringVar(&c.listenAddr, "listen_addr", ":8181", "set the listening address of the server")
	flag.StringVar(&c.policyDir, "policy_dir", "policies", "set the path of the directory to store policies")
	flag.StringVar(&c.clusterURL, "cluster_url", "http://localhost:8001/api/v1", "set the Kubernetes API URL")
	flag.StringVar(&c.fitDoc, "fit", "/io/k8s/scheduler/fit", "set the path of the fit document")
	flag.Parse()
	return &c
}

func parsePath(p string) []interface{} {
	if p[0] != '/' {
		glog.Fatalf("Invalid path: %v", p)
	}
	parts := strings.Split(p[1:], "/")
	r := make([]interface{}, len(parts))
	for i := range parts {
		r[i] = parts[i]
	}
	return r
}

func main() {
	c := parseArgs()
	if c.showVersion {
		cmdPrintVersion()
	} else {
		cmdServer(c)
	}
}
