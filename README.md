# argocd-hydrate

> Kubernetes manifest "Hydration" refers to the process of turning abstract or templated Kubernetes manifest files into complete, environment-specific manifests ready for deployment. This is a key concept in GitOps and Kubernetes configuration management practices.

This tool hydrates ArgoCD Applications into Kubernetes manifests based on ArgoCD Application CRD, and diffs two such renders into an HTML report. It supports applications with a single source, as well as multi-source applications. Useful for building Rendered Manifests GitOps pattern or previewing real unabstracted PR diffs.

## Features

- Load and parse ArgoCD Application CRDs from YAML file(s)
- Pulls all helm charts from remotes to local cache, so that subsequent runs are much faster
- Process Helm chart sources using the Helm Go SDK
- Process directory-based sources, with support for recursive traversal
- Output rendered manifests to a specified directory
- Diff two rendered trees into a self-contained HTML report (`argocd-hydrate diff`)

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

## Diffing two renders

`argocd-hydrate diff` compares two rendered trees, as the command above writes
them, and produces a self-contained HTML report: files grouped by application,
unified or split view, context expansion, filters by status, application,
environment, cluster and kind, intra-line change highlighting, dark mode, and
keyboard navigation. Each input is a directory or a zip archive of one. Only
`.yaml`/`.yml` files take part.

```bash
argocd-hydrate diff --base base-manifests --head head-manifests --output diff.html \
  --repo hostinger/kubernetes-apps --pr-number 42 --pr-title "Bump nginx" \
  --base-ref master --head-ref feat/nginx --author alice
```

In GitHub Actions, `--repo`, `--base-ref` and `--head-ref` default to
`GITHUB_REPOSITORY`, `GITHUB_BASE_REF` and `GITHUB_HEAD_REF`, and `--pr-number`
to the number in `GITHUB_REF` on `pull_request` events (or to `GITHUB_PR_NUMBER`
if your workflow exports it), so only `--base` and `--head` are needed.
`--context` sets the unchanged lines kept around a change (default 3) and
`--full-text-limit` the largest modified file whose whole text is embedded so
context can be expanded in the viewer (default 2000 lines).

The report embeds the diff as JSON and renders it in the browser, so a page
with hundreds of changed manifests stays small and only draws what is on
screen. Secrets are already redacted by the hydrate step.

`--summary <file>` also writes a Markdown summary for a pull request comment:
a totals line ("11 manifests changed: 4 added, 6 modified, 1 deleted") and a
table with one row per application, environment and cluster, collapsed when
it has more than eight rows. A pull request that renders to no change says so
in one line.

## Smoke test

`make smoke` hydrates the example under [`test/smoke/`](test/smoke/) for a base
and a head commit and diffs them into `test/smoke/out/example/diff.html`. See
[test/smoke/README.md](test/smoke/README.md) for using it with your own
scenarios.

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

This project is licensed under the [Apache License](LICENSE).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
