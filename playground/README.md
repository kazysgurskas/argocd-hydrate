# Playground

A place to run `argocd-hydrate` and `argocd-hydrate diff` end to end without a
cluster or a real GitOps repository.

```console
$ make playground          # or ./playground/run.sh
open .../playground/out/example/diff.html
```

`example/` is committed: two applications with directory sources, in a `base`
and a `head` version. The scenario is an Ingress to Gateway API migration in
`myapp` (an `Ingress` deleted, an `HTTPRoute`, `ServiceAccount`,
`PodDisruptionBudget` and `NetworkPolicy` added, a `Deployment`, `Service`,
`ConfigMap`, `Secret` and `HorizontalPodAutoscaler` modified) plus an image
bump in `otherapp`.

Everything the run produces goes under `out/` and `cache/`, which are
gitignored, as is `local/`: put your own scenarios there, for example a copy of
a real repository's `applications.yaml` for two commits, and run
`./playground/run.sh local/<name>`.
