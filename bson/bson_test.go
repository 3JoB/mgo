// BSON library for Go
//
// Copyright (c) 2010-2012 - Gustavo Niemeyer <gustavo@niemeyer.net>
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
// gobson - BSON library for Go.

package bson_test

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v2"

	"github.com/3JoB/mgo/bson"
)

func TestAll(t *testing.T) {
	TestingT(t)
}

type S struct{}

var _ = Suite(&S{})

// Wrap up the document elements contained in data, prepending the int32
// length of the data, and appending the '\x00' value closing the document.
func wrapInDoc(data string) string {
	result := make([]byte, len(data)+5)
	binary.LittleEndian.PutUint32(result, uint32(len(result)))
	copy(result[4:], []byte(data))
	return string(result)
}

func makeZeroDoc(value any) (zero any) {
	v := reflect.ValueOf(value)
	t := v.Type()
	switch t.Kind() {
	case reflect.Map:
		mv := reflect.MakeMap(t)
		zero = mv.Interface()
	case reflect.Ptr:
		pv := reflect.New(v.Type().Elem())
		zero = pv.Interface()
	case reflect.Slice, reflect.Int, reflect.Int64, reflect.Struct:
		zero = reflect.New(t).Interface()
	default:
		panic("unsupported doc type: " + t.Name())
	}
	return zero
}

func testUnmarshal(c *C, data string, obj any) {
	zero := makeZeroDoc(obj)
	err := bson.Unmarshal([]byte(data), zero)
	c.Assert(err, IsNil)
	c.Assert(zero, DeepEquals, obj)
}

type testItemType struct {
	obj  any
	data string
}

// --------------------------------------------------------------------------
// Samples from bsonspec.org:

var sampleItems = []testItemType{
	{obj: bson.M{"hello": "world"},
		data: "\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00"},
	{obj: bson.M{"BSON": []any{"awesome", float64(5.05), 1986}},
		data: "1\x00\x00\x00\x04BSON\x00&\x00\x00\x00\x020\x00\x08\x00\x00\x00" +
			"awesome\x00\x011\x00333333\x14@\x102\x00\xc2\x07\x00\x00\x00\x00"},
}

func (s *S) TestMarshalSampleItems(c *C) {
	for i, item := range sampleItems {
		data, err := bson.Marshal(item.obj)
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, item.data, Commentf("Failed on item %d", i))
	}
}

func (s *S) TestUnmarshalSampleItems(c *C) {
	for i, item := range sampleItems {
		value := bson.M{}
		err := bson.Unmarshal([]byte(item.data), value)
		c.Assert(err, IsNil)
		c.Assert(value, DeepEquals, item.obj, Commentf("Failed on item %d", i))
	}
}

// --------------------------------------------------------------------------
// Every type, ordered by the type flag. These are not wrapped with the
// length and last \x00 from the document. wrapInDoc() computes them.
// Note that all of them should be supported as two-way conversions.

var allItems = []testItemType{
	{obj: bson.M{},
		data: ""},
	{obj: bson.M{"_": float64(5.05)},
		data: "\x01_\x00333333\x14@"},
	{obj: bson.M{"_": "yo"},
		data: "\x02_\x00\x03\x00\x00\x00yo\x00"},
	{obj: bson.M{"_": bson.M{"a": true}},
		data: "\x03_\x00\x09\x00\x00\x00\x08a\x00\x01\x00"},
	{obj: bson.M{"_": []any{true, false}},
		data: "\x04_\x00\r\x00\x00\x00\x080\x00\x01\x081\x00\x00\x00"},
	{obj: bson.M{"_": []byte("yo")},
		data: "\x05_\x00\x02\x00\x00\x00\x00yo"},
	{obj: bson.M{"_": bson.Binary{Kind: 0x80, Data: []byte("udef")}},
		data: "\x05_\x00\x04\x00\x00\x00\x80udef"},
	{obj: bson.M{"_": bson.Undefined}, // Obsolete, but still seen in the wild.
		data: "\x06_\x00"},
	{obj: bson.M{"_": bson.ObjectId("0123456789ab")},
		data: "\x07_\x000123456789ab"},
	{obj: bson.M{"_": bson.DBPointer{Namespace: "testnamespace", Id: bson.ObjectId("0123456789ab")}},
		data: "\x0C_\x00\x0e\x00\x00\x00testnamespace\x000123456789ab"},
	{obj: bson.M{"_": false},
		data: "\x08_\x00\x00"},
	{obj: bson.M{"_": true},
		data: "\x08_\x00\x01"},
	{obj: bson.M{"_": time.Unix(0, 258e6)}, // Note the NS <=> MS conversion.
		data: "\x09_\x00\x02\x01\x00\x00\x00\x00\x00\x00"},
	{obj: bson.M{"_": nil},
		data: "\x0A_\x00"},
	{obj: bson.M{"_": bson.RegEx{Pattern: "ab", Options: "cd"}},
		data: "\x0B_\x00ab\x00cd\x00"},
	{obj: bson.M{"_": bson.JavaScript{Code: "code", Scope: nil}},
		data: "\x0D_\x00\x05\x00\x00\x00code\x00"},
	{obj: bson.M{"_": bson.Symbol("sym")},
		data: "\x0E_\x00\x04\x00\x00\x00sym\x00"},
	{obj: bson.M{"_": bson.JavaScript{Code: "code", Scope: bson.M{"": nil}}},
		data: "\x0F_\x00\x14\x00\x00\x00\x05\x00\x00\x00code\x00" +
			"\x07\x00\x00\x00\x0A\x00\x00"},
	{obj: bson.M{"_": 258},
		data: "\x10_\x00\x02\x01\x00\x00"},
	{obj: bson.M{"_": bson.MongoTimestamp(258)},
		data: "\x11_\x00\x02\x01\x00\x00\x00\x00\x00\x00"},
	{obj: bson.M{"_": int64(258)},
		data: "\x12_\x00\x02\x01\x00\x00\x00\x00\x00\x00"},
	{obj: bson.M{"_": int64(258 << 32)},
		data: "\x12_\x00\x00\x00\x00\x00\x02\x01\x00\x00"},
	{obj: bson.M{"_": bson.MaxKey},
		data: "\x7F_\x00"},
	{obj: bson.M{"_": bson.MinKey},
		data: "\xFF_\x00"},
}

func (s *S) TestMarshalAllItems(c *C) {
	for i, item := range allItems {
		data, err := bson.Marshal(item.obj)
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, wrapInDoc(item.data), Commentf("Failed on item %d: %#v", i, item))
	}
}

func (s *S) TestUnmarshalAllItems(c *C) {
	for i, item := range allItems {
		value := bson.M{}
		err := bson.Unmarshal([]byte(wrapInDoc(item.data)), value)
		c.Assert(err, IsNil)
		c.Assert(value, DeepEquals, item.obj, Commentf("Failed on item %d: %#v", i, item))
	}
}

func (s *S) TestUnmarshalRawAllItems(c *C) {
	for i, item := range allItems {
		if len(item.data) == 0 {
			continue
		}
		value := item.obj.(bson.M)["_"]
		if value == nil {
			continue
		}
		pv := reflect.New(reflect.ValueOf(value).Type())
		raw := bson.Raw{Kind: item.data[0], Data: []byte(item.data[3:])}
		c.Logf("Unmarshal raw: %#v, %#v", raw, pv.Interface())
		err := raw.Unmarshal(pv.Interface())
		c.Assert(err, IsNil)
		c.Assert(pv.Elem().Interface(), DeepEquals, value, Commentf("Failed on item %d: %#v", i, item))
	}
}

func (s *S) TestUnmarshalRawIncompatible(c *C) {
	raw := bson.Raw{Kind: 0x08, Data: []byte{0x01}} // true
	err := raw.Unmarshal(&struct{}{})
	c.Assert(err, ErrorMatches, "BSON kind 0x08 isn't compatible with type struct \\{\\}")
}

func (s *S) TestUnmarshalZeroesStruct(c *C) {
	data, err := bson.Marshal(bson.M{"b": 2})
	c.Assert(err, IsNil)
	type T struct{ A, B int }
	v := T{A: 1}
	err = bson.Unmarshal(data, &v)
	c.Assert(err, IsNil)
	c.Assert(v.A, Equals, 0)
	c.Assert(v.B, Equals, 2)
}

func (s *S) TestUnmarshalZeroesMap(c *C) {
	data, err := bson.Marshal(bson.M{"b": 2})
	c.Assert(err, IsNil)
	m := bson.M{"a": 1}
	err = bson.Unmarshal(data, &m)
	c.Assert(err, IsNil)
	c.Assert(m, DeepEquals, bson.M{"b": 2})
}

func (s *S) TestUnmarshalNonNilInterface(c *C) {
	data, err := bson.Marshal(bson.M{"b": 2})
	c.Assert(err, IsNil)
	m := bson.M{"a": 1}
	var i any
	i = m
	err = bson.Unmarshal(data, &i)
	c.Assert(err, IsNil)
	c.Assert(i, DeepEquals, bson.M{"b": 2})
	c.Assert(m, DeepEquals, bson.M{"a": 1})
}

// --------------------------------------------------------------------------
// Some one way marshaling operations which would unmarshal differently.

