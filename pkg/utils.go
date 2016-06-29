// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package pkg

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

func httpErr(req *http.Request, resp *http.Response) error {
	if resp.Header.Get("Content-Type") == "application/json" {
		decoder := json.NewDecoder(resp.Body)
		var v interface{}
		if err := decoder.Decode(&v); err != nil {
			return errors.Wrapf(err, "response for %v %v", req.Method, req.URL.String())
		}
		return errors.Errorf("response for %v %v: %v", req.Method, req.URL.String(), v)
	}
	return errors.Errorf("response for %v %v:  %v", req.Method, req.URL.String(), resp.StatusCode)
}
