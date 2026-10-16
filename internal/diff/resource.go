package diff

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Resource identifies the Kubernetes object a manifest file holds, and the
// ArgoCD application it was rendered for.
type Resource struct {
	// App, Environment and Cluster come from the application directory name
	// when it follows the <app>.<environment>.<cluster> convention; otherwise
	// App is the whole directory name.
	App         string `json:"app,omitempty"`
	Environment string `json:"environment,omitempty"`
	Cluster     string `json:"cluster,omitempty"`
	APIVersion  string `json:"apiVersion,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Name        string `json:"name,omitempty"`
}

// resourceFor describes the manifest at path with the given content. The
// path layout is the one argocd-hydrate writes:
//
//	<application>/<Kind>/<name>.yaml                 cluster-scoped
//	<application>/<namespace>/<Kind>/<name>.yaml     namespaced
//
// The manifest body is authoritative for apiVersion, kind, namespace and
// name when it parses; the path fills in what the body lacks.
func resourceFor(path, content string) *Resource {
	r := &Resource{}
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		appParts := strings.SplitN(parts[0], ".", 3)
		if len(appParts) == 3 {
			r.App, r.Environment, r.Cluster = appParts[0], appParts[1], appParts[2]
		} else {
			r.App = parts[0]
		}
		base := strings.TrimSuffix(strings.TrimSuffix(parts[len(parts)-1], ".yaml"), ".yml")
		switch len(parts) {
		case 3:
			r.Kind, r.Name = parts[1], base
		case 4:
			r.Namespace, r.Kind, r.Name = parts[1], parts[2], base
		}
	}
	if meta := parseManifest(content); meta != nil {
		if meta.APIVersion != "" {
			r.APIVersion = meta.APIVersion
		}
		if meta.Kind != "" {
			r.Kind = meta.Kind
		}
		if meta.Name != "" {
			r.Name = meta.Name
		}
		if meta.Namespace != "" {
			r.Namespace = meta.Namespace
		}
	}
	if *r == (Resource{}) {
		return nil
	}
	return r
}

// parseManifest reads the identifying fields of the first Kubernetes object
// in a YAML document stream.
func parseManifest(content string) *Resource {
	decoder := yaml.NewDecoder(strings.NewReader(content))
	for {
		var obj struct {
			APIVersion string `yaml:"apiVersion"`
			Kind       string `yaml:"kind"`
			Metadata   struct {
				Name      string `yaml:"name"`
				Namespace string `yaml:"namespace"`
			} `yaml:"metadata"`
		}
		if err := decoder.Decode(&obj); err != nil {
			return nil
		}
		if obj.Kind == "" && obj.APIVersion == "" {
			continue
		}
		return &Resource{
			APIVersion: obj.APIVersion,
			Kind:       obj.Kind,
			Name:       obj.Metadata.Name,
			Namespace:  obj.Metadata.Namespace,
		}
	}
}
