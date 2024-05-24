package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cmdpdx/tf-state-import/pkg/resources"
	"github.com/cmdpdx/tf-state-import/pkg/state"
)

func main() {
	tfstate := flag.String("tfstate", "terraform.tfstate", "tfstate file to create import statements from. If empty, looks in the current directory for 'terraform.tfstate'")
	script := flag.String("out", "", "Import script name to produce. If empty, prints to stdout.")
	includeRemove := flag.Bool("include-remove", true, "Include `terraform rm` statements to alter state in place.")
	provider := flag.String("provider", "", "Only include resources from the given provider. If empty, all resources will be included.")
	flag.Parse()

	rm, err := getResources(*tfstate, *provider)
	if err != nil {
		log.Fatal(err)
	}

	ordered := rm.Order()

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

func output(out io.Writer, resources []*resources.Tuple, includeRemove bool) error {
	var removes []string
	if includeRemove {
		removes = make([]string, len(resources))
		for i, r := range resources {
			removes[len(removes)-1-i] = fmt.Sprintf("terraform state rm '%s'", r.Address())
		}
		_, err := out.Write([]byte(strings.Join(removes, "\n") + "\n"))
		if err != nil {
			return err
		}
	}

	imports := make([]string, len(resources))
	for i, r := range resources {
		imports[i] = fmt.Sprintf("terraform import '%s' %s", r.Address(), r.ID)
	}
	_, err := out.Write([]byte(strings.Join(imports, "\n") + "\n"))
	return err
}

// getResources returns a map of resource name to ResourceTuple from the given file.
func getResources(filename, provider string) (resources.ResourceMap, error) {
	state, err := state.ParseStateFile(filename)
	if err != nil {
		return nil, err
	}

	rm := make(map[string]resources.Tuple, len(state.Resources))
	for _, r := range state.Resources {
		if r.Mode == "data" {
			continue
		}
		if provider != "" && r.Provider != provider {
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
			t := resources.Tuple{
				Type:         r.Type,
				Name:         r.Name,
				ID:           id,
				IndexKey:     inst.IndexKey,
				Dependencies: inst.Dependencies,
			}
			rm[t.Address()] = t
		}
	}

	return rm, nil
}
