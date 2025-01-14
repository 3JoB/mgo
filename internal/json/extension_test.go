package json

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

type funcN struct {
	Arg1 int `json:"arg1"`
	Arg2 int `json:"arg2"`
}

type funcs struct {
	Func2 *funcN `json:"$func2"`
	Func1 *funcN `json:"$func1"`
}

type funcsText struct {
	Func1 jsonText `json:"$func1"`
	Func2 jsonText `json:"$func2"`
}

type jsonText struct {
	json string
}

func (jt *jsonText) UnmarshalJSON(data []byte) error {
	jt.json = string(data)
	return nil
}

type nestedText struct {
	F jsonText
	B bool
}

type unquotedKey struct {
	S string `json:"$k_1"`
}

var ext Extension

type keyed string

func decodeKeyed(data []byte) (any, error) {
	return keyed(data), nil
}

type keyedType struct {
	K keyed
	I int
}

type docint int

type const1Type struct{}

var const1 = new(const1Type)

func init() {
	ext.DecodeFunc("Func1", "$func1")
	ext.DecodeFunc("Func2", "$func2", "arg1", "arg2")
	ext.DecodeFunc("Func3", "$func3", "arg1")
	ext.DecodeFunc("new Func4", "$func4", "arg1")

	ext.DecodeConst("Const1", const1)

	ext.DecodeKeyed("$key1", decodeKeyed)
	ext.DecodeKeyed("$func3", decodeKeyed)

	ext.EncodeType(docint(0), func(v any) ([]byte, error) {
		s := `{"$docint": ` + strconv.Itoa(int(v.(docint))) + `}`
		return []byte(s), nil
	})

	ext.DecodeUnquotedKeys(true)
	ext.DecodeTrailingCommas(true)
}

type extDecodeTest struct {
	in  string
	ptr any
	out any
	err error

	noext bool
}

