// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package test

import (
	"fmt"

	"github.com/golang/mock/gomock"
)

// Source: https://github.com/golang/mock/issues/43#issuecomment-1292042897

// doMatch keeps state of the custom lambda matcher.
// match is a lambda function that asserts actual value matching.
// x keeps actual value.
type doMatch[V any] struct {
	match func(v V) bool
	x     any
}

// DoMatch creates lambda matcher instance equipped with
// lambda function to detect if actual value matches
// some arbitrary criteria.
// Lambda matcher implements gomock customer matcher
// interface https://github.com/golang/mock/blob/5b455625bd2c8ffbcc0de6a0873f864ba3820904/gomock/matchers.go#L25.
// Sample of usage:
//
// mock.EXPECT().Foo(gomock.All(
//
//	   DoMatch(func(v Bar) bool {
//		      return v.Greeting == "Hello world"
//	   }),
//
// ))
func DoMatch[V any](m func(v V) bool) gomock.Matcher {
	return &doMatch[V]{
		match: m,
	}
}

// Matches receives actual value x casts it to specific type defined as a type parameter V
// and calls labmda function 'match' to resolve if x matches or not.
func (o *doMatch[V]) Matches(x any) bool {
	o.x = x
	v, ok := x.(V)
	if !ok {
		return false
	}

	return o.match(v)
}

// String describes what matcher matches.
func (o *doMatch[V]) String() string {
	return fmt.Sprintf("is matched to %v", o.x)
}
