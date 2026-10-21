package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kazysgurskas/argocd-hydrate/internal/application"
	"github.com/kazysgurskas/argocd-hydrate/internal/helm"
	"github.com/kazysgurskas/argocd-hydrate/internal/hydrate"
)

type hydrateOptions struct {
	applications string
	output       string
	chartsDir    string
	kubeVersion  string
}

// addHydrateFlags binds the hydrate flags to o. The root command and the
// hydrate subcommand share one set, so either form sets the same options.
func addHydrateFlags(cmd *cobra.Command, o *hydrateOptions) {
	f := cmd.Flags()
	f.StringVar(&o.applications, "applications", "manifests/applications.yaml",
		"Path to the file containing ArgoCD Application CRDs")
	f.StringVar(&o.output, "output", "manifests",
		"Output directory for the rendered manifests")
	f.StringVar(&o.chartsDir, "charts-dir", "cache",
		"Directory for storing downloaded Helm charts")
	f.StringVar(&o.kubeVersion, "kube-version", helm.DefaultKubeVersion,
		"Kubernetes version to use for rendering Helm charts")
}

// runHydrate renders every application in o.applications and writes each
// manifest to o.output/<application>/<manifest path>.
func runHydrate(o *hydrateOptions) error {
	applications, err := application.Load(o.applications)
	if err != nil {
		return fmt.Errorf("load applications: %w", err)
	}
	slog.Info("loaded applications", "path", o.applications, "count", len(applications))
	if err := os.MkdirAll(o.output, 0o755); err != nil {
		return err
	}

	opts := hydrate.Options{ChartsDir: o.chartsDir, KubeVersion: o.kubeVersion}
	for _, app := range applications {
		name := app.Metadata.Name
		slog.Info("hydrating application", "application", name)
		dir := filepath.Join(o.output, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		manifests, err := hydrate.Render(app, opts)
		if err != nil {
			return fmt.Errorf("hydrate %s: %w", name, err)
		}
		if len(manifests) == 0 {
			slog.Warn("no manifests rendered", "application", name)
			continue
		}
		for _, m := range manifests {
			path := filepath.Join(dir, m.Path())
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(m.Content), 0o644); err != nil {
				return err
			}
		}
		slog.Info("wrote manifests", "application", name, "count", len(manifests))
	}
	return nil
}