var extDecodeTests = []extDecodeTest{
	// Functions
	{in: `Func1()`, ptr: new(any), out: map[string]any{
		"$func1": map[string]any{},
	}},
	{in: `{"v": Func1()}`, ptr: new(any), out: map[string]any{
		"v": map[string]any{"$func1": map[string]any{}},
	}},
	{in: `Func2(1)`, ptr: new(any), out: map[string]any{
		"$func2": map[string]any{"arg1": float64(1)},
	}},
	{in: `Func2(1, 2)`, ptr: new(any), out: map[string]any{
		"$func2": map[string]any{"arg1": float64(1), "arg2": float64(2)},
	}},
	{in: `Func2(Func1())`, ptr: new(any), out: map[string]any{
		"$func2": map[string]any{"arg1": map[string]any{"$func1": map[string]any{}}},
	}},
	{in: `Func2(1, 2, 3)`, ptr: new(any), err: errors.New("json: too many arguments for function Func2")},
	{in: `BadFunc()`, ptr: new(any), err: errors.New(`json: unknown function "BadFunc"`)},

	{in: `Func1()`, ptr: new(funcs), out: funcs{Func1: &funcN{}}},
	{in: `Func2(1)`, ptr: new(funcs), out: funcs{Func2: &funcN{Arg1: 1}}},
	{in: `Func2(1, 2)`, ptr: new(funcs), out: funcs{Func2: &funcN{Arg1: 1, Arg2: 2}}},

	{in: `Func2(1, 2, 3)`, ptr: new(funcs), err: errors.New("json: too many arguments for function Func2")},
	{in: `BadFunc()`, ptr: new(funcs), err: errors.New(`json: unknown function "BadFunc"`)},

	{in: `Func2(1)`, ptr: new(jsonText), out: jsonText{json: "Func2(1)"}},
	{in: `Func2(1, 2)`, ptr: new(funcsText), out: funcsText{Func2: jsonText{json: "Func2(1, 2)"}}},
	{in: `{"f": Func2(1, 2), "b": true}`, ptr: new(nestedText), out: nestedText{F: jsonText{json: "Func2(1, 2)"}, B: true}},

	{in: `Func1()`, ptr: new(struct{}), out: struct{}{}},

	// Functions with "new" prefix
	{in: `new Func4(1)`, ptr: new(any), out: map[string]any{
		"$func4": map[string]any{"arg1": float64(1)},
	}},

	// Constants
	{in: `Const1`, ptr: new(any), out: const1},
	{in: `{"c": Const1}`, ptr: new(struct{ C *const1Type }), out: struct{ C *const1Type }{C: const1}},

	// Keyed documents
	{in: `{"v": {"$key1": 1}}`, ptr: new(any), out: map[string]any{"v": keyed(`{"$key1": 1}`)}},
	{in: `{"k": {"$key1": 1}}`, ptr: new(keyedType), out: keyedType{K: keyed(`{"$key1": 1}`)}},
	{in: `{"i": {"$key1": 1}}`, ptr: new(keyedType), err: &UnmarshalTypeError{Value: "object", Type: reflect.TypeOf(0), Offset: 18}},

	// Keyed function documents
	{in: `{"v": Func3()}`, ptr: new(any), out: map[string]any{"v": keyed(`Func3()`)}},
	{in: `{"k": Func3()}`, ptr: new(keyedType), out: keyedType{K: keyed(`Func3()`)}},
	{in: `{"i": Func3()}`, ptr: new(keyedType), err: &UnmarshalTypeError{Value: "object", Type: reflect.TypeOf(0), Offset: 13}},

	// Unquoted keys
	{in: `{$k_1: "bar"}`, ptr: new(any), out: map[string]any{"$k_1": "bar"}},
	{in: `{$k_1: "bar"}`, ptr: new(unquotedKey), out: unquotedKey{S: "bar"}},

	{in: `{$k_1: "bar"}`, noext: true, ptr: new(any),
		err: &SyntaxError{msg: "invalid character '$' looking for beginning of object key string", Offset: 2}},
	{in: `{$k_1: "bar"}`, noext: true, ptr: new(unquotedKey),
		err: &SyntaxError{msg: "invalid character '$' looking for beginning of object key string", Offset: 2}},

	// Trailing commas
	{in: `{"k": "v",}`, ptr: new(any), out: map[string]any{"k": "v"}},
	{in: `{"k": "v",}`, ptr: new(struct{}), out: struct{}{}},
	{in: `["v",]`, ptr: new(any), out: []any{"v"}},

	{in: `{"k": "v",}`, noext: true, ptr: new(any),
		err: &SyntaxError{msg: "invalid character '}' looking for beginning of object key string", Offset: 11}},
	{in: `{"k": "v",}`, noext: true, ptr: new(struct{}),
		err: &SyntaxError{msg: "invalid character '}' looking for beginning of object key string", Offset: 11}},
	{in: `["a",]`, noext: true, ptr: new(any),
		err: &SyntaxError{msg: "invalid character ']' looking for beginning of value", Offset: 6}},
}

type extEncodeTest struct {
	in  any
	out string
	err error
}

var extEncodeTests = []extEncodeTest{
	{in: docint(13), out: "{\"$docint\":13}\n"},
}

func TestExtensionDecode(t *testing.T) {
	for i, tt := range extDecodeTests {
		in := []byte(tt.in)

		// v = new(right-type)
		v := reflect.New(reflect.TypeOf(tt.ptr).Elem())
		dec := NewDecoder(bytes.NewReader(in))
		if !tt.noext {
			dec.Extend(&ext)
		}
		if err := dec.Decode(v.Interface()); !reflect.DeepEqual(err, tt.err) {
			t.Errorf("#%d: %v, want %v", i, err, tt.err)
			continue
		} else if err != nil {
			continue
		}
		if !reflect.DeepEqual(v.Elem().Interface(), tt.out) {
			t.Errorf("#%d: mismatch\nhave: %#+v\nwant: %#+v", i, v.Elem().Interface(), tt.out)
			data, _ := Marshal(v.Elem().Interface())
			t.Logf("%s", string(data))
			data, _ = Marshal(tt.out)
			t.Logf("%s", string(data))
			continue
		}
	}
}

func TestExtensionEncode(t *testing.T) {
	var buf bytes.Buffer
	for i, tt := range extEncodeTests {
		buf.Truncate(0)
		enc := NewEncoder(&buf)
		enc.Extend(&ext)
		err := enc.Encode(tt.in)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("#%d: %v, want %v", i, err, tt.err)
			continue
		}
		if buf.String() != tt.out {
			t.Errorf("#%d: mismatch\nhave: %q\nwant: %q", i, buf.String(), tt.out)
		}
	}
}
