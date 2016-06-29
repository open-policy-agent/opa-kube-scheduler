// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package pkg

import (
	"fmt"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/topdown"
	"github.com/pkg/errors"
	"k8s.io/kubernetes/pkg/api/resource"
)

var quantityToNumber = ast.Var("q2n")
var quantityToNumberScaled = ast.Var("q2ns")

func q2n(ctx *topdown.Context, expr *ast.Expr, iter topdown.Iterator) (err error) {

	ops := expr.Terms.([]*ast.Term)
	a, b := ops[1], ops[2]

	str, err := topdown.ValueToString(a.Value, ctx)
	if err != nil {
		return errors.Wrapf(err, "q2n")
	}

	q, err := resource.ParseQuantity(str)
	if err != nil {
		return errors.Wrapf(err, "q2n")
	}

	n := ast.Number(q.ScaledValue(0))

	bv, ok := b.Value.(ast.Var)
	if !ok {
		return fmt.Errorf("q2n: destination must be an unbound variable")
	}

	return topdown.Continue(ctx, bv, n, iter)
}

func q2ns(ctx *topdown.Context, expr *ast.Expr, iter topdown.Iterator) (err error) {

	ops := expr.Terms.([]*ast.Term)
	a, b, c := ops[1], ops[2], ops[3]

	str, err := topdown.ValueToString(a.Value, ctx)
	if err != nil {
		return errors.Wrapf(err, "q2ns")
	}

	q, err := resource.ParseQuantity(str)
	if err != nil {
		return errors.Wrapf(err, "q2ns")
	}

	f, err := topdown.ValueToFloat64(b.Value, ctx)
	if err != nil {
		return errors.Wrapf(err, "q2ns")
	}

	n := ast.Number(q.ScaledValue(resource.Scale(f)))

	cv, ok := c.Value.(ast.Var)
	if !ok {
		return fmt.Errorf("q2ns: destination must be an unbound variable")
	}

	return topdown.Continue(ctx, cv, n, iter)
}

func init() {

	ast.RegisterBuiltin(&ast.Builtin{
		Name:      quantityToNumber,
		NumArgs:   2,
		TargetPos: []int{1},
	})

	topdown.RegisterBuiltinFunc(quantityToNumber, q2n)

	ast.RegisterBuiltin(&ast.Builtin{
		Name:      quantityToNumberScaled,
		NumArgs:   3,
		TargetPos: []int{2},
	})

	topdown.RegisterBuiltinFunc(quantityToNumberScaled, q2ns)
}
