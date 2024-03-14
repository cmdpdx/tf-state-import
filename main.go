package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"golang.org/x/exp/maps"
)

func main() {
	tfstate := flag.String("tfstate", "terraform.tfstate", "tfstate file to create import statements from. If empty, looks in the current directory for 'terraform.tfstate'")
	script := flag.String("out", "", "Import script name to produce. If empty, prints to stdout.")
	includeRemove := flag.Bool("include-remove", true, "Include `terraform rm` statements to alter state in place.")
	flag.Parse()

	resources, err := resources(*tfstate)
	if err != nil {
		log.Fatal(err)
	}

	ordered := order(resources)

	// Create the output file, if needed. Or pass stdout.
	var out io.Writer
	switch {
	case *script != "":
		if !strings.HasSuffix(*script, ".sh") {
			*script += ".sh"
		}
		f, err := os.Create(*script)
		if err != nil {
			log.Fatalf("failed to create script file %s: %s", *script, err)
		}
		out = f
	default:
		out = os.Stdout
	}

	err = output(out, ordered, *includeRemove)
	if err != nil {
		log.Fatal(err)
	}

	if *script != "" {
		fmt.Fprintf(os.Stderr, "Import script created: %s\n", *script)
	}
}

type resourceOrdering struct {
	m       map[string]resourceTuple
	ordered []*resourceTuple

	checking map[string]interface{}
	done     map[string]interface{}

	once sync.Once
	keys []string
}

func order(resources map[string]resourceTuple) []*resourceTuple {
	ro := resourceOrdering{
		m: resources,
	}
	return ro.order()
}

// order walks the dependencies of resources in a depth-first search to produce an ordered
// slice from least-dependent to most-dependent resource.
func (ro *resourceOrdering) order() []*resourceTuple {
	ro.done = make(map[string]interface{}, len(ro.m))
	ro.checking = make(map[string]interface{}, len(ro.m))
	ro.ordered = make([]*resourceTuple, 0, len(ro.m))

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

func (ro *resourceOrdering) collectionResources(address string) []resourceTuple {
	rs := make([]resourceTuple, 0, 4)
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

func (ro *resourceOrdering) visit(r resourceTuple) {
	if _, found := ro.done[r.address()]; found {
		return
	}
	if _, found := ro.checking[r.address()]; found {
		log.Fatalf("cycle detected at: %#v", r)
	}

	ro.checking[r.address()] = struct{}{}

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

	delete(ro.checking, r.address())
	ro.done[r.address()] = struct{}{}
	ro.ordered = append(ro.ordered, &r)
}

type state struct {
	Resources []resource
}

type resource struct {
	Mode      string
	Type      string
	Name      string
	Provider  string
	Instances []resourceInstance
}

type resourceInstance struct {
	Attributes   map[string]interface{}
	Dependencies []string
	IndexKey     interface{} `json:"index_key"`
}

type resourceTuple struct {
	Type         string
	Name         string
	ID           string
	IndexKey     interface{}
	Dependencies []string
}

// address is the unique friendly name of a resource as {Type}.{Name}.
// This address matches the format found in Dependencies.
// resources defined with `for_each` have an index key and are
// addressed as {Type}.{Name}["{Index key}"]
func (r resourceTuple) address() string {
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

func output(out io.Writer, resources []*resourceTuple, includeRemove bool) error {
	var removes []string
	if includeRemove {
		removes = make([]string, len(resources))
		for i, r := range resources {
			removes[len(removes)-1-i] = fmt.Sprintf("terraform state rm '%s'", r.address())
		}
		_, err := out.Write([]byte(strings.Join(removes, "\n") + "\n"))
		if err != nil {
			return err
		}
	}

	imports := make([]string, len(resources))
	for i, r := range resources {
		imports[i] = fmt.Sprintf("terraform import '%s' %s", r.address(), r.ID)
	}
	_, err := out.Write([]byte(strings.Join(imports, "\n") + "\n"))
	return err
}

func parseStateFile(filename string) (state, error) {
	bs, err := os.ReadFile(filename)
	if err != nil {
		return state{}, err
	}

	var s state
	err = json.Unmarshal(bs, &s)
	if err != nil {
		return state{}, err
	}
	return s, err
}

// resources returns a map of resource name to resourceTuple from the given file.
func resources(filename string) (map[string]resourceTuple, error) {
	state, err := parseStateFile(filename)
	if err != nil {
		return nil, err
	}

	resources := make(map[string]resourceTuple, len(state.Resources))
	for _, r := range state.Resources {
		if r.Mode == "data" {
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
			t := resourceTuple{
				Type:         r.Type,
				Name:         r.Name,
				ID:           id,
				IndexKey:     inst.IndexKey,
				Dependencies: inst.Dependencies,
			}
			resources[t.address()] = t
		}
	}

	return resources, nil
}
