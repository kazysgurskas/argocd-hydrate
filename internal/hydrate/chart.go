package hydrate

import (
	"strings"

	"github.com/kazysgurskas/argocd-diff/internal/application"
	"github.com/kazysgurskas/argocd-diff/internal/helm"
)

// renderChart pulls the source's chart and templates it. Value files are
// taken relative to the working directory; the $values/ prefix, which
// ArgoCD resolves against the values source, is dropped.
func renderChart(source *application.Source, release, namespace string, opts Options) (string, error) {
	path, err := helm.Pull(opts.ChartsDir, source.RepoURL, source.Chart, source.TargetRevision)
	if err != nil {
		return "", err
	}
	valueFiles := make([]string, 0, len(source.Helm.ValueFiles))
	for _, file := range source.Helm.ValueFiles {
		valueFiles = append(valueFiles, strings.TrimPrefix(file, "$values/"))
	}
	stream, err := helm.Render(path, release, namespace, opts.KubeVersion, valueFiles)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stream), nil
}
