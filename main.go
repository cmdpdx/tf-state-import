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
	includeRemove := flag.Bool("include-remove", true, "Include `terraform rm` statements to alter state in place.")
	provider := flag.String("provider", "", "Filter resources by the given provider string, including partial matches. If empty, all resources will be included.")
	format := flag.String("format", "command", "How to structure the output, one of 'command' or 'block'. 'block' implies include-remove=false")
	flag.Parse()

	if *format == "block" && *includeRemove {
		log.Println("format=block implies includeRemove=false...")
		*includeRemove = false
	}

	st, err := state.ParseStateFile(*tfstate)
	if err != nil {
		log.Fatal(err)
	}

	rm := resources.FromState(st, *provider)
	ordered := rm.Order()

	err = output(os.Stdout, ordered, *includeRemove, *format)
	if err != nil {
		log.Fatal(err)
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
		imports[i] = generateImport(r.Address(), r.ImportableID(), format)
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
