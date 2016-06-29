// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package pkg

type metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type target struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

type binding struct {
	APIVersion string   `json:"apiVerson"`
	Kind       string   `json:"kind"`
	Metadata   metadata `json:"metadata"`
	Target     target   `json:"target"`
}
