package utils

import (
	"reflect"
	"testing"
)

func Test_DeepCopy(t *testing.T) {
	t.Parallel()
	type Case struct {
		src  interface{}
		dest interface{}
	}

	cases := []Case{
		{nil, nil},
		{true, true},
		{"abc", "abc"},
		{123, float64(123)},
		{3.1415926, 3.1415926},
		{map[string]interface{}{}, map[string]interface{}{}},
		{[]string{"a", "b", "c"}, []interface{}{"a", "b", "c"}},
		{map[string]interface{}{"a": 1, "b": "x"}, map[string]interface{}{"b": "x", "a": float64(1)}},
	}

	for _, c := range cases {
		d, err := DeepCopy(c.src)
		if err != nil || !reflect.DeepEqual(d, c.dest) {
			t.Fail()
		}
		if d != nil && &d == &c.src {
			t.Fail()
		}
		// t.Logf("running %v", c)
	}
}

func Test_SortIt(t *testing.T) {
	t.Parallel()
	A := []interface{}{
		map[string]interface{}{"name": "customHTTPProfile"},
		map[string]interface{}{"name": "customTCPProfile"},
	}
	B := []interface{}{
		map[string]interface{}{"name": "customTCPProfile"},
		map[string]interface{}{"name": "customHTTPProfile"},
	}

	C := SortIt(&A)
	D := SortIt(&B)
	if !reflect.DeepEqual(C, D) {
		t.Fail()
	}
	if reflect.DeepEqual(A, C) && reflect.DeepEqual(B, D) {
		t.Fail()
	}

	a := []interface{}{1, 2, 3, 4, 5, 6}
	b := []interface{}{1, 3, 5, 6, 4, 2}
	if !reflect.DeepEqual(SortIt(&a), SortIt(&b)) {
		t.Fail()
	}
}