var oneWayMarshalItems = []testItemType{
	// These are being passed as pointers, and will unmarshal as values.
	{obj: bson.M{"": &bson.Binary{Kind: 0x02, Data: []byte("old")}},
		data: "\x05\x00\x07\x00\x00\x00\x02\x03\x00\x00\x00old"},
	{obj: bson.M{"": &bson.Binary{Kind: 0x80, Data: []byte("udef")}},
		data: "\x05\x00\x04\x00\x00\x00\x80udef"},
	{obj: bson.M{"": &bson.RegEx{Pattern: "ab", Options: "cd"}},
		data: "\x0B\x00ab\x00cd\x00"},
	{obj: bson.M{"": &bson.JavaScript{Code: "code", Scope: nil}},
		data: "\x0D\x00\x05\x00\x00\x00code\x00"},
	{obj: bson.M{"": &bson.JavaScript{Code: "code", Scope: bson.M{"": nil}}},
		data: "\x0F\x00\x14\x00\x00\x00\x05\x00\x00\x00code\x00" +
			"\x07\x00\x00\x00\x0A\x00\x00"},

	// There's no float32 type in BSON.  Will encode as a float64.
	{obj: bson.M{"": float32(5.05)},
		data: "\x01\x00\x00\x00\x00@33\x14@"},

	// The array will be unmarshaled as a slice instead.
	{obj: bson.M{"": [2]bool{true, false}},
		data: "\x04\x00\r\x00\x00\x00\x080\x00\x01\x081\x00\x00\x00"},

	// The typed slice will be unmarshaled as []interface{}.
	{obj: bson.M{"": []bool{true, false}},
		data: "\x04\x00\r\x00\x00\x00\x080\x00\x01\x081\x00\x00\x00"},

	// Will unmarshal as a []byte.
	{obj: bson.M{"": bson.Binary{Kind: 0x00, Data: []byte("yo")}},
		data: "\x05\x00\x02\x00\x00\x00\x00yo"},
	{obj: bson.M{"": bson.Binary{Kind: 0x02, Data: []byte("old")}},
		data: "\x05\x00\x07\x00\x00\x00\x02\x03\x00\x00\x00old"},

	// No way to preserve the type information here. We might encode as a zero
	// value, but this would mean that pointer values in structs wouldn't be
	// able to correctly distinguish between unset and set to the zero value.
	{obj: bson.M{"": (*byte)(nil)},
		data: "\x0A\x00"},

	// No int types smaller than int32 in BSON. Could encode this as a char,
	// but it would still be ambiguous, take more, and be awkward in Go when
	// loaded without typing information.
	{obj: bson.M{"": byte(8)},
		data: "\x10\x00\x08\x00\x00\x00"},

	// There are no unsigned types in BSON.  Will unmarshal as int32 or int64.
	{obj: bson.M{"": uint32(258)},
		data: "\x10\x00\x02\x01\x00\x00"},
	{obj: bson.M{"": uint64(258)},
		data: "\x12\x00\x02\x01\x00\x00\x00\x00\x00\x00"},
	{obj: bson.M{"": uint64(258 << 32)},
		data: "\x12\x00\x00\x00\x00\x00\x02\x01\x00\x00"},

	// This will unmarshal as int.
	{obj: bson.M{"": int32(258)},
		data: "\x10\x00\x02\x01\x00\x00"},

	// That's a special case. The unsigned value is too large for an int32,
	// so an int64 is used instead.
	{obj: bson.M{"": uint32(1<<32 - 1)},
		data: "\x12\x00\xFF\xFF\xFF\xFF\x00\x00\x00\x00"},
	{obj: bson.M{"": uint(1<<32 - 1)},
		data: "\x12\x00\xFF\xFF\xFF\xFF\x00\x00\x00\x00"},
}

func (s *S) TestOneWayMarshalItems(c *C) {
	for i, item := range oneWayMarshalItems {
		data, err := bson.Marshal(item.obj)
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, wrapInDoc(item.data),
			Commentf("Failed on item %d", i))
	}
}

// --------------------------------------------------------------------------
// Two-way tests for user-defined structures using the samples
// from bsonspec.org.

type specSample1 struct {
	Hello string
}

type specSample2 struct {
	BSON []any "BSON"
}

var structSampleItems = []testItemType{
	{obj: &specSample1{Hello: "world"},
		data: "\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00"},
	{obj: &specSample2{BSON: []any{"awesome", float64(5.05), 1986}},
		data: "1\x00\x00\x00\x04BSON\x00&\x00\x00\x00\x020\x00\x08\x00\x00\x00" +
			"awesome\x00\x011\x00333333\x14@\x102\x00\xc2\x07\x00\x00\x00\x00"},
}

func (s *S) TestMarshalStructSampleItems(c *C) {
	for i, item := range structSampleItems {
		data, err := bson.Marshal(item.obj)
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, item.data,
			Commentf("Failed on item %d", i))
	}
}

func (s *S) TestUnmarshalStructSampleItems(c *C) {
	for _, item := range structSampleItems {
		testUnmarshal(c, item.data, item.obj)
	}
}

func (s *S) Test64bitInt(c *C) {
	var i int64 = (1 << 31)
	if int(i) > 0 {
		data, err := bson.Marshal(bson.M{"i": int(i)})
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, wrapInDoc("\x12i\x00\x00\x00\x00\x80\x00\x00\x00\x00"))

		var result struct{ I int }
		err = bson.Unmarshal(data, &result)
		c.Assert(err, IsNil)
		c.Assert(int64(result.I), Equals, i)
	}
}

// --------------------------------------------------------------------------
// Generic two-way struct marshaling tests.

var bytevar = byte(8)
var byteptr = &bytevar

var structItems = []testItemType{
	{obj: &struct{ Ptr *byte }{Ptr: nil},
		data: "\x0Aptr\x00"},
	{obj: &struct{ Ptr *byte }{Ptr: &bytevar},
		data: "\x10ptr\x00\x08\x00\x00\x00"},
	{obj: &struct{ Ptr **byte }{Ptr: &byteptr},
		data: "\x10ptr\x00\x08\x00\x00\x00"},
	{obj: &struct{ Byte byte }{Byte: 8},
		data: "\x10byte\x00\x08\x00\x00\x00"},
	{obj: &struct{ Byte byte }{Byte: 0},
		data: "\x10byte\x00\x00\x00\x00\x00"},
	{obj: &struct {
		V byte "Tag"
	}{V: 8},
		data: "\x10Tag\x00\x08\x00\x00\x00"},
	{obj: &struct {
		V *struct {
			Byte byte
		}
	}{V: &struct{ Byte byte }{Byte: 8}},
		data: "\x03v\x00" + "\x0f\x00\x00\x00\x10byte\x00\b\x00\x00\x00\x00"},
	{obj: &struct{ priv byte }{}, data: ""},

	// The order of the dumped fields should be the same in the struct.
	{obj: &struct{ A, C, B, D, F, E *byte }{},
		data: "\x0Aa\x00\x0Ac\x00\x0Ab\x00\x0Ad\x00\x0Af\x00\x0Ae\x00"},

	{obj: &struct{ V bson.Raw }{V: bson.Raw{Kind: 0x03, Data: []byte("\x0f\x00\x00\x00\x10byte\x00\b\x00\x00\x00\x00")}},
		data: "\x03v\x00" + "\x0f\x00\x00\x00\x10byte\x00\b\x00\x00\x00\x00"},
	{obj: &struct{ V bson.Raw }{V: bson.Raw{Kind: 0x10, Data: []byte("\x00\x00\x00\x00")}},
		data: "\x10v\x00" + "\x00\x00\x00\x00"},

	// Byte arrays.
	{obj: &struct{ V [2]byte }{V: [2]byte{'y', 'o'}},
		data: "\x05v\x00\x02\x00\x00\x00\x00yo"},
}

func (s *S) TestMarshalStructItems(c *C) {
	for i, item := range structItems {
		data, err := bson.Marshal(item.obj)
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, wrapInDoc(item.data),
			Commentf("Failed on item %d", i))
	}
}

func (s *S) TestUnmarshalStructItems(c *C) {
	for _, item := range structItems {
		testUnmarshal(c, wrapInDoc(item.data), item.obj)
	}
}

func (s *S) TestUnmarshalRawStructItems(c *C) {
	for i, item := range structItems {
		raw := bson.Raw{Kind: 0x03, Data: []byte(wrapInDoc(item.data))}
		zero := makeZeroDoc(item.obj)
		err := raw.Unmarshal(zero)
		c.Assert(err, IsNil)
		c.Assert(zero, DeepEquals, item.obj, Commentf("Failed on item %d: %#v", i, item))
	}
}

func (s *S) TestUnmarshalRawNil(c *C) {
	// Regression test: shouldn't try to nil out the pointer itself,
	// as it's not settable.
	raw := bson.Raw{Kind: 0x0A, Data: []byte{}}
	err := raw.Unmarshal(&struct{}{})
	c.Assert(err, IsNil)
}

// --------------------------------------------------------------------------
// One-way marshaling tests.

type dOnIface struct {
	D any
}

type ignoreField struct {
	Before string
	Ignore string `bson:"-"`
	After  string
}

