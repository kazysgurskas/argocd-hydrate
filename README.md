# argocd-hydrate

> Kubernetes manifest "Hydration" refers to the process of turning abstract or templated Kubernetes manifest files into complete, environment-specific manifests ready for deployment. This is a key concept in GitOps and Kubernetes configuration management practices.

This tool hydrates ArgoCD Applications into Kubernetes manifests based on ArgoCD Application CRD. It supports applications with a single source, as well as multi-source applications. Useful for building Rendered Manifests GitOps pattern or previewing real unabstracted PR diffs.

## Features

- Load and parse ArgoCD Application CRDs from YAML file(s)
- Pulls all helm charts from remotes to local cache, so that subsequent runs are much faster
- Process Helm chart sources using the Helm Go SDK
- Process directory-based sources, with support for recursive traversal
- Output rendered manifests to a specified directory

## Usage

```bash
~ argocd-hydrate  --help
ArgoCD Hydrate - Render ArgoCD Applications into Kubernetes manifests

This tool takes ArgoCD Application custom resources and renders Kubernetes manifests. It supports applications that use Helm charts
and directory-based sources.

Usage:
  argocd-hydrate [flags]

Examples:
  # Use default values
  argocd-hydrate

  # Specify custom applications file and output directory
  argocd-hydrate --applications=apps/applications.yaml --output=rendered

  # Specify custom charts directory
  argocd-hydrate --charts-dir=/path/to/charts

Flags:
      --applications string   Path to the file containing ArgoCD Application CRDs (default "manifests/applications.yaml")
      --charts-dir string     Directory for storing downloaded Helm charts (default "charts")
  -h, --help                  help for argocd-hydrate
      --output string         Output directory for the rendered manifests (default "manifests")
  -v, --version               version for argocd-hydrate
```

## Local Development and Testing

To test locally:

1. Clone the repository
2. Build the Docker image:
   ```bash
   docker build -t argocd-hydrate .
   ```
3. Run the image:
   ```bash
   docker run -v $(pwd):/workspace argocd-hydrate --applications=/workspace/manifests/applications.yaml --output=/workspace/hydrated-manifests
   ```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
