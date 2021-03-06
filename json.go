package main

import (
	"io"
	"io/ioutil"
	"fmt"
	"encoding/json"
)

func jsonifyData(data []byte) (string, error) {
	doc, err := parseYAML(data)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(deinterface(doc))
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func JSONifyIO(in io.Reader) (string, error) {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("Error reading input: %s", err)
	}
	return jsonifyData(data)
}

func JSONifyFiles(paths []string) ([]string, error) {
	l := make([]string, len(paths))

	for i, path := range paths {
		DEBUG("Processing file '%s'", path)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("Error reading file %s: %s", path, err)
		}

		if l[i], err = jsonifyData(data); err != nil {
			return nil, fmt.Errorf("%s: %s", path, err)
		}
	}

	return l, nil
}

func deinterface(o interface{}) interface{} {
	switch o.(type) {
	case map[interface{}] interface{}:
		return deinterfaceMap(o.(map[interface{}] interface{}))
	case []interface{}:
		return deinterfaceList(o.([]interface{}))
	default:
		return o
	}
}

func deinterfaceMap(o map[interface{}] interface{}) map[string] interface{} {
	m := map[string] interface{} {}
	for k, v := range o {
		m[fmt.Sprintf("%v", k)] = deinterface(v)
	}
	return m
}

func deinterfaceList(o []interface{}) []interface{} {
	l := make([]interface{}, len(o))
	for i, v := range o {
		l[i] = deinterface(v)
	}
	return l
}
