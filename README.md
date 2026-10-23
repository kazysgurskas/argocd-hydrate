# argocd-diff

argocd-diff renders ArgoCD Applications and shows
what a pull request changes in the resulting Kubernetes manifests. It needs no
ArgoCD server, cluster or kubeconfig. It reads Application resources from a
checkout of the GitOps repository, pulls and templates Helm charts with the
Helm SDK, copies directory sources as they are, and writes the manifests the
way ArgoCD's repo server would, with Secret values redacted. The only network
access is to the chart repositories the Applications reference, and pulled
charts are cached between runs.

Given the base and head checkouts of a pull request, it renders both, diffs
them, and writes a self-contained HTML report plus a Markdown summary for a
pull request comment. Unlike `argocd app diff`, which compares the desired
state against a live cluster, argocd-diff compares two versions of the desired
state. That makes it usable in CI on every pull request, in repositories whose
ArgoCD isn't reachable from CI, and with the rendered-manifests pattern, where
the rendered tree is what gets reviewed and committed.

Supported sources are Helm charts from HTTP and OCI repositories and
directories of plain manifests, in both single-source and multi-source
Applications. Private chart registries work the same way as with `helm pull`:
log in with `helm registry login` or a Docker credential helper first.

## Usage

```bash
argocd-diff --base base-checkout --head head-checkout \
  --applications manifests/applications.yaml \
  --kube-version 1.35.6 --charts-dir charts --rendered rendered \
  --output diff.html --summary diff.md
```

`--applications` is the path, inside each
checkout, of the file with the ArgoCD Application resources — for example the
output of `helm template` on an app-of-apps chart, or a plain file of
Application manifests. Source paths in those Applications (`path:`,
`$values/…`) are resolved against their own checkout. Both sides are rendered
and then diffed. The rendered trees are written to `--rendered` (replacing its
`base/` and `head/`), where other tools such as kubeconform can read them, or
to a temporary directory if the flag is omitted. Since there is no cluster to
ask, charts are rendered for `--kube-version`; a chart whose `kubeVersion`
constraint doesn't match fails with an error saying so. Charts are downloaded
once into `--charts-dir` and reused on later runs, so cache that directory in
CI.

```console
$ argocd-diff --help
Compare two trees of rendered manifests, as "argocd-diff hydrate" writes
them, and write a self-contained HTML report of what changed. Each input is a
directory or a zip archive of one. With --applications, each input is a
checkout of the GitOps repository (a directory or zip of one) holding that
Application file, and both are hydrated first, so one command does the whole
job.

Pull request details are optional. Inside GitHub Actions the repository, base
and head refs come from GITHUB_REPOSITORY, GITHUB_BASE_REF and GITHUB_HEAD_REF,
and the number from GITHUB_REF on pull_request events (refs/pull/<n>/merge) or
from GITHUB_PR_NUMBER if the workflow exports it, so only --base and --head are
needed there.

Usage:
  argocd-diff [flags]
  argocd-diff [command]

Examples:
  # Hydrate the base and head checkouts of a pull request and diff them
  argocd-diff --base base-checkout --head head-checkout --applications manifests/applications.yaml \
      --kube-version 1.35.6 --charts-dir charts --rendered rendered --summary diff.md

  # Or diff two trees rendered earlier with "argocd-diff hydrate"
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
      --applications string        Path, inside each tree, of the ArgoCD Application file; each tree is then a checkout of the repository, hydrated before diffing, rather than rendered manifests
      --author string              Pull request author
      --base string                Base manifest tree: a directory or a .zip (required)
      --base-applications string   ArgoCD Application resources of the base tree, for project and cluster labels (default: applications.yaml in the base tree, if present)
      --base-ref string            Base branch or ref
      --charts-dir string          With --applications, directory for storing downloaded Helm charts (default "cache")
      --context int                Unchanged lines kept around each change (default 3)
      --full-text-limit int        Largest modified file, in lines, whose full text is embedded so context can be expanded in the viewer (-1: never) (default 2000)
      --head string                Head manifest tree: a directory or a .zip (required)
      --head-applications string   ArgoCD Application resources of the head tree (default: applications.yaml in the head tree, if present)
      --head-ref string            Head branch or ref
  -h, --help                       help for argocd-diff
      --kube-version string        With --applications, Kubernetes version to use for rendering Helm charts (default "1.35.6")
      --output string              Output HTML file (default "diff.html")
      --pr-number int              Pull request number
      --pr-title string            Pull request title
      --pr-url string              Pull request URL (default: derived from --repo and --pr-number on github.com)
      --rendered string            With --applications, keep the rendered trees in this directory, replacing its base/ and head/ (default: a temporary directory)
      --repo string                Repository as owner/name
      --summary string             Also write a Markdown summary, for a pull request comment, to this file
      --title string               Report title (default: the pull request, or the input paths)
  -v, --version                    version for argocd-diff

Use "argocd-diff [command] --help" for more information about a command.
```

