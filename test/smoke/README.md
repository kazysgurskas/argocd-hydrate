# Smoke test

A place to run `argocd-diff hydrate` and `argocd-diff` end to end without a
cluster or a real GitOps repository.

```console
$ make smoke          # or ./test/smoke/run.sh
open .../test/smoke/out/example/diff.html
```

`example/` is committed: `base/` and `head/` are two checkouts of a GitOps
repository, each with an `applications.yaml` of two directory-source
applications, and the script runs the one-command form that hydrates both and
diffs them. The scenario is an Ingress to Gateway API migration in
`myapp` (an `Ingress` deleted, an `HTTPRoute`, `ServiceAccount`,
`PodDisruptionBudget` and `NetworkPolicy` added, a `Deployment`, `Service`,
`ConfigMap`, `Secret` and `HorizontalPodAutoscaler` modified) plus an image
bump in `otherapp`.

Everything the run produces goes under `out/` (the rendered trees as `base/`
and `head/`, `diff.html` and `diff.md`) and `cache/`, which are gitignored, as
is `local/`: put your own scenarios there, for example two checkouts of a real
repository with their `applications.yaml`, and run `./test/smoke/run.sh local/<name>`.
