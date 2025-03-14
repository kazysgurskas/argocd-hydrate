package application

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Application represents the ArgoCD Application CRD
type Application struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Destination struct {
			Namespace string `yaml:"namespace"`
		} `yaml:"destination"`
		Source  *Source   `yaml:"source,omitempty"`
		Sources []*Source `yaml:"sources,omitempty"`
	} `yaml:"spec"`
}

// Source represents a source configuration in an ArgoCD Application
type Source struct {
	RepoURL        string          `yaml:"repoURL"`
	Chart          string          `yaml:"chart,omitempty"`
	TargetRevision string          `yaml:"targetRevision"`
	Path           string          `yaml:"path,omitempty"`
	Ref            string          `yaml:"ref,omitempty"`
	Helm           HelmSource      `yaml:"helm,omitempty"`
	Directory      *DirectorySource `yaml:"directory,omitempty"`
}

// HelmSource represents Helm-specific configuration
type HelmSource struct {
	ReleaseName string   `yaml:"releaseName,omitempty"`
	ValueFiles  []string `yaml:"valueFiles,omitempty"`
	Values      string   `yaml:"values,omitempty"`
}

// DirectorySource represents directory source settings
type DirectorySource struct {
	Recurse bool `yaml:"recurse,omitempty"`
}

// LoadApplications loads and parses ArgoCD Application CRDs from a file
func LoadApplications(path string) ([]Application, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("applications file %s not found: %w", path, err)
	}

	// Split the file by YAML document separator
	documents := strings.Split(string(content), "---")
	applications := []Application{}

	for _, doc := range documents {
		if strings.TrimSpace(doc) == "" {
			continue
		}

		var manifest Application
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		if manifest.Kind == "Application" && strings.HasPrefix(manifest.APIVersion, "argoproj.io/") {
			applications = append(applications, manifest)
		}
	}

	return applications, nil
}

// GetEffectiveNamespace returns the effective namespace for the application
func (app *Application) GetEffectiveNamespace() string {
	namespace := app.Spec.Destination.Namespace
	if namespace == "" {
		namespace = app.Metadata.Name
	}
	return namespace
}

// GetSources returns all sources for the application
func (app *Application) GetSources() []*Source {
	var sources []*Source
	if app.Spec.Source != nil {
		sources = append(sources, app.Spec.Source)
	} else {
		sources = app.Spec.Sources
	}
	return sources
}

// IsValueSource returns true if this source is only used for reference values
func (s *Source) IsValueSource() bool {
	return s.Ref == "values"
}

// IsHelmChart returns true if this source is a Helm chart
func (s *Source) IsHelmChart() bool {
	return s.Chart != ""
}

// IsDirectory returns true if this source is a directory
func (s *Source) IsDirectory() bool {
	return s.Directory != nil
}

// GetEffectiveReleaseName returns the release name to use, defaulting to the application name if not specified
func (s *Source) GetEffectiveReleaseName(defaultName string) string {
	if s.Helm.ReleaseName != "" {
		return s.Helm.ReleaseName
	}
	return defaultName
}

// ShouldRecurseDirectory returns true if the directory should be recursively processed
func (s *Source) ShouldRecurseDirectory() bool {
	return s.Directory != nil && s.Directory.Recurse
}
