package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	tfstate := flag.String("tfstate", "", "tfstate file to create import statements from.")
	script := flag.String("out", "import.sh", "Import script name to produce. Default: import.sh")
	flag.Parse()
	if *tfstate == "" {
		log.Fatal("must pass tfstate file")
	}

	resources, err := resources(*tfstate)
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasSuffix(*script, ".sh") {
		*script += ".sh"
	}
	err = createScript(*script, resources)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Import script created:", *script)
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
	Attributes map[string]interface{}
}

type resourceTuple struct {
	Type string
	Name string
	ID   string
}

func createScript(name string, resources []resourceTuple) error {
	lines := make([]string, len(resources))
	for i, r := range resources {
		lines[i] = fmt.Sprintf("terraform import %s.%s %s", r.Type, r.Name, r.ID)
	}

	return os.WriteFile(name, []byte(strings.Join(lines, "\n")), 0744)
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

func resources(filename string) ([]resourceTuple, error) {
	state, err := parseStateFile(filename)
	if err != nil {
		return nil, err
	}

	var resources []resourceTuple
	for _, r := range state.Resources {
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
			resources = append(resources, resourceTuple{
				Type: r.Type,
				Name: r.Name,
				ID:   id,
			})
		}
	}
	return resources, nil
}
