package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_resourceOrdering_order(t *testing.T) {
	tests := []struct {
		name string
		ro   resourceOrdering
		want []*Tuple
	}{{
		name: "empty",
		ro:   resourceOrdering{},
		want: []*Tuple{},
	}, {
		name: "no dependencies",
		ro: resourceOrdering{
			m: map[string]Tuple{
				"foo": {Name: "foo"},
				"bar": {Name: "bar"},
				"baz": {Name: "baz"},
			},
		},
		want: []*Tuple{
			{Name: "bar"},
			{Name: "baz"},
			{Name: "foo"},
		},
	}, {
		name: "one dependency",
		ro: resourceOrdering{
			m: map[string]Tuple{
				"t.foo": {
					Type: "t",
					Name: "foo",
				},
				"t.bar": {
					Type:         "t",
					Name:         "bar",
					Dependencies: []string{"t.foo"},
				},
				"t.baz": {
					Type: "t",
					Name: "baz",
				},
			},
		},
		want: []*Tuple{
			{
				Type: "t",
				Name: "foo",
			}, {
				Type:         "t",
				Name:         "bar",
				Dependencies: []string{"t.foo"},
			}, {
				Type: "t",
				Name: "baz",
			},
		},
	}, {
		name: "multiple dependencies on one resource",
		ro: resourceOrdering{
			m: map[string]Tuple{
				"t.foo": {
					Type: "t",
					Name: "foo",
				},
				"t.bar": {
					Type:         "t",
					Name:         "bar",
					Dependencies: []string{"t.foo", "t.baz"},
				},
				"t.baz": {
					Type: "t",
					Name: "baz",
				},
			},
		},
		want: []*Tuple{
			{
				Type: "t",
				Name: "foo",
			}, {
				Type: "t",
				Name: "baz",
			}, {
				Type:         "t",
				Name:         "bar",
				Dependencies: []string{"t.foo", "t.baz"},
			},
		},
	}, {
		name: "dependencies on multiple resources",
		ro: resourceOrdering{
			m: map[string]Tuple{
				"t.foo": {
					Type:         "t",
					Name:         "foo",
					Dependencies: []string{"t.baz"},
				},
				"t.bar": {
					Type:         "t",
					Name:         "bar",
					Dependencies: []string{"t.foo"},
				},
				"t.baz": {
					Type: "t",
					Name: "baz",
				},
			},
		},
		want: []*Tuple{
			{
				Type: "t",
				Name: "baz",
			}, {
				Type:         "t",
				Name:         "foo",
				Dependencies: []string{"t.baz"},
			}, {
				Type:         "t",
				Name:         "bar",
				Dependencies: []string{"t.foo"},
			},
		},
	}, {
		name: "dependencies on multiple resources w/ duplicate names",
		ro: resourceOrdering{
			m: map[string]Tuple{
				"t.foo": {
					Type:         "t",
					Name:         "foo",
					Dependencies: []string{"t.baz"},
				},
				"t.bar": {
					Type:         "t",
					Name:         "bar",
					Dependencies: []string{"t.foo"},
				},
				"t.baz": {
					Type: "t",
					Name: "baz",
				},
			},
		},
		want: []*Tuple{
			{
				Type: "t",
				Name: "baz",
			}, {
				Type:         "t",
				Name:         "foo",
				Dependencies: []string{"t.baz"},
			},
			{
				Type:         "t",
				Name:         "bar",
				Dependencies: []string{"t.foo"},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.ro.order()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Error("order() return mismatch (-want, +got):", diff)
			}
		})
	}
}
