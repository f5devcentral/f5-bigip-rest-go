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

func Test_FieldsIsExpected(t *testing.T) {
	type args struct {
		fields   interface{}
		expected interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "empty json",
			args: args{
				fields:   map[string]interface{}{},
				expected: map[string]interface{}{},
			},
			want: true,
		},
		{
			name: "1 field json true",
			args: args{
				fields: map[string]interface{}{
					"a": 1,
				},
				expected: map[string]interface{}{
					"a": 1,
				},
			},
			want: true,
		},
		{
			name: "1 field json false",
			args: args{
				fields: map[string]interface{}{
					"a": 1,
				},
				expected: map[string]interface{}{
					"a": 2,
				},
			},
			want: false,
		},
		{
			name: "n field json true",
			args: args{
				fields: map[string]interface{}{
					"a": 1,
				},
				expected: map[string]interface{}{
					"a": 1,
					"b": "x",
				},
			},
			want: true,
		},
		{
			name: "n field json false",
			args: args{
				fields: map[string]interface{}{
					"a": 1,
				},
				expected: map[string]interface{}{
					"a": 2,
					"b": "x",
				},
			},
			want: false,
		},
		{
			name: "n field json true",
			args: args{
				fields: map[string]interface{}{
					"a": map[string]interface{}{
						"x": 1,
						"y": 2,
					},
				},
				expected: map[string]interface{}{
					"a": map[string]interface{}{
						"x": 1,
						"y": 2,
					},
					"b": "x",
				},
			},
			want: true,
		},
		{
			name: "n field json true",
			args: args{
				fields: map[string]interface{}{
					"a": []interface{}{
						1, "a",
					},
				},
				expected: map[string]interface{}{
					"a": []interface{}{
						1, "a",
					},
					"b": "x",
				},
			},
			want: true,
		},
		{
			name: "n field json false",
			args: args{
				fields: map[string]interface{}{
					"a": []interface{}{
						"a", 1,
					},
				},
				expected: map[string]interface{}{
					"a": []interface{}{
						1, "a",
					},
					"b": "x",
				},
			},
			want: false,
		},
		{
			name: "nil nil true",
			args: args{
				fields:   nil,
				expected: nil,
			},
			want: true,
		},
		{
			name: "nil json true",
			args: args{
				fields: nil,
				expected: map[string]interface{}{
					"a": []interface{}{
						1, "a",
					},
					"b": "x",
				},
			},
			want: true,
		},
		{
			name: "array true",
			args: args{
				fields:   []string{"a", "b"},
				expected: []string{"a", "b"},
			},
			want: true,
		},
		{
			name: "string true",
			args: args{
				fields:   "f5-bigip-rest-go",
				expected: "f5-bigip-rest-go",
			},
			want: true,
		},
		{
			name: "int and float64 true",
			args: args{
				fields:   23,
				expected: float64(23),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FieldsIsExpected(tt.args.fields, tt.args.expected); got != tt.want {
				t.Errorf("FieldsIsExpected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeepEqual(t *testing.T) {
	type args struct {
		a interface{}
		b interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "int",
			args: args{
				a: 11,
				b: 11,
			},
			want: true,
		},
		{
			name: "int float64",
			args: args{
				a: 11,
				b: float64(11),
			},
			want: true,
		},
		{
			name: "string",
			args: args{
				a: "11",
				b: "11",
			},
			want: true,
		},
		{
			name: "array true",
			args: args{
				a: []int{11},
				b: []int{11},
			},
			want: true,
		},
		{
			name: "array true",
			args: args{
				a: []int{11},
				b: []float64{11},
			},
			want: true,
		},
		{
			name: "array true",
			args: args{
				a: []float32{11},
				b: []float64{11},
			},
			want: true,
		},
		{
			name: "array false",
			args: args{
				a: []int{11, 12},
				b: []int{12, 11},
			},
			want: false,
		},
		{
			name: "map true",
			args: args{
				a: map[string]interface{}{
					"a": 11,
				},
				b: map[string]int{
					"a": 11,
				},
			},
			want: true,
		},
		{
			name: "map true",
			args: args{
				a: map[string]interface{}{
					"a": 11,
				},
				b: map[string]string{
					"a": "11",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeepEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("DeepEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
