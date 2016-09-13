// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package pkg

import (
	"strings"

	"github.com/golang/glog"
	"github.com/open-policy-agent/opa/topdown"
)

// glogtracer implements the topdown.Tracer interface to emit trace messages via
// glog. Tracing can be enabled by running with --v=3.
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
