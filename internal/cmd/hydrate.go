package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-diff/internal/application"
	"github.com/kazysgurskas/argocd-diff/internal/helm"
	"github.com/kazysgurskas/argocd-diff/internal/hydrate"
)

// defaultChartsDir is where pulled charts are kept unless --charts-dir says
// otherwise.
const defaultChartsDir = "cache"

type hydrateOptions struct {
	applications string
	output       string
	chartsDir    string
	kubeVersion  string
}

func newHydrateCommand() *cobra.Command {
	o := &hydrateOptions{}
	cmd := &cobra.Command{
		Use:   "hydrate",
		Short: "Render ArgoCD applications into Kubernetes manifests",
		Long: `Render ArgoCD Application resources into the Kubernetes manifests they would
deploy. Helm chart sources are pulled and templated, directory sources are read
as they are, and Secret values are redacted. Each application is written to
<output>/<application>/<namespace>/<Kind>/<name>.yaml, or
<output>/<application>/<Kind>/<name>.yaml for a cluster-scoped object.`,
		Example: `  argocd-diff hydrate
  argocd-diff hydrate --applications apps/applications.yaml --output rendered --kube-version 1.35.6`,
		RunE: func(*cobra.Command, []string) error {
			return runHydrate(o)
		},
	}
	addHydrateFlags(cmd, o)
	return cmd
}

func addHydrateFlags(cmd *cobra.Command, o *hydrateOptions) {
	f := cmd.Flags()
	f.StringVar(&o.applications, "applications", "manifests/applications.yaml",
		"Path to the file containing ArgoCD Application CRDs")
	f.StringVar(&o.output, "output", "manifests",
		"Output directory for the rendered manifests")
	f.StringVar(&o.chartsDir, "charts-dir", defaultChartsDir,
		"Directory for storing downloaded Helm charts")
	f.StringVar(&o.kubeVersion, "kube-version", helm.DefaultKubeVersion,
		"Kubernetes version to use for rendering Helm charts")
}

func runHydrate(o *hydrateOptions) error {
	_, err := hydrateTree(o.applications, o.output, hydrate.Options{ChartsDir: o.chartsDir, KubeVersion: o.kubeVersion})
	return err
}

// hydrateTree renders every Application in the file at applications and
// writes each manifest to out/<application>/<manifest path>. It returns the
// Applications it read.
func hydrateTree(applications, out string, opts hydrate.Options) ([]application.Application, error) {
	apps, err := application.Load(applications)
	if err != nil {
		return nil, fmt.Errorf("load applications: %w", err)
	}
	slog.Info("loaded applications", "path", applications, "count", len(apps))
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}

	for _, app := range apps {
		name := app.Metadata.Name
		slog.Info("hydrating application", "application", name)
		dir := filepath.Join(out, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		manifests, err := hydrate.Render(app, opts)
		if err != nil {
			return nil, fmt.Errorf("hydrate %s: %w", name, err)
		}
		if len(manifests) == 0 {
			slog.Warn("no manifests rendered", "application", name)
			continue
		}
		for _, m := range manifests {
			path := filepath.Join(dir, m.Path())
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(path, []byte(m.Content), 0o644); err != nil {
				return nil, err
			}
		}
		slog.Info("wrote manifests", "application", name, "count", len(manifests))
	}
	return apps, nil
}
