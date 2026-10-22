package diff

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
)

// Application is what the report shows about the ArgoCD Application a
// manifest was rendered for.
type Application struct {
	// Project is the Application's spec.project.
	Project string
	// Cluster is the Application's spec.destination.name.
	Cluster string
}

// Applications indexes Applications by metadata.name, which is also the
// directory argocd-hydrate writes each Application's manifests to.
type Applications map[string]Application

// LoadApplications reads the ArgoCD Applications in the YAML stream at path
// and indexes them by name.
func LoadApplications(path string) (Applications, error) {
	loaded, err := application.Load(path)
	if err != nil {
		return nil, err
	}
	apps := make(Applications, len(loaded))
	for _, app := range loaded {
		apps[app.Metadata.Name] = Application{Project: app.Spec.Project, Cluster: app.Spec.Destination.Name}
	}
	return apps, nil
}

// Merge returns the union of apps and other; other wins for a name both
// hold.
func (apps Applications) Merge(other Applications) Applications {
	out := make(Applications, len(apps)+len(other))
	for name, app := range apps {
		out[name] = app
	}
	for name, app := range other {
		out[name] = app
	}
	return out
}

// Resource identifies the Kubernetes object a manifest file holds, and the
// ArgoCD application it was rendered for.
type Resource struct {
	// App is the ArgoCD Application the manifest was rendered for: the
	// first segment of its path, which is the Application's metadata.name.
	App string `json:"app,omitempty"`
	// Project and Cluster are the Application's spec.project and
	// spec.destination.name, when the Application is known.
	Project    string `json:"project,omitempty"`
	Cluster    string `json:"cluster,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name,omitempty"`
}

// resourceFor describes the manifest at path with the given content. The
// path layout is the one argocd-hydrate writes:
//
//	<application>/<Kind>/<name>.yaml                 cluster-scoped
//	<application>/<namespace>/<Kind>/<name>.yaml     namespaced
//
// The manifest body is authoritative for apiVersion, kind, namespace and
// name when it parses; the path fills in what the body lacks. apps supplies
// the project and cluster of the Application named by the first segment.
func resourceFor(path, content string, apps Applications) *Resource {
	r := &Resource{}
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		r.App = parts[0]
		if app, ok := apps[parts[0]]; ok {
			r.Project, r.Cluster = app.Project, app.Cluster
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