var marshalItems = []testItemType{
	// Ordered document dump.  Will unmarshal as a dictionary by default.
	{obj: bson.D{{Name: "a", Value: nil}, {Name: "c", Value: nil}, {Name: "b", Value: nil}, {Name: "d", Value: nil}, {Name: "f", Value: nil}, {Name: "e", Value: true}},
		data: "\x0Aa\x00\x0Ac\x00\x0Ab\x00\x0Ad\x00\x0Af\x00\x08e\x00\x01"},
	{obj: MyD{{Name: "a", Value: nil}, {Name: "c", Value: nil}, {Name: "b", Value: nil}, {Name: "d", Value: nil}, {Name: "f", Value: nil}, {Name: "e", Value: true}},
		data: "\x0Aa\x00\x0Ac\x00\x0Ab\x00\x0Ad\x00\x0Af\x00\x08e\x00\x01"},
	{obj: &dOnIface{D: bson.D{{Name: "a", Value: nil}, {Name: "c", Value: nil}, {Name: "b", Value: nil}, {Name: "d", Value: true}}},
		data: "\x03d\x00" + wrapInDoc("\x0Aa\x00\x0Ac\x00\x0Ab\x00\x08d\x00\x01")},

	{obj: bson.RawD{{Name: "a", Value: bson.Raw{Kind: 0x0A, Data: nil}}, {Name: "c", Value: bson.Raw{Kind: 0x0A, Data: nil}}, {Name: "b", Value: bson.Raw{Kind: 0x08, Data: []byte{0x01}}}},
		data: "\x0Aa\x00" + "\x0Ac\x00" + "\x08b\x00\x01"},
	{obj: MyRawD{{Name: "a", Value: bson.Raw{Kind: 0x0A, Data: nil}}, {Name: "c", Value: bson.Raw{Kind: 0x0A, Data: nil}}, {Name: "b", Value: bson.Raw{Kind: 0x08, Data: []byte{0x01}}}},
		data: "\x0Aa\x00" + "\x0Ac\x00" + "\x08b\x00\x01"},
	{obj: &dOnIface{D: bson.RawD{{Name: "a", Value: bson.Raw{Kind: 0x0A, Data: nil}}, {Name: "c", Value: bson.Raw{Kind: 0x0A, Data: nil}}, {Name: "b", Value: bson.Raw{Kind: 0x08, Data: []byte{0x01}}}}},
		data: "\x03d\x00" + wrapInDoc("\x0Aa\x00"+"\x0Ac\x00"+"\x08b\x00\x01")},

	{obj: &ignoreField{Before: "before", Ignore: "ignore", After: "after"},
		data: "\x02before\x00\a\x00\x00\x00before\x00\x02after\x00\x06\x00\x00\x00after\x00"},

	// Marshalling a Raw document does nothing.
	{obj: bson.Raw{Kind: 0x03, Data: []byte(wrapInDoc("anything"))},
		data: "anything"},
	{obj: bson.Raw{Data: []byte(wrapInDoc("anything"))},
		data: "anything"},
}

func (s *S) TestMarshalOneWayItems(c *C) {
	for _, item := range marshalItems {
		data, err := bson.Marshal(item.obj)
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, wrapInDoc(item.data))
	}
}

// --------------------------------------------------------------------------
// One-way unmarshaling tests.

var unmarshalItems = []testItemType{
	// Field is private.  Should not attempt to unmarshal it.
	{obj: &struct{ priv byte }{},
		data: "\x10priv\x00\x08\x00\x00\x00"},

	// Wrong casing. Field names are lowercased.
	{obj: &struct{ Byte byte }{},
		data: "\x10Byte\x00\x08\x00\x00\x00"},

	// Ignore non-existing field.
	{obj: &struct{ Byte byte }{Byte: 9},
		data: "\x10boot\x00\x08\x00\x00\x00" + "\x10byte\x00\x09\x00\x00\x00"},

	// Do not unmarshal on ignored field.
	{obj: &ignoreField{Before: "before", Ignore: "", After: "after"},
		data: "\x02before\x00\a\x00\x00\x00before\x00" +
			"\x02-\x00\a\x00\x00\x00ignore\x00" +
			"\x02after\x00\x06\x00\x00\x00after\x00"},

	// Ignore unsuitable types silently.
	{obj: map[string]string{"str": "s"},
		data: "\x02str\x00\x02\x00\x00\x00s\x00" + "\x10int\x00\x01\x00\x00\x00"},
	{obj: map[string][]int{"array": {5, 9}},
		data: "\x04array\x00" + wrapInDoc("\x100\x00\x05\x00\x00\x00"+"\x021\x00\x02\x00\x00\x00s\x00"+"\x102\x00\x09\x00\x00\x00")},

	// Wrong type. Shouldn't init pointer.
	{obj: &struct{ Str *byte }{},
		data: "\x02str\x00\x02\x00\x00\x00s\x00"},
	{obj: &struct{ Str *struct{ Str string } }{},
		data: "\x02str\x00\x02\x00\x00\x00s\x00"},

	// Ordered document.
	{obj: &struct{ bson.D }{D: bson.D{{Name: "a", Value: nil}, {Name: "c", Value: nil}, {Name: "b", Value: nil}, {Name: "d", Value: true}}},
		data: "\x03d\x00" + wrapInDoc("\x0Aa\x00\x0Ac\x00\x0Ab\x00\x08d\x00\x01")},

	// Raw document.
	{obj: &bson.Raw{Kind: 0x03, Data: []byte(wrapInDoc("\x10byte\x00\x08\x00\x00\x00"))},
		data: "\x10byte\x00\x08\x00\x00\x00"},

	// RawD document.
	{obj: &struct{ bson.RawD }{RawD: bson.RawD{{Name: "a", Value: bson.Raw{Kind: 0x0A, Data: []byte{}}}, {Name: "c", Value: bson.Raw{Kind: 0x0A, Data: []byte{}}}, {Name: "b", Value: bson.Raw{Kind: 0x08, Data: []byte{0x01}}}}},
		data: "\x03rawd\x00" + wrapInDoc("\x0Aa\x00\x0Ac\x00\x08b\x00\x01")},

	// Decode old binary.
	{obj: bson.M{"_": []byte("old")},
		data: "\x05_\x00\x07\x00\x00\x00\x02\x03\x00\x00\x00old"},

	// Decode old binary without length. According to the spec, this shouldn't happen.
	{obj: bson.M{"_": []byte("old")},
		data: "\x05_\x00\x03\x00\x00\x00\x02old"},

	// Decode a doc within a doc in to a slice within a doc; shouldn't error
	{obj: &struct{ Foo []string }{},
		data: "\x03\x66\x6f\x6f\x00\x05\x00\x00\x00\x00"},
}

func (s *S) TestUnmarshalOneWayItems(c *C) {
	for _, item := range unmarshalItems {
		testUnmarshal(c, wrapInDoc(item.data), item.obj)
	}
}

func (s *S) TestUnmarshalNilInStruct(c *C) {
	// Nil is the default value, so we need to ensure it's indeed being set.
	b := byte(1)
	v := &struct{ Ptr *byte }{Ptr: &b}
	err := bson.Unmarshal([]byte(wrapInDoc("\x0Aptr\x00")), v)
	c.Assert(err, IsNil)
	c.Assert(v, DeepEquals, &struct{ Ptr *byte }{Ptr: nil})
}

// --------------------------------------------------------------------------
// Marshalling error cases.

type structWithDupKeys struct {
	Name  byte
	Other byte "name" // Tag should precede.
}

var marshalErrorItems = []testItemType{
	{obj: bson.M{"": uint64(1 << 63)},
		data: "BSON has no uint64 type, and value is too large to fit correctly in an int64"},
	{obj: bson.M{"": bson.ObjectId("tooshort")},
		data: "ObjectIDs must be exactly 12 bytes long \\(got 8\\)"},
	{obj: int64(123),
		data: "Can't marshal int64 as a BSON document"},
	{obj: bson.M{"": 1i},
		data: "Can't marshal complex128 in a BSON document"},
	{obj: &structWithDupKeys{},
		data: "Duplicated key 'name' in struct bson_test.structWithDupKeys"},
	{obj: bson.Raw{Kind: 0xA, Data: []byte{}},
		data: "Attempted to marshal Raw kind 10 as a document"},
	{obj: bson.Raw{Kind: 0x3, Data: []byte{}},
		data: "Attempted to marshal empty Raw document"},
	{obj: bson.M{"w": bson.Raw{Kind: 0x3, Data: []byte{}}},
		data: "Attempted to marshal empty Raw document"},
	{obj: &inlineCantPtr{V: &struct{ A, B int }{A: 1, B: 2}},
		data: "Option ,inline needs a struct value or map field"},
	{obj: &inlineDupName{A: 1, V: struct{ A, B int }{A: 2, B: 3}},
		data: "Duplicated key 'a' in struct bson_test.inlineDupName"},
	{obj: &inlineDupMap{},
		data: "Multiple ,inline maps in struct bson_test.inlineDupMap"},
	{obj: &inlineBadKeyMap{},
		data: "Option ,inline needs a map with string keys in struct bson_test.inlineBadKeyMap"},
	{obj: &inlineMap{A: 1, M: map[string]any{"a": 1}},
		data: `Can't have key "a" in inlined map; conflicts with struct field`},
}

func (s *S) TestMarshalErrorItems(c *C) {
	for _, item := range marshalErrorItems {
		data, err := bson.Marshal(item.obj)
		c.Assert(err, ErrorMatches, item.data)
		c.Assert(data, IsNil)
	}
}

// --------------------------------------------------------------------------
// Unmarshalling error cases.

type unmarshalErrorType struct {
	obj   any
	data  string
	error string
}

