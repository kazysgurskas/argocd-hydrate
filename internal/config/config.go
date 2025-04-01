package config

// Configuration holds all application configuration
type Configuration struct {
	// ApplicationsFile is the path to the file containing ArgoCD Application CRDs
	ApplicationsFile string

	// OutputDir is the output directory for the rendered manifests
	OutputDir string

	// ChartsDir is the directory for storing downloaded Helm charts
	ChartsDir string
}

// DefaultConfig returns the current configuration with default values
func GetConfig() *Configuration {
	return &Configuration{
		ApplicationsFile: "manifests/applications.yaml",
		OutputDir:        "manifests",
		ChartsDir:        "charts",
	}
}
