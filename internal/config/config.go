package config

// Configuration holds the hydrate command's settings.
type Configuration struct {
	// ApplicationsFile is the path to the file containing ArgoCD Application CRDs
	ApplicationsFile string

	// OutputDir is the output directory for the rendered manifests
	OutputDir string

	// ChartsDir is the directory for storing downloaded Helm charts
	ChartsDir string

	// KubeVersion is the Kubernetes version to use for rendering Helm charts
	KubeVersion string
}

// DefaultKubeVersion is the Kubernetes version charts are rendered for unless
// --kube-version says otherwise.
const DefaultKubeVersion = "1.35.6"

var instance *Configuration

// GetConfig returns the current configuration, initializing it if needed
func GetConfig() *Configuration {
	if instance == nil {
		instance = NewConfig()
	}
	return instance
}

// SetConfig replaces the current configuration instance
func SetConfig(config *Configuration) {
	instance = config
}

// NewConfig creates a new configuration with default values
func NewConfig() *Configuration {
	return &Configuration{
		ApplicationsFile: "manifests/applications.yaml",
		OutputDir:        "manifests",
		ChartsDir:        "cache",
		KubeVersion:      DefaultKubeVersion,
	}
}