var unmarshalErrorItems = []unmarshalErrorType{
	// Tag name conflicts with existing parameter.
	{obj: &structWithDupKeys{},
		data:  "\x10name\x00\x08\x00\x00\x00",
		error: "Duplicated key 'name' in struct bson_test.structWithDupKeys"},

	// Non-string map key.
	{obj: map[int]any{},
		data:  "\x10name\x00\x08\x00\x00\x00",
		error: "BSON map must have string keys. Got: map\\[int\\]interface \\{\\}"},

	{obj: nil,
		data:  "\xEEname\x00",
		error: "Unknown element kind \\(0xEE\\)"},

	{obj: struct{ Name bool }{},
		data:  "\x10name\x00\x08\x00\x00\x00",
		error: "Unmarshal can't deal with struct values. Use a pointer."},

	{obj: 123,
		data:  "\x10name\x00\x08\x00\x00\x00",
		error: "Unmarshal needs a map or a pointer to a struct."},

	{obj: nil,
		data:  "\x08\x62\x00\x02",
		error: "encoded boolean must be 1 or 0, found 2"},
}

func (s *S) TestUnmarshalErrorItems(c *C) {
	for _, item := range unmarshalErrorItems {
		data := []byte(wrapInDoc(item.data))
		var value any
		switch reflect.ValueOf(item.obj).Kind() {
		case reflect.Map, reflect.Ptr:
			value = makeZeroDoc(item.obj)
		case reflect.Invalid:
			value = bson.M{}
		default:
			value = item.obj
		}
		err := bson.Unmarshal(data, value)
		c.Assert(err, ErrorMatches, item.error)
	}
}

type unmarshalRawErrorType struct {
	obj   any
	raw   bson.Raw
	error string
}

var unmarshalRawErrorItems = []unmarshalRawErrorType{
	// Tag name conflicts with existing parameter.
	{obj: &structWithDupKeys{},
		raw:   bson.Raw{Kind: 0x03, Data: []byte("\x10byte\x00\x08\x00\x00\x00")},
		error: "Duplicated key 'name' in struct bson_test.structWithDupKeys"},

	{obj: &struct{}{},
		raw:   bson.Raw{Kind: 0xEE, Data: []byte{}},
		error: "Unknown element kind \\(0xEE\\)"},

	{obj: struct{ Name bool }{},
		raw:   bson.Raw{Kind: 0x10, Data: []byte("\x08\x00\x00\x00")},
		error: "Raw Unmarshal can't deal with struct values. Use a pointer."},

	{obj: 123,
		raw:   bson.Raw{Kind: 0x10, Data: []byte("\x08\x00\x00\x00")},
		error: "Raw Unmarshal needs a map or a valid pointer."},
}

func (s *S) TestUnmarshalRawErrorItems(c *C) {
	for i, item := range unmarshalRawErrorItems {
		err := item.raw.Unmarshal(item.obj)
		c.Assert(err, ErrorMatches, item.error, Commentf("Failed on item %d: %#v\n", i, item))
	}
}

var corruptedData = []string{
	"\x04\x00\x00\x00\x00",         // Document shorter than minimum
	"\x06\x00\x00\x00\x00",         // Not enough data
	"\x05\x00\x00",                 // Broken length
	"\x05\x00\x00\x00\xff",         // Corrupted termination
	"\x0A\x00\x00\x00\x0Aooop\x00", // Unfinished C string

	// Array end past end of string (s[2]=0x07 is correct)
	wrapInDoc("\x04\x00\x09\x00\x00\x00\x0A\x00\x00"),

	// Array end within string, but past acceptable.
	wrapInDoc("\x04\x00\x08\x00\x00\x00\x0A\x00\x00"),

	// Document end within string, but past acceptable.
	wrapInDoc("\x03\x00\x08\x00\x00\x00\x0A\x00\x00"),

	// String with corrupted end.
	wrapInDoc("\x02\x00\x03\x00\x00\x00yo\xFF"),

	// String with negative length (issue #116).
	"\x0c\x00\x00\x00\x02x\x00\xff\xff\xff\xff\x00",

	// String with zero length (must include trailing '\x00')
	"\x0c\x00\x00\x00\x02x\x00\x00\x00\x00\x00\x00",

	// Binary with negative length.
	"\r\x00\x00\x00\x05x\x00\xff\xff\xff\xff\x00\x00",
}

func (s *S) TestUnmarshalMapDocumentTooShort(c *C) {
	for _, data := range corruptedData {
		err := bson.Unmarshal([]byte(data), bson.M{})
		c.Assert(err, ErrorMatches, "Document is corrupted")

		err = bson.Unmarshal([]byte(data), &struct{}{})
		c.Assert(err, ErrorMatches, "Document is corrupted")
	}
}

// --------------------------------------------------------------------------
// Setter test cases.

var setterResult = map[string]error{}

type setterType struct {
	received any
}

func (o *setterType) SetBSON(raw bson.Raw) error {
	err := raw.Unmarshal(&o.received)
	if err != nil {
		panic("The panic:" + err.Error())
	}
	if s, ok := o.received.(string); ok {
		if result, ok := setterResult[s]; ok {
			return result
		}
	}
	return nil
}

type ptrSetterDoc struct {
	Field *setterType "_"
}

type valSetterDoc struct {
	Field setterType "_"
}

func (s *S) TestUnmarshalAllItemsWithPtrSetter(c *C) {
	for _, item := range allItems {
		for i := 0; i != 2; i++ {
			var field *setterType
			if i == 0 {
				obj := &ptrSetterDoc{}
				err := bson.Unmarshal([]byte(wrapInDoc(item.data)), obj)
				c.Assert(err, IsNil)
				field = obj.Field
			} else {
				obj := &valSetterDoc{}
				err := bson.Unmarshal([]byte(wrapInDoc(item.data)), obj)
				c.Assert(err, IsNil)
				field = &obj.Field
			}
			if item.data == "" {
				// Nothing to unmarshal. Should be untouched.
				if i == 0 {
					c.Assert(field, IsNil)
				} else {
					c.Assert(field.received, IsNil)
				}
			} else {
				expected := item.obj.(bson.M)["_"]
				c.Assert(field, NotNil, Commentf("Pointer not initialized (%#v)", expected))
				c.Assert(field.received, DeepEquals, expected)
			}
		}
	}
}

func (s *S) TestUnmarshalWholeDocumentWithSetter(c *C) {
	obj := &setterType{}
	err := bson.Unmarshal([]byte(sampleItems[0].data), obj)
	c.Assert(err, IsNil)
	c.Assert(obj.received, DeepEquals, bson.M{"hello": "world"})
}

func (s *S) TestUnmarshalSetterOmits(c *C) {
	setterResult["2"] = &bson.TypeError{}
	setterResult["4"] = &bson.TypeError{}
	defer func() {
		delete(setterResult, "2")
		delete(setterResult, "4")
	}()

	m := map[string]*setterType{}
	data := wrapInDoc("\x02abc\x00\x02\x00\x00\x001\x00" +
		"\x02def\x00\x02\x00\x00\x002\x00" +
		"\x02ghi\x00\x02\x00\x00\x003\x00" +
		"\x02jkl\x00\x02\x00\x00\x004\x00")
	err := bson.Unmarshal([]byte(data), m)
	c.Assert(err, IsNil)
	c.Assert(m["abc"], NotNil)
	c.Assert(m["def"], IsNil)
	c.Assert(m["ghi"], NotNil)
	c.Assert(m["jkl"], IsNil)

	c.Assert(m["abc"].received, Equals, "1")
	c.Assert(m["ghi"].received, Equals, "3")
}

func (s *S) TestUnmarshalSetterErrors(c *C) {
	boom := errors.New("BOOM")
	setterResult["2"] = boom
	defer delete(setterResult, "2")

	m := map[string]*setterType{}
	data := wrapInDoc("\x02abc\x00\x02\x00\x00\x001\x00" +
		"\x02def\x00\x02\x00\x00\x002\x00" +
		"\x02ghi\x00\x02\x00\x00\x003\x00")
	err := bson.Unmarshal([]byte(data), m)
	c.Assert(err, Equals, boom)
	c.Assert(m["abc"], NotNil)
	c.Assert(m["def"], IsNil)
	c.Assert(m["ghi"], IsNil)

	c.Assert(m["abc"].received, Equals, "1")
}

func (s *S) TestDMap(c *C) {
	d := bson.D{{Name: "a", Value: 1}, {Name: "b", Value: 2}}
	c.Assert(d.Map(), DeepEquals, bson.M{"a": 1, "b": 2})
}

func (s *S) TestUnmarshalSetterSetZero(c *C) {
	setterResult["foo"] = bson.SetZero
	defer delete(setterResult, "field")

	data, err := bson.Marshal(bson.M{"field": "foo"})
	c.Assert(err, IsNil)

	m := map[string]*setterType{}
	err = bson.Unmarshal([]byte(data), m)
	c.Assert(err, IsNil)

	value, ok := m["field"]
	c.Assert(ok, Equals, true)
	c.Assert(value, IsNil)
}

// --------------------------------------------------------------------------
// Getter test cases.

type typeWithGetter struct {
	result any
	err    error
}

func (t *typeWithGetter) GetBSON() (any, error) {
	if t == nil {
		return "<value is nil>", nil
	}
	return t.result, t.err
}

type docWithGetterField struct {
	Field *typeWithGetter "_"
}

func (s *S) TestMarshalAllItemsWithGetter(c *C) {
	for i, item := range allItems {
		if item.data == "" {
			continue
		}
		obj := &docWithGetterField{}
		obj.Field = &typeWithGetter{result: item.obj.(bson.M)["_"]}
		data, err := bson.Marshal(obj)
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, wrapInDoc(item.data),
			Commentf("Failed on item #%d", i))
	}
}

