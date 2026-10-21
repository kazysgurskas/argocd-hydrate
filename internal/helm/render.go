package helm

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

// DefaultKubeVersion is the Kubernetes version charts are rendered for
// unless the caller says otherwise.
const DefaultKubeVersion = "1.35.6"

// Render templates the chart at chartPath as release in namespace for
// kubeVersion (X.Y.Z, with or without a leading v), with valueFiles merged
// in order so that later files win. It returns the manifest stream, CRDs
// included, as `helm template --include-crds` would.
func Render(chartPath, release, namespace, kubeVersion string, valueFiles []string) (string, error) {
	kube, err := chartutil.ParseKubeVersion(kubeVersion)
	if err != nil {
		return "", fmt.Errorf("kubernetes version %q is not of the form X.Y.Z: %w", kubeVersion, err)
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("load chart %s: %w", chartPath, err)
	}
	if constraint := chart.Metadata.KubeVersion; constraint != "" && !chartutil.IsCompatibleRange(constraint, kube.Version) {
		return "", fmt.Errorf("chart %s requires kubernetes %s, rendering for %s; pass --kube-version", chart.Name(), constraint, kube.Version)
	}
	values := map[string]any{}
	for _, file := range valueFiles {
		fileValues, err := readValues(file)
		if err != nil {
			return "", err
		}
		mergeValues(values, fileValues)
	}

	install := action.NewInstall(new(action.Configuration))
	install.DryRun = true
	install.ClientOnly = true
	install.IncludeCRDs = true
	install.ReleaseName = release
	install.Namespace = namespace
	install.KubeVersion = kube
	slog.Info("rendering chart", "chart", chart.Name(), "version", chart.Metadata.Version,
		"release", release, "namespace", namespace, "kube_version", kube.Version)
	rel, err := install.Run(chart, values)
	if err != nil {
		return "", fmt.Errorf("render chart %s: %w", chart.Name(), err)
	}
	return rel.Manifest, nil
}

// readValues reads a YAML values file.
func readValues(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]any{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parse values file %s: %w", path, err)
	}
	return values, nil
}

// mergeValues merges src into dst, map by map, with src winning where both
// set a key.
func mergeValues(dst, src map[string]any) {
	for k, v := range src {
		if dstMap, ok := dst[k].(map[string]any); ok {
			if srcMap, ok := v.(map[string]any); ok {
				mergeValues(dstMap, srcMap)
				continue
			}
		}
		dst[k] = v
	}
}
