package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
)

func main() {
	tfstate := flag.String("tfstate", "terraform.tfstate", "tfstate file to create import statements from. If empty, looks in the current directory for 'terraform.tfstate'")
	script := flag.String("out", "", "Import script name to produce. If empty, prints to stdout.")
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

	err = output(out, ordered)
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
}

func order(resources map[string]resourceTuple) []*resourceTuple {
	ro := resourceOrdering{
		m: resources,
	}
	return ro.order()
}

// order walks the dependencies of resources in a DFS search to produce an ordered
// slice from least-dependent to most-dependent resource. 
func (rm *resourceOrdering) order() []*resourceTuple {
	done := make(map[string]interface{}, len(rm.m))
	checking := make(map[string]interface{}, len(rm.m))
	rm.ordered = make([]*resourceTuple, 0, len(rm.m))

	// Order resources by name, by default.
	keys := maps.Keys(rm.m)
	sort.Strings(keys)

	for _, key := range keys {
		rm.visit(rm.m[key], checking, done)
	}

	return rm.ordered
}

func (rm *resourceOrdering) visit(r resourceTuple, checking map[string]interface{}, done map[string]interface{}) {
	if _, found := done[r.identifier()]; found {
		return
	}
	if _, found := checking[r.identifier()]; found {
		log.Fatalf("cycle detected at: %#v", r)
	}

	checking[r.identifier()] = struct{}{}

	for _, d := range r.Dependencies {
		rm.visit(rm.m[d], checking, done)
	}

	delete(checking, r.identifier())
	done[r.identifier()] = struct{}{}
	rm.ordered = append(rm.ordered, &r)
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
}

type resourceTuple struct {
	Type         string
	Name         string
	ID           string
	Dependencies []string
}

// identifier is the unique friendly name of a resource as [Type].[Name].
// This identifier matches the format found in Dependencies.
func (r resourceTuple) identifier() string {
	return fmt.Sprintf("%s.%s", r.Type, r.Name)
}

func output(out io.Writer, resources []*resourceTuple) error {
	lines := make([]string, len(resources)*2)
	for i, r := range resources {
		lines[len(resources)-1-i] = fmt.Sprintf("terraform state rm %s.%s", r.Type, r.Name)
		lines[len(resources)+i] = fmt.Sprintf("terraform import %s.%s %s", r.Type, r.Name, r.ID)
	}

	_, err := out.Write([]byte(strings.Join(lines, "\n") + "\n"))
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
				Dependencies: inst.Dependencies,
			}
			resources[t.identifier()] = t
		}
	}

	return resources, nil
}
