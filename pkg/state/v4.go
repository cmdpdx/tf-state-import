package state

import (
	"encoding/json"
	"os"
)

type V4 struct {
	Resources []Resource
	Version   int
}

type Resource struct {
	Module    string
	Mode      string
	Type      string
	Name      string
	Provider  string
	Instances []Instance
}

type Instance struct {
	Attributes   map[string]interface{}
	Dependencies []string
	IndexKey     interface{} `json:"index_key"`
}

func ParseStateFile(filename string) (V4, error) {
	bs, err := os.ReadFile(filename)
	if err != nil {
		return V4{}, err
	}

	var s V4
	err = json.Unmarshal(bs, &s)

	return s, err
}