func (s *S) TestMarshalWholeDocumentWithGetter(c *C) {
	obj := &typeWithGetter{result: sampleItems[0].obj}
	data, err := bson.Marshal(obj)
	c.Assert(err, IsNil)
	c.Assert(string(data), Equals, sampleItems[0].data)
}

func (s *S) TestGetterErrors(c *C) {
	e := errors.New("oops")

	obj1 := &docWithGetterField{}
	obj1.Field = &typeWithGetter{result: sampleItems[0].obj, err: e}
	data, err := bson.Marshal(obj1)
	c.Assert(err, ErrorMatches, "oops")
	c.Assert(data, IsNil)

	obj2 := &typeWithGetter{result: sampleItems[0].obj, err: e}
	data, err = bson.Marshal(obj2)
	c.Assert(err, ErrorMatches, "oops")
	c.Assert(data, IsNil)
}

type intGetter int64

func (t intGetter) GetBSON() (any, error) {
	return int64(t), nil
}

type typeWithIntGetter struct {
	V intGetter ",minsize"
}

func (s *S) TestMarshalShortWithGetter(c *C) {
	obj := typeWithIntGetter{V: 42}
	data, err := bson.Marshal(obj)
	c.Assert(err, IsNil)
	m := bson.M{}
	err = bson.Unmarshal(data, m)
	c.Assert(err, IsNil)
	c.Assert(m["v"], Equals, 42)
}

func (s *S) TestMarshalWithGetterNil(c *C) {
	obj := docWithGetterField{}
	data, err := bson.Marshal(obj)
	c.Assert(err, IsNil)
	m := bson.M{}
	err = bson.Unmarshal(data, m)
	c.Assert(err, IsNil)
	c.Assert(m, DeepEquals, bson.M{"_": "<value is nil>"})
}

// --------------------------------------------------------------------------
// Cross-type conversion tests.

type crossTypeItem struct {
	obj1 any
	obj2 any
}

type condStr struct {
	V string ",omitempty"
}

type condStrNS struct {
	V string `a:"A" bson:",omitempty" b:"B"`
}

type condBool struct {
	V bool ",omitempty"
}

type condInt struct {
	V int ",omitempty"
}

type condUInt struct {
	V uint ",omitempty"
}

type condFloat struct {
	V float64 ",omitempty"
}

type condIface struct {
	V any ",omitempty"
}

type condPtr struct {
	V *bool ",omitempty"
}

type condSlice struct {
	V []string ",omitempty"
}

type condMap struct {
	V map[string]int ",omitempty"
}

type namedCondStr struct {
	V string "myv,omitempty"
}

type condTime struct {
	V time.Time ",omitempty"
}

type condStruct struct {
	V struct{ A []int } ",omitempty"
}

type condRaw struct {
	V bson.Raw ",omitempty"
}

type shortInt struct {
	V int64 ",minsize"
}

type shortUint struct {
	V uint64 ",minsize"
}

type shortIface struct {
	V any ",minsize"
}

type shortPtr struct {
	V *int64 ",minsize"
}

type shortNonEmptyInt struct {
	V int64 ",minsize,omitempty"
}

type inlineInt struct {
	V struct{ A, B int } ",inline"
}

type inlineCantPtr struct {
	V *struct{ A, B int } ",inline"
}

type inlineDupName struct {
	A int
	V struct{ A, B int } ",inline"
}

type inlineMap struct {
	A int
	M map[string]any ",inline"
}

type inlineMapInt struct {
	A int
	M map[string]int ",inline"
}

type inlineMapMyM struct {
	A int
	M MyM ",inline"
}

type inlineDupMap struct {
	M1 map[string]any ",inline"
	M2 map[string]any ",inline"
}

type inlineBadKeyMap struct {
	M map[int]int ",inline"
}

type inlineUnexported struct {
	M          map[string]any ",inline"
	unexported ",inline"
}

type unexported struct {
	A int
}

type getterSetterD bson.D

func (s getterSetterD) GetBSON() (any, error) {
	if len(s) == 0 {
		return bson.D{}, nil
	}
	return bson.D(s[:len(s)-1]), nil
}

func (s *getterSetterD) SetBSON(raw bson.Raw) error {
	var doc bson.D
	err := raw.Unmarshal(&doc)
	doc = append(doc, bson.DocElem{Name: "suffix", Value: true})
	*s = getterSetterD(doc)
	return err
}

type getterSetterInt int

func (i getterSetterInt) GetBSON() (any, error) {
	return bson.D{{Name: "a", Value: int(i)}}, nil
}

func (i *getterSetterInt) SetBSON(raw bson.Raw) error {
	var doc struct{ A int }
	err := raw.Unmarshal(&doc)
	*i = getterSetterInt(doc.A)
	return err
}

type ifaceType interface {
	Hello()
}

type ifaceSlice []ifaceType

func (s *ifaceSlice) SetBSON(raw bson.Raw) error {
	var ns []int
	if err := raw.Unmarshal(&ns); err != nil {
		return err
	}
	*s = make(ifaceSlice, ns[0])
	return nil
}

func (s ifaceSlice) GetBSON() (any, error) {
	return []int{len(s)}, nil
}

type (
	MyString string

	MyBytes []byte

	MyBool bool

	MyD []bson.DocElem

	MyRawD []bson.RawDocElem

	MyM map[string]any
)

var (
	truevar  = true
	falsevar = false

	int64var = int64(42)
	int64ptr = &int64var
	intvar   = int(42)
	intptr   = &intvar

	gsintvar = getterSetterInt(42)
)

func parseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// That's a pretty fun test.  It will dump the first item, generate a zero
// value equivalent to the second one, load the dumped data onto it, and then
// verify that the resulting value is deep-equal to the untouched second value.
// Then, it will do the same in the *opposite* direction!
var twoWayCrossItems = []crossTypeItem{
	// int<=>int
	{obj1: &struct{ I int }{I: 42}, obj2: &struct{ I int8 }{I: 42}},
	{obj1: &struct{ I int }{I: 42}, obj2: &struct{ I int32 }{I: 42}},
	{obj1: &struct{ I int }{I: 42}, obj2: &struct{ I int64 }{I: 42}},
	{obj1: &struct{ I int8 }{I: 42}, obj2: &struct{ I int32 }{I: 42}},
	{obj1: &struct{ I int8 }{I: 42}, obj2: &struct{ I int64 }{I: 42}},
	{obj1: &struct{ I int32 }{I: 42}, obj2: &struct{ I int64 }{I: 42}},

	// uint<=>uint
	{obj1: &struct{ I uint }{I: 42}, obj2: &struct{ I uint8 }{I: 42}},
	{obj1: &struct{ I uint }{I: 42}, obj2: &struct{ I uint32 }{I: 42}},
	{obj1: &struct{ I uint }{I: 42}, obj2: &struct{ I uint64 }{I: 42}},
	{obj1: &struct{ I uint8 }{I: 42}, obj2: &struct{ I uint32 }{I: 42}},
	{obj1: &struct{ I uint8 }{I: 42}, obj2: &struct{ I uint64 }{I: 42}},
	{obj1: &struct{ I uint32 }{I: 42}, obj2: &struct{ I uint64 }{I: 42}},

	// float32<=>float64
	{obj1: &struct{ I float32 }{I: 42}, obj2: &struct{ I float64 }{I: 42}},

	// int<=>uint
	{obj1: &struct{ I uint }{I: 42}, obj2: &struct{ I int }{I: 42}},
	{obj1: &struct{ I uint }{I: 42}, obj2: &struct{ I int8 }{I: 42}},
	{obj1: &struct{ I uint }{I: 42}, obj2: &struct{ I int32 }{I: 42}},
	{obj1: &struct{ I uint }{I: 42}, obj2: &struct{ I int64 }{I: 42}},
	{obj1: &struct{ I uint8 }{I: 42}, obj2: &struct{ I int }{I: 42}},
	{obj1: &struct{ I uint8 }{I: 42}, obj2: &struct{ I int8 }{I: 42}},
	{obj1: &struct{ I uint8 }{I: 42}, obj2: &struct{ I int32 }{I: 42}},
	{obj1: &struct{ I uint8 }{I: 42}, obj2: &struct{ I int64 }{I: 42}},
	{obj1: &struct{ I uint32 }{I: 42}, obj2: &struct{ I int }{I: 42}},
	{obj1: &struct{ I uint32 }{I: 42}, obj2: &struct{ I int8 }{I: 42}},
	{obj1: &struct{ I uint32 }{I: 42}, obj2: &struct{ I int32 }{I: 42}},
	{obj1: &struct{ I uint32 }{I: 42}, obj2: &struct{ I int64 }{I: 42}},
	{obj1: &struct{ I uint64 }{I: 42}, obj2: &struct{ I int }{I: 42}},
	{obj1: &struct{ I uint64 }{I: 42}, obj2: &struct{ I int8 }{I: 42}},
	{obj1: &struct{ I uint64 }{I: 42}, obj2: &struct{ I int32 }{I: 42}},
	{obj1: &struct{ I uint64 }{I: 42}, obj2: &struct{ I int64 }{I: 42}},

	// int <=> float
	{obj1: &struct{ I int }{I: 42}, obj2: &struct{ I float64 }{I: 42}},

	// int <=> bool
	{obj1: &struct{ I int }{I: 1}, obj2: &struct{ I bool }{I: true}},
	{obj1: &struct{ I int }{I: 0}, obj2: &struct{ I bool }{I: false}},

	// uint <=> float64
	{obj1: &struct{ I uint }{I: 42}, obj2: &struct{ I float64 }{I: 42}},

	// uint <=> bool
	{obj1: &struct{ I uint }{I: 1}, obj2: &struct{ I bool }{I: true}},
	{obj1: &struct{ I uint }{I: 0}, obj2: &struct{ I bool }{I: false}},

	// float64 <=> bool
	{obj1: &struct{ I float64 }{I: 1}, obj2: &struct{ I bool }{I: true}},
	{obj1: &struct{ I float64 }{I: 0}, obj2: &struct{ I bool }{I: false}},

	// string <=> string and string <=> []byte
	{obj1: &struct{ S []byte }{S: []byte("abc")}, obj2: &struct{ S string }{S: "abc"}},
	{obj1: &struct{ S []byte }{S: []byte("def")}, obj2: &struct{ S bson.Symbol }{S: "def"}},
	{obj1: &struct{ S string }{S: "ghi"}, obj2: &struct{ S bson.Symbol }{S: "ghi"}},

	// map <=> struct
	{obj1: &struct {
		A struct {
			B, C int
		}
	}{A: struct{ B, C int }{B: 1, C: 2}},
		obj2: map[string]map[string]int{"a": {"b": 1, "c": 2}}},

	{obj1: &struct{ A bson.Symbol }{A: "abc"}, obj2: map[string]string{"a": "abc"}},
	{obj1: &struct{ A bson.Symbol }{A: "abc"}, obj2: map[string][]byte{"a": []byte("abc")}},
	{obj1: &struct{ A []byte }{A: []byte("abc")}, obj2: map[string]string{"a": "abc"}},
	{obj1: &struct{ A uint }{A: 42}, obj2: map[string]int{"a": 42}},
	{obj1: &struct{ A uint }{A: 42}, obj2: map[string]float64{"a": 42}},
	{obj1: &struct{ A uint }{A: 1}, obj2: map[string]bool{"a": true}},
	{obj1: &struct{ A int }{A: 42}, obj2: map[string]uint{"a": 42}},
	{obj1: &struct{ A int }{A: 42}, obj2: map[string]float64{"a": 42}},
	{obj1: &struct{ A int }{A: 1}, obj2: map[string]bool{"a": true}},
	{obj1: &struct{ A float64 }{A: 42}, obj2: map[string]float32{"a": 42}},
	{obj1: &struct{ A float64 }{A: 42}, obj2: map[string]int{"a": 42}},
	{obj1: &struct{ A float64 }{A: 42}, obj2: map[string]uint{"a": 42}},
	{obj1: &struct{ A float64 }{A: 1}, obj2: map[string]bool{"a": true}},
	{obj1: &struct{ A bool }{A: true}, obj2: map[string]int{"a": 1}},
	{obj1: &struct{ A bool }{A: true}, obj2: map[string]uint{"a": 1}},
	{obj1: &struct{ A bool }{A: true}, obj2: map[string]float64{"a": 1}},
	{obj1: &struct{ A **byte }{A: &byteptr}, obj2: map[string]byte{"a": 8}},

	// url.URL <=> string
	{obj1: &struct{ URL *url.URL }{URL: parseURL("h://e.c/p")}, obj2: map[string]string{"url": "h://e.c/p"}},
	{obj1: &struct{ URL url.URL }{URL: *parseURL("h://e.c/p")}, obj2: map[string]string{"url": "h://e.c/p"}},

	// Slices
	{obj1: &struct{ S []int }{S: []int{1, 2, 3}}, obj2: map[string][]int{"s": {1, 2, 3}}},
	{obj1: &struct{ S *[]int }{S: &[]int{1, 2, 3}}, obj2: map[string][]int{"s": {1, 2, 3}}},

	// Conditionals
	{obj1: &condBool{V: true}, obj2: map[string]bool{"v": true}},
	{obj1: &condBool{}, obj2: map[string]bool{}},
	{obj1: &condInt{V: 1}, obj2: map[string]int{"v": 1}},
	{obj1: &condInt{}, obj2: map[string]int{}},
	{obj1: &condUInt{V: 1}, obj2: map[string]uint{"v": 1}},
	{obj1: &condUInt{}, obj2: map[string]uint{}},
	{obj1: &condFloat{}, obj2: map[string]int{}},
	{obj1: &condStr{V: "yo"}, obj2: map[string]string{"v": "yo"}},
	{obj1: &condStr{}, obj2: map[string]string{}},
	{obj1: &condStrNS{V: "yo"}, obj2: map[string]string{"v": "yo"}},
	{obj1: &condStrNS{}, obj2: map[string]string{}},
	{obj1: &condSlice{V: []string{"yo"}}, obj2: map[string][]string{"v": {"yo"}}},
	{obj1: &condSlice{}, obj2: map[string][]string{}},
	{obj1: &condMap{V: map[string]int{"k": 1}}, obj2: bson.M{"v": bson.M{"k": 1}}},
	{obj1: &condMap{}, obj2: map[string][]string{}},
	{obj1: &condIface{V: "yo"}, obj2: map[string]string{"v": "yo"}},
	{obj1: &condIface{V: ""}, obj2: map[string]string{"v": ""}},
	{obj1: &condIface{}, obj2: map[string]string{}},
	{obj1: &condPtr{V: &truevar}, obj2: map[string]bool{"v": true}},
	{obj1: &condPtr{V: &falsevar}, obj2: map[string]bool{"v": false}},
	{obj1: &condPtr{}, obj2: map[string]string{}},

	{obj1: &condTime{V: time.Unix(123456789, 123e6)}, obj2: map[string]time.Time{"v": time.Unix(123456789, 123e6)}},
	{obj1: &condTime{}, obj2: map[string]string{}},

	{obj1: &condStruct{V: struct{ A []int }{A: []int{1}}}, obj2: bson.M{"v": bson.M{"a": []any{1}}}},
	{obj1: &condStruct{V: struct{ A []int }{}}, obj2: bson.M{}},

	{obj1: &condRaw{V: bson.Raw{Kind: 0x0A, Data: []byte{}}}, obj2: bson.M{"v": nil}},
	{obj1: &condRaw{V: bson.Raw{Kind: 0x00}}, obj2: bson.M{}},

	{obj1: &namedCondStr{V: "yo"}, obj2: map[string]string{"myv": "yo"}},
	{obj1: &namedCondStr{}, obj2: map[string]string{}},

	{obj1: &shortInt{V: 1}, obj2: map[string]any{"v": 1}},
	{obj1: &shortInt{V: 1 << 30}, obj2: map[string]any{"v": 1 << 30}},
	{obj1: &shortInt{V: 1 << 31}, obj2: map[string]any{"v": int64(1 << 31)}},
	{obj1: &shortUint{V: 1 << 30}, obj2: map[string]any{"v": 1 << 30}},
	{obj1: &shortUint{V: 1 << 31}, obj2: map[string]any{"v": int64(1 << 31)}},
	{obj1: &shortIface{V: int64(1) << 31}, obj2: map[string]any{"v": int64(1 << 31)}},
	{obj1: &shortPtr{V: int64ptr}, obj2: map[string]any{"v": intvar}},

	{obj1: &shortNonEmptyInt{V: 1}, obj2: map[string]any{"v": 1}},
	{obj1: &shortNonEmptyInt{V: 1 << 31}, obj2: map[string]any{"v": int64(1 << 31)}},
	{obj1: &shortNonEmptyInt{}, obj2: map[string]any{}},

	{obj1: &inlineInt{V: struct{ A, B int }{A: 1, B: 2}}, obj2: map[string]any{"a": 1, "b": 2}},
	{obj1: &inlineMap{A: 1, M: map[string]any{"b": 2}}, obj2: map[string]any{"a": 1, "b": 2}},
	{obj1: &inlineMap{A: 1, M: nil}, obj2: map[string]any{"a": 1}},
	{obj1: &inlineMapInt{A: 1, M: map[string]int{"b": 2}}, obj2: map[string]int{"a": 1, "b": 2}},
	{obj1: &inlineMapInt{A: 1, M: nil}, obj2: map[string]int{"a": 1}},
	{obj1: &inlineMapMyM{A: 1, M: MyM{"b": MyM{"c": 3}}}, obj2: map[string]any{"a": 1, "b": map[string]any{"c": 3}}},
	{obj1: &inlineUnexported{M: map[string]any{"b": 1}, unexported: unexported{A: 2}}, obj2: map[string]any{"b": 1, "a": 2}},

	// []byte <=> Binary
	{obj1: &struct{ B []byte }{B: []byte("abc")}, obj2: map[string]bson.Binary{"b": {Data: []byte("abc")}}},

	// []byte <=> MyBytes
	{obj1: &struct{ B MyBytes }{B: []byte("abc")}, obj2: map[string]string{"b": "abc"}},
	{obj1: &struct{ B MyBytes }{B: []byte{}}, obj2: map[string]string{"b": ""}},
	{obj1: &struct{ B MyBytes }{}, obj2: map[string]bool{}},
	{obj1: &struct{ B []byte }{B: []byte("abc")}, obj2: map[string]MyBytes{"b": []byte("abc")}},

	// bool <=> MyBool
	{obj1: &struct{ B MyBool }{B: true}, obj2: map[string]bool{"b": true}},
	{obj1: &struct{ B MyBool }{}, obj2: map[string]bool{"b": false}},
	{obj1: &struct{ B MyBool }{}, obj2: map[string]string{}},
	{obj1: &struct{ B bool }{}, obj2: map[string]MyBool{"b": false}},

	// arrays
	{obj1: &struct{ V [2]int }{V: [...]int{1, 2}}, obj2: map[string][2]int{"v": {1, 2}}},
	{obj1: &struct{ V [2]byte }{V: [...]byte{1, 2}}, obj2: map[string][2]byte{"v": {1, 2}}},

	// zero time
	{obj1: &struct{ V time.Time }{}, obj2: map[string]any{"v": time.Time{}}},

	// zero time + 1 second + 1 millisecond; overflows int64 as nanoseconds
	{obj1: &struct{ V time.Time }{V: time.Unix(-62135596799, 1e6).Local()},
		obj2: map[string]any{"v": time.Unix(-62135596799, 1e6).Local()}},

	// bson.D <=> []DocElem
	{obj1: &bson.D{{Name: "a", Value: bson.D{{Name: "b", Value: 1}, {Name: "c", Value: 2}}}}, obj2: &bson.D{{Name: "a", Value: bson.D{{Name: "b", Value: 1}, {Name: "c", Value: 2}}}}},
	{obj1: &bson.D{{Name: "a", Value: bson.D{{Name: "b", Value: 1}, {Name: "c", Value: 2}}}}, obj2: &MyD{{Name: "a", Value: MyD{{Name: "b", Value: 1}, {Name: "c", Value: 2}}}}},
	{obj1: &struct{ V MyD }{V: MyD{{Name: "a", Value: 1}}}, obj2: &bson.D{{Name: "v", Value: bson.D{{Name: "a", Value: 1}}}}},

	// bson.RawD <=> []RawDocElem
	{obj1: &bson.RawD{{Name: "a", Value: bson.Raw{Kind: 0x08, Data: []byte{0x01}}}}, obj2: &bson.RawD{{Name: "a", Value: bson.Raw{Kind: 0x08, Data: []byte{0x01}}}}},
	{obj1: &bson.RawD{{Name: "a", Value: bson.Raw{Kind: 0x08, Data: []byte{0x01}}}}, obj2: &MyRawD{{Name: "a", Value: bson.Raw{Kind: 0x08, Data: []byte{0x01}}}}},

	// bson.M <=> map
	{obj1: bson.M{"a": bson.M{"b": 1, "c": 2}}, obj2: MyM{"a": MyM{"b": 1, "c": 2}}},
	{obj1: bson.M{"a": bson.M{"b": 1, "c": 2}}, obj2: map[string]any{"a": map[string]any{"b": 1, "c": 2}}},

	// bson.M <=> map[MyString]
	{obj1: bson.M{"a": bson.M{"b": 1, "c": 2}}, obj2: map[MyString]any{"a": map[MyString]any{"b": 1, "c": 2}}},

	// json.Number <=> int64, float64
	{obj1: &struct{ N json.Number }{N: "5"}, obj2: map[string]any{"n": int64(5)}},
	{obj1: &struct{ N json.Number }{N: "5.05"}, obj2: map[string]any{"n": 5.05}},
	{obj1: &struct{ N json.Number }{N: "9223372036854776000"}, obj2: map[string]any{"n": float64(1 << 63)}},

	// bson.D <=> non-struct getter/setter
	{obj1: &bson.D{{Name: "a", Value: 1}}, obj2: &getterSetterD{{Name: "a", Value: 1}, {Name: "suffix", Value: true}}},
	{obj1: &bson.D{{Name: "a", Value: 42}}, obj2: &gsintvar},

	// Interface slice setter.
	{obj1: &struct{ V ifaceSlice }{V: ifaceSlice{nil, nil, nil}}, obj2: bson.M{"v": []any{3}}},
}

