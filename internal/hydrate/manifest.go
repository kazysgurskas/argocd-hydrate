package hydrate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest is one Kubernetes object rendered from an application.
type Manifest struct {
	Kind      string
	Name      string
	Namespace string
	// Content is the object's YAML document, starting with "---" and ending
	// with a newline.
	Content string
}

// Path is where the manifest is written below its application's directory:
// <namespace>/<Kind>/<name>.yaml, or <Kind>/<name>.yaml for a cluster-scoped
// object. Characters that cannot appear in a file name are replaced with
// "-".
func (m Manifest) Path() string {
	return filepath.Join(fileName(m.Namespace), m.Kind, fileName(m.Name)+".yaml")
}

var unsafeFileNameChars = strings.NewReplacer(":", "-", "/", "-", "\\", "-", ".", "-", " ", "-")

func fileName(name string) string {
	return unsafeFileNameChars.Replace(name)
}

// redactedValue stands in for every Secret value.
const redactedValue = "***REDACTED***"

// documentSeparator matches the "---" that starts a YAML document.
var documentSeparator = regexp.MustCompile(`(?m)^---`)

// parseManifests splits a YAML stream into manifests. Documents without a
// kind, or without a name or generateName, are skipped. Secret values are
// redacted; every other document is kept as written.
func parseManifests(stream string) ([]Manifest, error) {
	var manifests []Manifest
	for _, doc := range documentSeparator.Split(stream, -1) {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var obj struct {
			Kind     string `yaml:"kind"`
			Metadata struct {
				Name         string `yaml:"name"`
				GenerateName string `yaml:"generateName"`
				Namespace    string `yaml:"namespace"`
			} `yaml:"metadata"`
		}
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			return nil, fmt.Errorf("parse manifest: %w", err)
		}
		name := obj.Metadata.Name
		if name == "" && obj.Metadata.GenerateName != "" {
			name = obj.Metadata.GenerateName + "generated"
		}
		if obj.Kind == "" || name == "" {
			continue
		}
		if obj.Kind == "Secret" {
			var err error
			if doc, err = redactSecret(doc); err != nil {
				return nil, err
			}
		}
		manifests = append(manifests, Manifest{
			Kind:      obj.Kind,
			Name:      name,
			Namespace: obj.Metadata.Namespace,
			Content:   "---\n" + doc + "\n",
		})
	}
	return manifests, nil
}

// redactSecret replaces every value under data and stringData with
// redactedValue, keeping the keys, and returns the document re-marshalled.
func redactSecret(doc string) (string, error) {
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
		return "", fmt.Errorf("parse secret: %w", err)
	}
	for _, field := range []string{"data", "stringData"} {
		values, ok := obj[field].(map[string]any)
		if !ok {
			continue
		}
		redacted := make(map[string]any, len(values))
		for key := range values {
			redacted[key] = redactedValue
		}
		obj[field] = redacted
	}
	out, err := yaml.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshal secret: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
