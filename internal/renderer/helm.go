package renderer

import (
	"fmt"
	"strings"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
	"github.com/kazysgurskas/argocd-hydrate/internal/helm"
)

// ProcessHelmChart processes a Helm chart source
func ProcessHelmChart(source *application.Source, appName, namespace string) (string, error) {
	releaseName := source.GetEffectiveReleaseName(appName)

	chartPath, err := helm.PullChart(source.RepoURL, source.Chart, source.TargetRevision)
	if err != nil {
		return "", err
	}

	var valueFilesPaths []string
	for _, valueFile := range source.Helm.ValueFiles {
		// Remove $values prefix if present
		if strings.HasPrefix(valueFile, "$values/") {
			valueFile = strings.Replace(valueFile, "$values/", "", 1)
		}
		valueFilesPaths = append(valueFilesPaths, valueFile)
	}

	fmt.Printf("Rendering chart %s (version %s) with release name %s in namespace %s\n",
		source.Chart, source.TargetRevision, releaseName, namespace)

	renderedManifest, err := helm.RenderHelmChart(chartPath, releaseName, namespace, source.TargetRevision, valueFilesPaths)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(renderedManifest), nil
}