// Same thing, but only one way (obj1 => obj2).
var oneWayCrossItems = []crossTypeItem{
	// map <=> struct
	{obj1: map[string]any{"a": 1, "b": "2", "c": 3}, obj2: map[string]int{"a": 1, "c": 3}},

	// inline map elides badly typed values
	{obj1: map[string]any{"a": 1, "b": "2", "c": 3}, obj2: &inlineMapInt{A: 1, M: map[string]int{"c": 3}}},

	// Can't decode int into struct.
	{obj1: bson.M{"a": bson.M{"b": 2}}, obj2: &struct{ A bool }{}},

	// Would get decoded into a int32 too in the opposite direction.
	{obj1: &shortIface{V: int64(1) << 30}, obj2: map[string]any{"v": 1 << 30}},

	// Ensure omitempty on struct with private fields works properly.
	{obj1: &struct {
		V struct{ v time.Time } ",omitempty"
	}{}, obj2: map[string]any{}},

	// Attempt to marshal slice into RawD (issue #120).
	{obj1: bson.M{"x": []int{1, 2, 3}}, obj2: &struct{ X bson.RawD }{}},
}

func testCrossPair(c *C, dump any, load any) {
	c.Logf("Dump: %#v", dump)
	c.Logf("Load: %#v", load)
	zero := makeZeroDoc(load)
	data, err := bson.Marshal(dump)
	c.Assert(err, IsNil)
	c.Logf("Dumped: %#v", string(data))
	err = bson.Unmarshal(data, zero)
	c.Assert(err, IsNil)
	c.Logf("Loaded: %#v", zero)
	c.Assert(zero, DeepEquals, load)
}

func (s *S) TestTwoWayCrossPairs(c *C) {
	for _, item := range twoWayCrossItems {
		testCrossPair(c, item.obj1, item.obj2)
		testCrossPair(c, item.obj2, item.obj1)
	}
}

func (s *S) TestOneWayCrossPairs(c *C) {
	for _, item := range oneWayCrossItems {
		testCrossPair(c, item.obj1, item.obj2)
	}
}

// --------------------------------------------------------------------------
// ObjectId hex representation test.

func (s *S) TestObjectIdHex(c *C) {
	id := bson.ObjectIdHex("4d88e15b60f486e428412dc9")
	c.Assert(id.String(), Equals, `ObjectIdHex("4d88e15b60f486e428412dc9")`)
	c.Assert(id.Hex(), Equals, "4d88e15b60f486e428412dc9")
}

func (s *S) TestIsObjectIdHex(c *C) {
	test := []struct {
		id    string
		valid bool
	}{
		{id: "4d88e15b60f486e428412dc9", valid: true},
		{id: "4d88e15b60f486e428412dc", valid: false},
		{id: "4d88e15b60f486e428412dc9e", valid: false},
		{id: "4d88e15b60f486e428412dcx", valid: false},
	}
	for _, t := range test {
		c.Assert(bson.IsObjectIdHex(t.id), Equals, t.valid)
	}
}

// --------------------------------------------------------------------------
// ObjectId parts extraction tests.

type objectIdParts struct {
	id        bson.ObjectId
	timestamp int64
	machine   []byte
	pid       uint16
	counter   int32
}

var objectIds = []objectIdParts{
	{
		id:        bson.ObjectIdHex("4d88e15b60f486e428412dc9"),
		timestamp: 1300816219,
		machine:   []byte{0x60, 0xf4, 0x86},
		pid:       0xe428,
		counter:   4271561,
	},
	{
		id:        bson.ObjectIdHex("000000000000000000000000"),
		timestamp: 0,
		machine:   []byte{0x00, 0x00, 0x00},
		pid:       0x0000,
		counter:   0,
	},
	{
		id:        bson.ObjectIdHex("00000000aabbccddee000001"),
		timestamp: 0,
		machine:   []byte{0xaa, 0xbb, 0xcc},
		pid:       0xddee,
		counter:   1,
	},
}

func (s *S) TestObjectIdPartsExtraction(c *C) {
	for i, v := range objectIds {
		t := time.Unix(v.timestamp, 0)
		c.Assert(v.id.Time(), Equals, t, Commentf("#%d Wrong timestamp value", i))
		c.Assert(v.id.Machine(), DeepEquals, v.machine, Commentf("#%d Wrong machine id value", i))
		c.Assert(v.id.Pid(), Equals, v.pid, Commentf("#%d Wrong pid value", i))
		c.Assert(v.id.Counter(), Equals, v.counter, Commentf("#%d Wrong counter value", i))
	}
}

func (s *S) TestNow(c *C) {
	before := time.Now()
	time.Sleep(1e6)
	now := bson.Now()
	time.Sleep(1e6)
	after := time.Now()
	c.Assert(now.After(before) && now.Before(after), Equals, true, Commentf("now=%s, before=%s, after=%s", now, before, after))
}

// --------------------------------------------------------------------------
// ObjectId generation tests.

func (s *S) TestNewObjectId(c *C) {
	// Generate 10 ids
	ids := make([]bson.ObjectId, 10)
	for i := 0; i < 10; i++ {
		ids[i] = bson.NewObjectId()
	}
	for i := 1; i < 10; i++ {
		prevId := ids[i-1]
		id := ids[i]
		// Test for uniqueness among all other 9 generated ids
		for j, tid := range ids {
			if j != i {
				c.Assert(id, Not(Equals), tid, Commentf("Generated ObjectId is not unique"))
			}
		}
		// Check that timestamp was incremented and is within 30 seconds of the previous one
		secs := id.Time().Sub(prevId.Time()).Seconds()
		c.Assert((secs >= 0 && secs <= 30), Equals, true, Commentf("Wrong timestamp in generated ObjectId"))
		// Check that machine ids are the same
		c.Assert(id.Machine(), DeepEquals, prevId.Machine())
		// Check that pids are the same
		c.Assert(id.Pid(), Equals, prevId.Pid())
		// Test for proper increment
		delta := int(id.Counter() - prevId.Counter())
		c.Assert(delta, Equals, 1, Commentf("Wrong increment in generated ObjectId"))
	}
}

