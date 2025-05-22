package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/kazysgurskas/argocd-hydrate/internal/config"
	"github.com/kazysgurskas/argocd-hydrate/pkg/util"
)

// isValidKubeVersion checks if the given string is a valid Kubernetes version
func isValidKubeVersion(version string) bool {
	re := regexp.MustCompile(`^(\d+)\.(\d+)(\.(\d+))?(-[a-zA-Z0-9]+)?$`)
	return re.MatchString(version)
}

// isOCIURL checks if the URL is likely an OCI repository URL
func isOCIURL(url string) bool {
	if strings.HasPrefix(url, "oci://") {
		return true
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return false
	}

	parts := strings.Split(url, "/")
	if len(parts) >= 1 && strings.Contains(parts[0], ".") {
		return true
	}

	return false
}

// PullChart pulls a Helm chart from a repository using Helm Go packages
func PullChart(url, chartName, version string) (string, error) {
	config := config.GetConfig()

	// Use the configured charts directory instead of hardcoded value
	chartsDir := config.ChartsDir

	// Ensure the base charts directory exists
	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create base charts directory %s: %w", chartsDir, err)
	}

	chartDir := filepath.Join(chartsDir, version)
	chartPath := filepath.Join(chartDir, chartName)

	// Check if chart already exists
	if _, err := os.Stat(chartPath); err == nil {
		fmt.Printf("Chart %s (version %s) already exists at %s, skipping download.\n", chartName, version, chartPath)
		return chartPath, nil
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", chartDir, err)
	}

	// Initialize Helm settings
	settings := cli.New()

	// Create a new Helm pull action
	client := action.NewPullWithOpts(action.WithConfig(&action.Configuration{}))
	client.Settings = settings
	client.Version = version
	client.Untar = true
	client.DestDir = chartDir

	// Configure repository options
	repositoryCache := settings.RepositoryCache

	// Create the cache directory if it doesn't exist
	if err := os.MkdirAll(repositoryCache, 0755); err != nil {
		return "", fmt.Errorf("failed to create repository cache directory: %w", err)
	}

	// Determine if this is an OCI repository or an HTTP repository
	if isOCIURL(url) {
		// For OCI repositories, ensure the URL has the oci:// prefix
		ociURL := url
		if !strings.HasPrefix(ociURL, "oci://") {
			ociURL = "oci://" + ociURL
		}

		// The chart reference is the full OCI path + chart name
		chartRef := fmt.Sprintf("%s/%s", ociURL, chartName)
		fmt.Printf("Pulling chart from OCI repository: %s\n", chartRef)

		_, err := client.Run(chartRef)
		if err != nil {
			return "", fmt.Errorf("failed to download chart from OCI repository: %w", err)
		}
	} else if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		// For HTTP(S) repositories
		err := downloadHTTPSChart(url, chartName, repositoryCache, settings, client)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("unsupported repository URL format: %s", url)
	}

	fmt.Printf("Successfully pulled %s (version %s) to %s\n", chartName, version, chartPath)
	return chartPath, nil
}

// downloadHTTPSChart downloads a chart from an HTTPS repository
func downloadHTTPSChart(url, chartName, repositoryCache string, settings *cli.EnvSettings, client *action.Pull) error {
	// Generate a unique but consistent repo name based on the URL
	repoName := fmt.Sprintf("repo-%s", strings.ReplaceAll(url, "/", "-"))
	repoName = strings.ReplaceAll(repoName, ":", "-")
	repoName = strings.ReplaceAll(repoName, ".", "-")
	if len(repoName) > 63 {
		repoName = repoName[:63]
	}

	// Create a temporary repository entry
	repoEntry := repo.Entry{
		Name: repoName,
		URL:  url,
	}

	chartRef := fmt.Sprintf("%s/%s", repoEntry.Name, chartName)

	// Create a unique temporary repository config file just for this operation
	tempRepoFile := filepath.Join(os.TempDir(), fmt.Sprintf("helm-repo-%s.yaml", repoName))

	// Create a new empty repo file
	repoFile := repo.NewFile()
	repoFile.Add(&repoEntry)

	// Save the temporary repository file
	if err := repoFile.WriteFile(tempRepoFile, 0644); err != nil {
		return fmt.Errorf("failed to write temporary repository config: %w", err)
	}
	defer os.Remove(tempRepoFile) // Clean up when we're done

	// Set the client to use our temporary repo file
	client.Settings.RepositoryConfig = tempRepoFile

	// Initialize chart repository and download index
	chartRepo, err := repo.NewChartRepository(&repoEntry, getter.All(settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	chartRepo.CachePath = repositoryCache

	// Download the repository index
	_, err = chartRepo.DownloadIndexFile()
	if err != nil {
		return fmt.Errorf("failed to download repository index: %w", err)
	}

	// Download chart
	_, err = client.Run(chartRef)
	if err != nil {
		return fmt.Errorf("failed to download chart: %w", err)
	}

	return nil
}

// RenderHelmChart renders a Helm chart using the Helm Go library
func RenderHelmChart(chartPath, releaseName, namespace, version string, valueFiles []string) (string, error) {
	config := config.GetConfig()

	// Parse Kubernetes version
	kubeVersion := config.KubeVersion

	// Validate Kubernetes version format
	if !isValidKubeVersion(kubeVersion) {
		return "", fmt.Errorf("invalid Kubernetes version format: %s. Expected format: X.Y.Z", kubeVersion)
	}

	// Extract major and minor versions
	parts := strings.Split(kubeVersion, ".")
	major := "1"
	minor := "26"

	if len(parts) >= 2 {
		major = parts[0]
		minor = parts[1]

		// Remove any non-numeric suffixes from minor version
		for i, c := range minor {
			if c < '0' || c > '9' {
				minor = minor[:i]
				break
			}
		}
	}

	// Load the chart
	chartLoaded, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart %s: %w", chartPath, err)
	}

	// Initialize Helm action configuration
	actionConfig := new(action.Configuration)

	// Initialize Helm template action
	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Version = version
	client.ClientOnly = true
	client.IncludeCRDs = true
	client.KubeVersion = &chartutil.KubeVersion{
    Version: kubeVersion,
    Major:   major,
    Minor:   minor,
	}

	fmt.Printf("Using Kubernetes version %s for rendering chart %s\n", kubeVersion, chartPath)

	// Create values from files
	values := make(map[string]interface{})
	for _, valueFile := range valueFiles {
		currentValues, err := util.ReadValuesFile(valueFile)
		if err != nil {
			return "", fmt.Errorf("failed to read values file %s: %w", valueFile, err)
		}

		// Merge values
		util.MergeMaps(values, currentValues)
	}

	// Render the chart
	release, err := client.Run(chartLoaded, values)
	if err != nil {
		if strings.Contains(err.Error(), "kubeVersion") {
			fmt.Printf("Kubernetes version error detected. Chart requires: %s\n", chartLoaded.Metadata.KubeVersion)
			fmt.Printf("We're using Kubernetes version: %s\n", client.KubeVersion.String())
			fmt.Printf("Try using --kube-version flag to set a higher Kubernetes version\n")
		}
		return "", fmt.Errorf("failed to render chart: %w", err)
	}

	return release.Manifest, nil
}
