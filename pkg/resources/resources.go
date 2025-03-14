package resources

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/cmdpdx/tf-state-import/pkg/state"
)

type ResourceMap map[string]Tuple

type Tuple struct {
	Module       string
	Type         string
	Name         string
	ID           string
	IndexKey     interface{}
	Dependencies []string
	Attributes   map[string]interface{}
}

// FromState returns a map of resource name to ResourceTuple from the given state struct.
func FromState(state state.V4, provider string) ResourceMap {
	rm := make(map[string]Tuple, len(state.Resources))
	for _, r := range state.Resources {
		if r.Mode == "data" {
			continue
		}
		if provider != "" && !strings.Contains(r.Provider, provider) {
			continue
		}
		for _, inst := range r.Instances {
			rawID, ok := inst.Attributes["id"]
			if !ok {
				// Resource doesn't have ID attribute
				continue
			}
			id, ok := rawID.(string)
			if !ok {
				// Resource ID wasn't a string
				continue
			}
			t := Tuple{
				Module:       r.Module,
				Type:         r.Type,
				Name:         r.Name,
				ID:           id,
				IndexKey:     inst.IndexKey,
				Dependencies: inst.Dependencies,
				Attributes:   inst.Attributes,
			}
			rm[t.Address()] = t
		}
	}

	return rm
}

// Address is the unique friendly name of a resource as [{Module}.]{Type}.{Name}.
// This address matches the format found in Dependencies.
// resources defined with `for_each` have an index key and are
// addressed as {Type}.{Name}["{Index key}"]
func (r Tuple) Address() string {
	a := fmt.Sprintf("%s.%s", r.Type, r.Name)
	if r.Module != "" {
		a = fmt.Sprintf("%s.%s", r.Module, a)
	}

	switch v := r.IndexKey.(type) {
	case int, float64:
		return fmt.Sprintf("%s[%v]", a, v)
	case string:
		return fmt.Sprintf("%s[\"%s\"]", a, v)
	default:
		return a
	}
}

// ImportableID returns the id as expected by terraform to import the resource.
// For most resources, this is just the id as listed in the state file.
// However, there are some special cases that can be handled here.
func (r Tuple) ImportableID() string {
	switch {
	case r.Type == "google_project_iam_member":
		// TODO: condition
		return fmt.Sprintf("%s %s %s", r.Attributes["project"], r.Attributes["role"], r.Attributes["member"])
	case strings.HasSuffix(r.Type, "_iam_member"):
		return fmt.Sprintf("%s %s %s", r.Attributes["name"], r.Attributes["role"], r.Attributes["member"])
	case r.Type == "google_storage_bucket_iam_binding":
		// TODO: condition
		//nolint:forcetypeassert // we know bucket is a string
		return fmt.Sprintf("%s %s", strings.TrimPrefix(r.Attributes["bucket"].(string), "b/"), r.Attributes["role"])
	default:
		return r.ID
	}
}

type resourceOrdering struct {
	m       map[string]Tuple
	ordered []*Tuple

	checking map[string]interface{}
	done     map[string]interface{}

	once sync.Once
	keys []string
}

// Order returns a slice of Tuples in order of least to most dependent
// resource.
func (rm *ResourceMap) Order() []*Tuple {
	ro := resourceOrdering{
		m: *rm,
	}
	return ro.order()
}

// order walks the dependencies of resources in a depth-first search to produce an ordered
// slice from least-dependent to most-dependent resource.
func (ro *resourceOrdering) order() []*Tuple {
	ro.done = make(map[string]interface{}, len(ro.m))
	ro.checking = make(map[string]interface{}, len(ro.m))
	ro.ordered = make([]*Tuple, 0, len(ro.m))

	// Order resources by name, by default.
	for _, key := range ro.getKeys() {
		ro.visit(ro.m[key])
	}

	return ro.ordered
}

func (ro *resourceOrdering) getKeys() []string {
	ro.once.Do(func() {
		ro.keys = maps.Keys(ro.m)
		sort.Strings(ro.keys)
	})
	return ro.keys
}

func (ro *resourceOrdering) collectionResources(address string) []Tuple {
	rs := make([]Tuple, 0, 4)
	re, err := regexp.Compile(fmt.Sprintf(`%s\[".+"\]`, address))
	if err != nil {
		log.Println("failed to compile regex, can't find collection resources:", address)
		return nil
	}

	for _, key := range ro.getKeys() {
		if re.MatchString(key) {
			rs = append(rs, ro.m[key])
		}
	}
	return rs
}

func (ro *resourceOrdering) visit(r Tuple) {
	if _, found := ro.done[r.Address()]; found {
		return
	}
	if _, found := ro.checking[r.Address()]; found {
		log.Fatalf("cycle detected at: %#v", r)
	}

	ro.checking[r.Address()] = struct{}{}

	for _, d := range r.Dependencies {
		// Skip data dependencies.
		if strings.HasPrefix(d, "data.") {
			continue
		}
		// Collections dependencies do not include their index key
		// Look for all resources matching `{d}\[".+"\]`
		// e.g if the dependency is my_resource.name, look for all resources
		// that match my_resource.name["key"]
		if _, ok := ro.m[d]; !ok {
			deps := ro.collectionResources(d)

			for _, dep := range deps {
				ro.visit(dep)
			}
		} else {
			ro.visit(ro.m[d])
		}
	}

	delete(ro.checking, r.Address())
	ro.done[r.Address()] = struct{}{}
	ro.ordered = append(ro.ordered, &r)
}
