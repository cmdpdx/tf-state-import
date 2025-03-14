package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cmdpdx/tf-state-import/pkg/state"
)

func TestTupleAddress(t *testing.T) {
	tests := []struct {
		name string
		t    Tuple
		want string
	}{{
		name: "type and name",
		t: Tuple{
			Type: "type",
			Name: "name",
		},
		want: "type.name",
	}, {
		name: "type, name, and int index",
		t: Tuple{
			Type:     "type",
			Name:     "name",
			IndexKey: 1,
		},
		want: "type.name[1]",
	}, {
		name: "type, name, and float64 index",
		t: Tuple{
			Type:     "type",
			Name:     "name",
			IndexKey: float64(1),
		},
		want: "type.name[1]",
	}, {
		name: "type, name, and string index",
		t: Tuple{
			Type:     "type",
			Name:     "name",
			IndexKey: "key",
		},
		want: "type.name[\"key\"]",
	}, {
		name: "module, type, and name",
		t: Tuple{
			Module: "module",
			Type:   "type",
			Name:   "name",
		},
		want: "module.type.name",
	}, {
		name: "module, type, name, and int index",
		t: Tuple{
			Module:   "module",
			Type:     "type",
			Name:     "name",
			IndexKey: 1,
		},
		want: "module.type.name[1]",
	}, {
		name: "module, type, name, and string index",
		t: Tuple{
			Module:   "module",
			Type:     "type",
			Name:     "name",
			IndexKey: "key",
		},
		want: "module.type.name[\"key\"]",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.t.Address()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Address() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

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

func TestFromState(t *testing.T) {
	for _, tt := range []struct {
		name     string
		state    state.V4
		provider string
		want     ResourceMap
	}{{
		name:  "empty",
		state: state.V4{},
		want:  ResourceMap{},
	}, {
		name: "all permutations, no provider",
		state: state.V4{
			Resources: []state.Resource{{
				Mode:     "data",
				Type:     "chainguard_roles",
				Name:     "roles",
				Provider: "provider[\"registry.terraform.io/chainguard/chainguard\"]",
			}, {
				Mode:     "managed",
				Type:     "chainguard_group",
				Name:     "group",
				Provider: "provider[\"registry.terraform.io/chainguard/chainguard\"]",
				Instances: []state.Instance{{
					Attributes: map[string]interface{}{
						"id": "group-id",
					},
				}},
			}, {
				Module:   "module.my_module",
				Mode:     "managed",
				Type:     "chainguard_group",
				Name:     "group-int-index",
				Provider: "provider[\"registry.terraform.io/chainguard/chainguard\"]",
				Instances: []state.Instance{{
					Attributes: map[string]interface{}{
						"id": "group-int-index-id",
					},
					IndexKey: 0,
				}},
			}, {
				Module:   "module.my_module",
				Mode:     "managed",
				Type:     "chainguard_group",
				Name:     "group-string-index",
				Provider: "provider[\"registry.terraform.io/chainguard/chainguard\"]",
				Instances: []state.Instance{{
					Attributes: map[string]interface{}{
						"id": "group-string-index-id",
					},
					IndexKey: "foo",
				}},
			}, {
				Module:   "module.my_module[\"index\"]",
				Mode:     "managed",
				Type:     "chainguard_group",
				Name:     "group-int-index",
				Provider: "provider[\"registry.terraform.io/chainguard/chainguard\"]",
				Instances: []state.Instance{{
					Attributes: map[string]interface{}{
						"id": "group-int-index-id",
					},
					IndexKey: 0,
				}},
			}},
		},
		want: ResourceMap{
			"chainguard_group.group": Tuple{
				Type: "chainguard_group",
				Name: "group",
				ID:   "group-id",
				Attributes: map[string]interface{}{
					"id": "group-id",
				},
			},
			"module.my_module.chainguard_group.group-int-index[0]": Tuple{
				Module:   "module.my_module",
				Type:     "chainguard_group",
				Name:     "group-int-index",
				ID:       "group-int-index-id",
				IndexKey: 0,
				Attributes: map[string]interface{}{
					"id": "group-int-index-id",
				},
			},
			"module.my_module.chainguard_group.group-string-index[\"foo\"]": Tuple{
				Module:   "module.my_module",
				Type:     "chainguard_group",
				Name:     "group-string-index",
				ID:       "group-string-index-id",
				IndexKey: "foo",
				Attributes: map[string]interface{}{
					"id": "group-string-index-id",
				},
			},
			"module.my_module[\"index\"].chainguard_group.group-int-index[0]": Tuple{
				Module:   "module.my_module[\"index\"]",
				Type:     "chainguard_group",
				Name:     "group-int-index",
				ID:       "group-int-index-id",
				IndexKey: 0,
				Attributes: map[string]interface{}{
					"id": "group-int-index-id",
				},
			},
		},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			got := FromState(tt.state, tt.provider)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Error("FromState() return mismatch (-want, +got):", diff)
			}
		})
	}
}
