package config

// Configuration holds all application configuration
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

// Private configuration instance
var instance *Configuration

// GetConfig returns the current configuration, initializing it if needed
func GetConfig() *Configuration {
	if instance == nil {
		// Initialize with default values
		instance = &Configuration{
			ApplicationsFile: "manifests/applications.yaml",
			OutputDir:        "manifests",
			ChartsDir:        "cache",
			KubeVersion:      "1.31.1", // Default Kubernetes version
		}
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
		KubeVersion:      "1.31.1", // Default Kubernetes version
	}
}
