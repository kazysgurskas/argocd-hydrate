# argocd-diff

argocd-diff shows what a pull request to a GitOps repository does to the
Kubernetes manifests ArgoCD would apply. `argocd-diff hydrate` renders ArgoCD
Applications into a tree of plain manifests, the way ArgoCD's repo server
would, and `argocd-diff` compares two such trees, from the base and the head
of a pull request, into a self-contained HTML report and a Markdown summary
for a pull request comment. It supports Applications with a single source and
multi-source Applications, with Helm chart and directory sources. It was
called argocd-hydrate until v0.4.0.

## Diffing two renders

```console
$ argocd-diff --help
Compare two trees of rendered manifests, as "argocd-diff hydrate" writes
them, and write a self-contained HTML report of what changed. Each input is a
directory or a zip archive of one.

Pull request details are optional. Inside GitHub Actions the repository, base
and head refs come from GITHUB_REPOSITORY, GITHUB_BASE_REF and GITHUB_HEAD_REF,
and the number from GITHUB_REF on pull_request events (refs/pull/<n>/merge) or
from GITHUB_PR_NUMBER if the workflow exports it, so only --base and --head are
needed there.

Usage:
  argocd-diff [flags]
  argocd-diff [command]

Examples:
  # Render the two trees to compare, from each checkout
  argocd-diff hydrate --applications manifests/applications.yaml --output base-manifests
  argocd-diff hydrate --applications manifests/applications.yaml --output head-manifests

  argocd-diff --base base-manifests --head head-manifests
  argocd-diff --base base-manifests --head head-manifests --summary diff.md
  argocd-diff --base base.zip --head head.zip --output diff.html \
      --repo hostinger/kubernetes-apps --pr-number 42 --pr-title "Bump nginx" \
      --base-ref master --head-ref feat/nginx --author alice

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  hydrate     Render ArgoCD applications into Kubernetes manifests

Flags:
      --author string              Pull request author
      --base string                Base manifest tree: a directory or a .zip (required)
      --base-applications string   ArgoCD Application resources of the base tree, for project and cluster labels (default: applications.yaml in the base tree, if present)
      --base-ref string            Base branch or ref
      --context int                Unchanged lines kept around each change (default 3)
      --full-text-limit int        Largest modified file, in lines, whose full text is embedded so context can be expanded in the viewer (-1: never) (default 2000)
      --head string                Head manifest tree: a directory or a .zip (required)
      --head-applications string   ArgoCD Application resources of the head tree (default: applications.yaml in the head tree, if present)
      --head-ref string            Head branch or ref
  -h, --help                       help for argocd-diff
      --output string              Output HTML file (default "diff.html")
      --pr-number int              Pull request number
      --pr-title string            Pull request title
      --pr-url string              Pull request URL (default: derived from --repo and --pr-number on github.com)
      --repo string                Repository as owner/name
      --summary string             Also write a Markdown summary, for a pull request comment, to this file
      --title string               Report title (default: the pull request, or the input paths)
  -v, --version                    version for argocd-diff

Use "argocd-diff [command] --help" for more information about a command.
```

The report groups files by application and offers a unified or split view,
context expansion, filters by status, application, project, cluster and kind,
intra-line change highlighting, dark mode and keyboard navigation. Only
`.yaml`/`.yml` files take part.

In GitHub Actions, `--repo`, `--base-ref` and `--head-ref` default to
`GITHUB_REPOSITORY`, `GITHUB_BASE_REF` and `GITHUB_HEAD_REF`, and `--pr-number`
to the number in `GITHUB_REF` on `pull_request` events (or to `GITHUB_PR_NUMBER`
if your workflow exports it), so only `--base` and `--head` are needed.
`--context` sets the unchanged lines kept around a change (default 3) and
`--full-text-limit` the largest modified file whose whole text is embedded so
context can be expanded in the viewer (default 2000 lines).

Each file is labelled with the ArgoCD Application it was rendered for (the
first path segment, which is the Application's `metadata.name`) and with that
Application's `spec.project` and `spec.destination.name`. Those come from the
Application resources themselves: `applications.yaml` at the root of each tree
when it exists (the hydrate step's input file, when it is written next to the
output), or the files given with `--base-applications` and
`--head-applications`. Without them, files still group by Application; only
the project and cluster labels stay empty.

The report embeds the diff as JSON and renders it in the browser, so a page
with hundreds of changed manifests stays small and only draws what is on
screen. Secrets are already redacted by the hydrate step.

`--summary <file>` also writes a Markdown summary for a pull request comment:
a totals line ("11 manifests changed: 4 added, 6 modified, 1 deleted") and a
table with one row per application, project and cluster, collapsed when
it has more than eight rows. A pull request that renders to no change says so
in one line.

## Hydrating

```console
$ argocd-diff hydrate --help
Render ArgoCD Application resources into the Kubernetes manifests they would
deploy. Helm chart sources are pulled and templated, directory sources are read
as they are, and Secret values are redacted. Each application is written to
<output>/<application>/<namespace>/<Kind>/<name>.yaml, or
<output>/<application>/<Kind>/<name>.yaml for a cluster-scoped object.

Usage:
  argocd-diff hydrate [flags]

Examples:
  argocd-diff hydrate
  argocd-diff hydrate --applications apps/applications.yaml --output rendered --kube-version 1.35.6

Flags:
      --applications string   Path to the file containing ArgoCD Application CRDs (default "manifests/applications.yaml")
      --charts-dir string     Directory for storing downloaded Helm charts (default "cache")
  -h, --help                  help for hydrate
      --kube-version string   Kubernetes version to use for rendering Helm charts (default "1.35.6")
      --output string         Output directory for the rendered manifests (default "manifests")
```

Charts are pulled once into `--charts-dir` and reused on later runs, so cache
that directory in CI. They are rendered for `--kube-version` the way
`helm template --kube-version` renders them, so a chart whose `kubeVersion`
constraint excludes it fails with a message that says so. Progress is logged
to stderr; an error is reported once and exits with status 1.

## GitHub Action

This repository is also a GitHub Action that installs the release it is used
at:

```yaml
- uses: kazysgurskas/argocd-diff@v1.0.0
- run: argocd-diff hydrate --kube-version 1.35.6
```

When the action is pinned by commit SHA, name the release to install:

```yaml
- uses: kazysgurskas/argocd-diff@<sha> # v1.0.0
  with:
    version: v1.0.0
```

A pull request workflow checks out the base and the head of the pull request,
renders each with `argocd-diff hydrate`, runs `argocd-diff --base --head
--summary`, publishes the HTML where reviewers can open it, and posts the
summary as a comment.

## Smoke test

`make smoke` hydrates the example under [`test/smoke/`](test/smoke/) for a base
and a head commit and diffs them into `test/smoke/out/example/diff.html`. See
[test/smoke/README.md](test/smoke/README.md) for using it with your own
scenarios.

## Docker

```bash
docker build -t argocd-diff .
docker run -v $(pwd):/workspace argocd-diff hydrate \
  --applications=/workspace/manifests/applications.yaml --output=/workspace/rendered
```

## License

This project is licensed under the [Apache License](LICENSE).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
