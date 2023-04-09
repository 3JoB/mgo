package txn

import (
	"sort"

	. "gopkg.in/check.v1"
)

type DocKeySuite struct{}

var _ = Suite(&DocKeySuite{})

type T struct {
	A int
	B string
}

type T2 struct {
	A int
	B string
}

type T3 struct {
	A int
	B string
}

type T4 struct {
	A int
	B string
}

type T5 struct {
	F int
	Q string
}

type T6 struct {
	A int
	B string
}

type T7 struct {
	A bool
	B float64
}

type T8 struct {
	A int
	B string
}

type T9 struct {
	A int
	B string
	C bool
}

type T10 struct {
	C int    `bson:"a"`
	D string `bson:"b,omitempty"`
}

type T11 struct {
	C int
	D string
}

type T12 struct {
	S string
}

type T13 struct {
	p, q, r bool
	S       string
}

var docKeysTests = [][]docKeys{
	{{
		{C: "c", Id: 1},
		{C: "c", Id: 5},
		{C: "c", Id: 2},
	}, {
		{C: "c", Id: 1},
		{C: "c", Id: 2},
		{C: "c", Id: 5},
	}}, {{
		{C: "c", Id: "foo"},
		{C: "c", Id: "bar"},
		{C: "c", Id: "bob"},
	}, {
		{C: "c", Id: "bar"},
		{C: "c", Id: "bob"},
		{C: "c", Id: "foo"},
	}}, {{
		{C: "c", Id: 0.2},
		{C: "c", Id: 0.07},
		{C: "c", Id: 0.9},
	}, {
		{C: "c", Id: 0.07},
		{C: "c", Id: 0.2},
		{C: "c", Id: 0.9},
	}}, {{
		{C: "c", Id: true},
		{C: "c", Id: false},
		{C: "c", Id: true},
	}, {
		{C: "c", Id: false},
		{C: "c", Id: true},
		{C: "c", Id: true},
	}}, {{
		{C: "c", Id: T{A: 1, B: "b"}},
		{C: "c", Id: T{A: 1, B: "a"}},
		{C: "c", Id: T{A: 0, B: "b"}},
		{C: "c", Id: T{A: 0, B: "a"}},
	}, {
		{C: "c", Id: T{A: 0, B: "a"}},
		{C: "c", Id: T{A: 0, B: "b"}},
		{C: "c", Id: T{A: 1, B: "a"}},
		{C: "c", Id: T{A: 1, B: "b"}},
	}}, {{
		{C: "c", Id: T{A: 1, B: "a"}},
		{C: "c", Id: T{A: 0, B: "a"}},
	}, {
		{C: "c", Id: T{A: 0, B: "a"}},
		{C: "c", Id: T{A: 1, B: "a"}},
	}}, {{
		{C: "c", Id: T3{A: 0, B: "b"}},
		{C: "c", Id: T2{A: 1, B: "b"}},
		{C: "c", Id: T3{A: 1, B: "a"}},
		{C: "c", Id: T2{A: 0, B: "a"}},
	}, {
		{C: "c", Id: T2{A: 0, B: "a"}},
		{C: "c", Id: T3{A: 0, B: "b"}},
		{C: "c", Id: T3{A: 1, B: "a"}},
		{C: "c", Id: T2{A: 1, B: "b"}},
	}}, {{
		{C: "c", Id: T5{F: 1, Q: "b"}},
		{C: "c", Id: T4{A: 1, B: "b"}},
		{C: "c", Id: T5{F: 0, Q: "a"}},
		{C: "c", Id: T4{A: 0, B: "a"}},
	}, {
		{C: "c", Id: T4{A: 0, B: "a"}},
		{C: "c", Id: T5{F: 0, Q: "a"}},
		{C: "c", Id: T4{A: 1, B: "b"}},
		{C: "c", Id: T5{F: 1, Q: "b"}},
	}}, {{
		{C: "c", Id: T6{A: 1, B: "b"}},
		{C: "c", Id: T7{A: true, B: 0.2}},
		{C: "c", Id: T6{A: 0, B: "a"}},
		{C: "c", Id: T7{A: false, B: 0.04}},
	}, {
		{C: "c", Id: T6{A: 0, B: "a"}},
		{C: "c", Id: T6{A: 1, B: "b"}},
		{C: "c", Id: T7{A: false, B: 0.04}},
		{C: "c", Id: T7{A: true, B: 0.2}},
	}}, {{
		{C: "c", Id: T9{A: 1, B: "b", C: true}},
		{C: "c", Id: T8{A: 1, B: "b"}},
		{C: "c", Id: T9{A: 0, B: "a", C: false}},
		{C: "c", Id: T8{A: 0, B: "a"}},
	}, {
		{C: "c", Id: T9{A: 0, B: "a", C: false}},
		{C: "c", Id: T8{A: 0, B: "a"}},
		{C: "c", Id: T9{A: 1, B: "b", C: true}},
		{C: "c", Id: T8{A: 1, B: "b"}},
	}}, {{
		{C: "b", Id: 2},
		{C: "a", Id: 5},
		{C: "c", Id: 2},
		{C: "b", Id: 1},
	}, {
		{C: "a", Id: 5},
		{C: "b", Id: 1},
		{C: "b", Id: 2},
		{C: "c", Id: 2},
	}}, {{
		{C: "c", Id: T11{C: 1, D: "a"}},
		{C: "c", Id: T11{C: 1, D: "a"}},
		{C: "c", Id: T10{C: 1, D: "a"}},
	}, {
		{C: "c", Id: T10{C: 1, D: "a"}},
		{C: "c", Id: T11{C: 1, D: "a"}},
		{C: "c", Id: T11{C: 1, D: "a"}},
	}}, {{
		{C: "c", Id: T12{S: "a"}},
		{C: "c", Id: T13{p: false, q: true, r: false, S: "a"}},
		{C: "c", Id: T12{S: "b"}},
		{C: "c", Id: T13{p: false, q: true, r: false, S: "b"}},
	}, {
		{C: "c", Id: T12{S: "a"}},
		{C: "c", Id: T13{p: false, q: true, r: false, S: "a"}},
		{C: "c", Id: T12{S: "b"}},
		{C: "c", Id: T13{p: false, q: true, r: false, S: "b"}},
	}},
}

func (s *DocKeySuite) TestSort(c *C) {
	for _, test := range docKeysTests {
		keys := test[0]
		expected := test[1]
		sort.Sort(keys)
		c.Check(keys, DeepEquals, expected)
	}
}
