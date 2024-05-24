package resources

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"

	"golang.org/x/exp/maps"
)

type ResourceMap map[string]Tuple

type Tuple struct {
	Type         string
	Name         string
	ID           string
	IndexKey     interface{}
	Dependencies []string
}

// Address is the unique friendly name of a resource as {Type}.{Name}.
// This address matches the format found in Dependencies.
// resources defined with `for_each` have an index key and are
// addressed as {Type}.{Name}["{Index key}"]
func (r Tuple) Address() string {
	if r.IndexKey != nil {
		switch v := r.IndexKey.(type) {
		case int:
			return fmt.Sprintf("%s.%s[%d]", r.Type, r.Name, v)
		case string:
			return fmt.Sprintf("%s.%s[\"%s\"]", r.Type, r.Name, v)
		default:
			return fmt.Sprintf("%s.%s", r.Type, r.Name)
		}
	}
	return fmt.Sprintf("%s.%s", r.Type, r.Name)
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
