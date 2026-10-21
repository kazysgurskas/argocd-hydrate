// Package helm pulls charts and renders them with the Helm SDK, as
// `helm pull` and `helm template` would.
package helm

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// Pull downloads chart name at version from repoURL into
// chartsDir/<version>/<name> and returns that path. A chart already there
// is not downloaded again. repoURL is an HTTP(S) chart repository or an OCI
// registry path, with or without the oci:// scheme.
func Pull(chartsDir, repoURL, name, version string) (string, error) {
	dir := filepath.Join(chartsDir, version)
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		slog.Info("chart already pulled", "chart", name, "version", version, "path", path)
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	// Helm unpacks into a scratch directory beside the destination, and the
	// chart is then renamed into place. Unpacking straight into dir would
	// leave an empty directory named after the archive next to the chart
	// when the chart comes from an http(s) repository.
	scratch, err := os.MkdirTemp(dir, ".pull-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	untarDir, err := filepath.Abs(scratch)
	if err != nil {
		return "", err
	}

	pull := action.NewPullWithOpts(action.WithConfig(new(action.Configuration)))
	pull.Settings = cli.New()
	pull.Version = version
	pull.Untar = true
	pull.UntarDir = untarDir

	ref := name
	switch {
	case isOCI(repoURL):
		ref = "oci://" + strings.TrimPrefix(repoURL, "oci://") + "/" + name
	case strings.HasPrefix(repoURL, "https://") || strings.HasPrefix(repoURL, "http://"):
		pull.RepoURL = repoURL
	default:
		return "", fmt.Errorf("chart repository %q is neither an http(s) URL nor an OCI registry", repoURL)
	}
	slog.Info("pulling chart", "repo", repoURL, "chart", name, "version", version)
	if _, err := pull.Run(ref); err != nil {
		return "", fmt.Errorf("pull %s %s from %s: %w", name, version, repoURL, err)
	}
	if err := os.Rename(filepath.Join(scratch, name), path); err != nil {
		return "", fmt.Errorf("move %s into place: %w", name, err)
	}
	return path, nil
}

// isOCI reports whether repoURL names an OCI registry: an oci:// URL, or a
// bare host and path such as ghcr.io/org/charts, which is how ArgoCD
// records registries.
func isOCI(repoURL string) bool {
	if strings.HasPrefix(repoURL, "oci://") {
		return true
	}
	if strings.HasPrefix(repoURL, "https://") || strings.HasPrefix(repoURL, "http://") {
		return false
	}
	host, _, _ := strings.Cut(repoURL, "/")
	return strings.Contains(host, ".")
}
