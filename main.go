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
	outFile := flag.String("out", "", "Name of the output file to write results to. If empty, prints to stdout.")
	includeRemove := flag.Bool("include-remove", true, "Include `terraform rm` statements to alter state in place.")
	provider := flag.String("provider", "", "Filter resources by the given provider string, including partial matches. If empty, all resources will be included.")
	format := flag.String("format", "command", "How to structure the output, one of 'command' or 'block'. 'block' implies include-remove=false")
	flag.Parse()

	rm, err := getResources(*tfstate, *provider)
	if err != nil {
		log.Fatal(err)
	}

	if *format == "block" && *includeRemove {
		log.Println("format=block implies includeRemove=false...")
		*includeRemove = false
	}

	ordered := rm.Order()

	// Create the output file, if needed. Or pass stdout.
	var out io.Writer
	switch {
	case *outFile != "":
		if !strings.HasSuffix(*outFile, ".sh") {
			*outFile += ".sh"
		}
		f, err := os.Create(*outFile)
		if err != nil {
			log.Fatalf("failed to create script file %s: %s", *outFile, err)
		}
		out = f
	default:
		out = os.Stdout
	}

	err = output(out, ordered, *includeRemove, *format)
	if err != nil {
		log.Fatal(err)
	}

	if *outFile != "" {
		fmt.Fprintf(os.Stderr, "Results written to: %s\n", *outFile)
	}
}

func output(out io.Writer, resources []*resources.Tuple, includeRemove bool, format string) error {
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
		imports[i] = generateImport(r.Address(), r.ID, format)
	}
	_, err := out.Write([]byte(strings.Join(imports, "\n") + "\n"))
	return err
}

const importBlock = `import {
  to = %s
  id = "%s"
}`

func generateImport(address, id, format string) string {
	switch format {
	case "block":
		return fmt.Sprintf(importBlock, address, id)
	default:
		return fmt.Sprintf("terraform import '%s' %s", address, id)
	}
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