func (s *S) TestNewObjectIdWithTime(c *C) {
	t := time.Unix(12345678, 0)
	id := bson.NewObjectIdWithTime(t)
	c.Assert(id.Time(), Equals, t)
	c.Assert(id.Machine(), DeepEquals, []byte{0x00, 0x00, 0x00})
	c.Assert(int(id.Pid()), Equals, 0)
	c.Assert(int(id.Counter()), Equals, 0)
}

// --------------------------------------------------------------------------
// ObjectId JSON marshalling.

type jsonType struct {
	Id bson.ObjectId
}

var jsonIdTests = []struct {
	value     jsonType
	json      string
	marshal   bool
	unmarshal bool
	error     string
}{{
	value:     jsonType{Id: bson.ObjectIdHex("4d88e15b60f486e428412dc9")},
	json:      `{"Id":"4d88e15b60f486e428412dc9"}`,
	marshal:   true,
	unmarshal: true,
}, {
	value:     jsonType{},
	json:      `{"Id":""}`,
	marshal:   true,
	unmarshal: true,
}, {
	value:     jsonType{},
	json:      `{"Id":null}`,
	marshal:   false,
	unmarshal: true,
}, {
	json:      `{"Id":"4d88e15b60f486e428412dc9A"}`,
	error:     `invalid ObjectId in JSON: "4d88e15b60f486e428412dc9A"`,
	marshal:   false,
	unmarshal: true,
}, {
	json:      `{"Id":"4d88e15b60f486e428412dcZ"}`,
	error:     `invalid ObjectId in JSON: "4d88e15b60f486e428412dcZ" .*`,
	marshal:   false,
	unmarshal: true,
}}

func (s *S) TestObjectIdJSONMarshaling(c *C) {
	for _, test := range jsonIdTests {
		if test.marshal {
			data, err := json.Marshal(&test.value)
			if test.error == "" {
				c.Assert(err, IsNil)
				c.Assert(string(data), Equals, test.json)
			} else {
				c.Assert(err, ErrorMatches, test.error)
			}
		}

		if test.unmarshal {
			var value jsonType
			err := json.Unmarshal([]byte(test.json), &value)
			if test.error == "" {
				c.Assert(err, IsNil)
				c.Assert(value, DeepEquals, test.value)
			} else {
				c.Assert(err, ErrorMatches, test.error)
			}
		}
	}
}

// --------------------------------------------------------------------------
// Spec tests

type specTest struct {
	Description string
	Documents   []struct {
		Decoded    map[string]any
		Encoded    string
		DecodeOnly bool `yaml:"decodeOnly"`
		Error      any
	}
}

func (s *S) TestSpecTests(c *C) {
	for _, data := range specTests {
		var test specTest
		err := yaml.Unmarshal([]byte(data), &test)
		c.Assert(err, IsNil)

		c.Logf("Running spec test set %q", test.Description)

		for _, doc := range test.Documents {
			if doc.Error != nil {
				continue
			}
			c.Logf("Ensuring %q decodes as %v", doc.Encoded, doc.Decoded)
			var decoded map[string]any
			encoded, err := hex.DecodeString(doc.Encoded)
			c.Assert(err, IsNil)
			err = bson.Unmarshal(encoded, &decoded)
			c.Assert(err, IsNil)
			c.Assert(decoded, DeepEquals, doc.Decoded)
		}

		for _, doc := range test.Documents {
			if doc.DecodeOnly || doc.Error != nil {
				continue
			}
			c.Logf("Ensuring %v encodes as %q", doc.Decoded, doc.Encoded)
			encoded, err := bson.Marshal(doc.Decoded)
			c.Assert(err, IsNil)
			c.Assert(strings.ToUpper(hex.EncodeToString(encoded)), Equals, doc.Encoded)
		}

		for _, doc := range test.Documents {
			if doc.Error == nil {
				continue
			}
			c.Logf("Ensuring %q errors when decoded: %s", doc.Encoded, doc.Error)
			var decoded map[string]any
			encoded, err := hex.DecodeString(doc.Encoded)
			c.Assert(err, IsNil)
			err = bson.Unmarshal(encoded, &decoded)
			c.Assert(err, NotNil)
			c.Logf("Failed with: %v", err)
		}
	}
}

// --------------------------------------------------------------------------
// ObjectId Text encoding.TextUnmarshaler.

var textIdTests = []struct {
	value     bson.ObjectId
	text      string
	marshal   bool
	unmarshal bool
	error     string
}{{
	value:     bson.ObjectIdHex("4d88e15b60f486e428412dc9"),
	text:      "4d88e15b60f486e428412dc9",
	marshal:   true,
	unmarshal: true,
}, {
	text:      "",
	marshal:   true,
	unmarshal: true,
}, {
	text:      "4d88e15b60f486e428412dc9A",
	marshal:   false,
	unmarshal: true,
	error:     `invalid ObjectId: 4d88e15b60f486e428412dc9A`,
}, {
	text:      "4d88e15b60f486e428412dcZ",
	marshal:   false,
	unmarshal: true,
	error:     `invalid ObjectId: 4d88e15b60f486e428412dcZ .*`,
}}

func (s *S) TestObjectIdTextMarshaling(c *C) {
	for _, test := range textIdTests {
		if test.marshal {
			data, err := test.value.MarshalText()
			if test.error == "" {
				c.Assert(err, IsNil)
				c.Assert(string(data), Equals, test.text)
			} else {
				c.Assert(err, ErrorMatches, test.error)
			}
		}

		if test.unmarshal {
			err := test.value.UnmarshalText([]byte(test.text))
			if test.error == "" {
				c.Assert(err, IsNil)
				if test.value != "" {
					value := bson.ObjectIdHex(test.text)
					c.Assert(value, DeepEquals, test.value)
				}
			} else {
				c.Assert(err, ErrorMatches, test.error)
			}
		}
	}
}

// --------------------------------------------------------------------------
// ObjectId XML marshalling.

type xmlType struct {
	Id bson.ObjectId
}

var xmlIdTests = []struct {
	value     xmlType
	xml       string
	marshal   bool
	unmarshal bool
	error     string
}{{
	value:     xmlType{Id: bson.ObjectIdHex("4d88e15b60f486e428412dc9")},
	xml:       "<xmlType><Id>4d88e15b60f486e428412dc9</Id></xmlType>",
	marshal:   true,
	unmarshal: true,
}, {
	value:     xmlType{},
	xml:       "<xmlType><Id></Id></xmlType>",
	marshal:   true,
	unmarshal: true,
}, {
	xml:       "<xmlType><Id>4d88e15b60f486e428412dc9A</Id></xmlType>",
	marshal:   false,
	unmarshal: true,
	error:     `invalid ObjectId: 4d88e15b60f486e428412dc9A`,
}, {
	xml:       "<xmlType><Id>4d88e15b60f486e428412dcZ</Id></xmlType>",
	marshal:   false,
	unmarshal: true,
	error:     `invalid ObjectId: 4d88e15b60f486e428412dcZ .*`,
}}

func (s *S) TestObjectIdXMLMarshaling(c *C) {
	for _, test := range xmlIdTests {
		if test.marshal {
			data, err := xml.Marshal(&test.value)
			if test.error == "" {
				c.Assert(err, IsNil)
				c.Assert(string(data), Equals, test.xml)
			} else {
				c.Assert(err, ErrorMatches, test.error)
			}
		}

		if test.unmarshal {
			var value xmlType
			err := xml.Unmarshal([]byte(test.xml), &value)
			if test.error == "" {
				c.Assert(err, IsNil)
				c.Assert(value, DeepEquals, test.value)
			} else {
				c.Assert(err, ErrorMatches, test.error)
			}
		}
	}
}

// --------------------------------------------------------------------------
// Some simple benchmarks.

type BenchT struct {
	A, B, C, D, E, F string
}

type BenchRawT struct {
	A string
	B int
	C bson.M
	D []float64
}

func (s *S) BenchmarkUnmarhsalStruct(c *C) {
	v := BenchT{A: "A", D: "D", E: "E"}
	data, err := bson.Marshal(&v)
	if err != nil {
		panic(err)
	}
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		err = bson.Unmarshal(data, &v)
	}
	if err != nil {
		panic(err)
	}
}

func (s *S) BenchmarkUnmarhsalMap(c *C) {
	m := bson.M{"a": "a", "d": "d", "e": "e"}
	data, err := bson.Marshal(&m)
	if err != nil {
		panic(err)
	}
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		err = bson.Unmarshal(data, &m)
	}
	if err != nil {
		panic(err)
	}
}

func (s *S) BenchmarkUnmarshalRaw(c *C) {
	var err error
	m := BenchRawT{
		A: "test_string",
		B: 123,
		C: bson.M{
			"subdoc_int": 12312,
			"subdoc_doc": bson.M{"1": 1},
		},
		D: []float64{0.0, 1.3333, -99.9997, 3.1415},
	}
	data, err := bson.Marshal(&m)
	if err != nil {
		panic(err)
	}
	raw := bson.Raw{}
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		err = bson.Unmarshal(data, &raw)
	}
	if err != nil {
		panic(err)
	}
}

func (s *S) BenchmarkNewObjectId(c *C) {
	for i := 0; i < c.N; i++ {
		bson.NewObjectId()
	}
}
