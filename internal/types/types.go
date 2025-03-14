package types

// Configuration holds all application configuration
type Configuration struct {
	// ApplicationsFile is the path to the file containing ArgoCD Application CRDs
	ApplicationsFile string

	// OutputDir is the output directory for the rendered manifests
	OutputDir string

	// ChartsDir is the directory for storing downloaded Helm charts
	ChartsDir string
}

// Global configuration instance
var (
	// Config holds the current configuration
	Config = Configuration{
		ApplicationsFile: "manifests/applications.yaml",
		OutputDir:        "manifests",
		ChartsDir:        "charts",
	}
)
