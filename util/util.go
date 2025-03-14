package util

import (
	"os"

	"gopkg.in/yaml.v3"
)

// ReadValuesFile reads a YAML values file and returns it as a map
func ReadValuesFile(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	values := make(map[string]interface{})
	err = yaml.Unmarshal(data, &values)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// MergeMaps merges two maps together, with the second map taking precedence
func MergeMaps(dest, src map[string]interface{}) {
	for k, v := range src {
		if destMap, ok := dest[k].(map[string]interface{}); ok {
			if srcMap, ok := v.(map[string]interface{}); ok {
				MergeMaps(destMap, srcMap)
				continue
			}
		}
		dest[k] = v
	}
}