## The report

The report groups files by application and offers unified and split views,
context expansion, filters by status, application, project, cluster and kind,
intra-line change highlighting, dark mode and keyboard navigation. Only
`.yaml`/`.yml` files are included.

In GitHub Actions, `--repo`, `--base-ref` and `--head-ref` default to
`GITHUB_REPOSITORY`, `GITHUB_BASE_REF` and `GITHUB_HEAD_REF`, and `--pr-number`
is taken from `GITHUB_REF` on `pull_request` events (or from `GITHUB_PR_NUMBER`
if your workflow exports it), so only `--base` and `--head` are required.
`--context` sets how many unchanged lines are kept around each change
(default 3) and `--full-text-limit` the largest modified file whose full text
is embedded so context can be expanded in the viewer (default 2000 lines).

Each file is labelled with the Application it was rendered for (the first
path segment, i.e. the Application's `metadata.name`), plus that Application's
`spec.project` and `spec.destination.name`. With `--applications` these come
from the Applications just rendered, and `--base-applications`/
`--head-applications` are not allowed. When diffing previously rendered trees,
they come from `applications.yaml` at the root of each tree if present, or
from `--base-applications` and `--head-applications`. Without either, files
are still grouped by Application; only the project and cluster labels stay
empty.

The diff is embedded as JSON and rendered in the browser, so a report with
hundreds of changed manifests stays small and only draws what is on screen.
Secrets are redacted during hydration.

`--summary <file>` writes a Markdown summary for a pull request comment: a
totals line ("11 manifests changed: 4 added, 6 modified, 1 deleted") and a
table with one row per application, project and cluster, collapsed when it
has more than eight rows. A pull request with no rendered changes gets a
one-line note saying so.

## Rendering on its own

`argocd-diff hydrate` renders a single checkout into a tree of manifests. Use
it for the rendered-manifests pattern, for review tools that read plain
manifests, or to diff trees rendered at different times with
`argocd-diff --base <tree> --head <tree>`.

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

Progress is logged to stderr. On error, the message is printed once and the
exit status is 1.

## GitHub Action

This repository contains a GitHub Action. It installs the release matching the
tag it is used with:

```yaml
- uses: kazysgurskas/argocd-diff@v1.0.0
- run: |
    argocd-diff --base base-checkout --head head-checkout \
      --applications manifests/applications.yaml --rendered rendered \
      --output diff.html --summary diff.md
```

When pinning the action by commit SHA, specify the release to install:

```yaml
- uses: kazysgurskas/argocd-diff@<sha> # v1.0.0
  with:
    version: v1.0.0
```

A typical pull request workflow checks out the base and head of the pull
request into two directories, produces the Application file in each (for
example with `helm template` on an app-of-apps chart), runs the command above,
publishes `diff.html` somewhere reviewers can open it, and posts `diff.md` as
a comment. None of this needs credentials for ArgoCD or the clusters, only for
private chart registries.

## Smoke test

`make smoke` runs the one-command form on the example under
[`test/smoke/`](test/smoke/), two checkouts of a small GitOps repository, and
writes `test/smoke/out/example/diff.html`. See
[test/smoke/README.md](test/smoke/README.md) for using it with your own
scenarios.

## Docker

```bash
docker build -t argocd-diff .
docker run -v $(pwd):/workspace -w /workspace argocd-diff \
  --base base-checkout --head head-checkout --applications manifests/applications.yaml
```

## License

This project is licensed under the [Apache License](LICENSE).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
