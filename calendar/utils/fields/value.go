// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package fields

import "time"

type Value interface {
	Equals(Value) bool

	// For a single value, returns its String() method; for multi-value returns
	// the list of stringified values. For a composite value, returns nil.
	Strings() []string

	// For a composite value returns its fields, otehrwise returns nil.
	Fields() Fields
}

type stringValue struct {
	v string
}

func NewStringValue(s string) Value      { return &stringValue{s} }
func (sv stringValue) Strings() []string { return []string{sv.v} }
func (sv stringValue) Fields() Fields    { return nil }
func (sv stringValue) Equals(v Value) bool {
	other, ok := v.(*stringValue)
	if !ok {
		return false
	}
	return sv.v == other.v
}

type timeValue struct {
	v time.Time
}

func NewTimeValue(t time.Time) Value   { return &timeValue{t} }
func (tv timeValue) Strings() []string { return []string{tv.v.Format(time.RFC3339)} }
func (tv timeValue) Fields() Fields    { return nil }
func (tv timeValue) Equals(v Value) bool {
	other, ok := v.(*timeValue)
	if !ok {
		return false
	}
	return tv.v == other.v
}

type multiValue struct {
	v []Value
}

func NewMultiValue(vv ...Value) Value { return &multiValue{vv} }

func (mv multiValue) Strings() []string {
	var result []string
	for _, v := range mv.v {
		result = append(result, v.Strings()...)
	}
	return result
}

func (mv multiValue) Fields() Fields { return nil }

func (mv multiValue) Equals(v Value) bool {
	other, ok := v.(*multiValue)
	if !ok || len(mv.v) != len(other.v) {
		return false
	}
	for i, o := range other.v {
		if !o.Equals(mv.v[i]) {
			return false
		}
	}
	return true
}
