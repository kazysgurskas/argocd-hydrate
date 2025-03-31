package util

import (
	"os"
	"strings"

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

// SanitizeFileName makes sure a string is valid as a filename
func SanitizeFileName(name string) string {
	// Replace characters that might not be valid in filenames
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// GetNestedString safely extracts a string value from a nested map
func GetNestedString(obj map[string]interface{}, key string) (string, bool) {
	val, ok := obj[key]
	if !ok {
		return "", false
	}

	str, ok := val.(string)
	return str, ok
}
