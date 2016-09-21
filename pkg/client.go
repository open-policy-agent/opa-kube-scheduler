// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package pkg

import (
	"net/http"

	"k8s.io/kubernetes/pkg/client/restclient"
)

func baseURLFor(config *restclient.Config) string {
	return config.Host + "/api/v1"
}

func clientFor(config *restclient.Config) (*http.Client, error) {

	transport, err := restclient.TransportFor(config)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: transport,
	}

	return client, nil
}
