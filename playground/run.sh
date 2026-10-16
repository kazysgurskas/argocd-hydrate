#!/usr/bin/env bash
# Hydrates the example's base and head trees and diffs them into out/diff.html.
#
#   ./playground/run.sh              # the committed example
#   ./playground/run.sh local/mine   # your own playground under playground/local/ (gitignored)
#
# A playground directory holds base/applications.yaml and head/applications.yaml
# whose source paths are relative to playground/.
set -euo pipefail
cd "$(dirname "$0")"

example=${1:-example}
bin=${ARGOCD_HYDRATE_BIN:-../bin/argocd-hydrate}
if [ ! -x "$bin" ]; then
  echo "building ../bin/argocd-hydrate" >&2
  (cd .. && make build >/dev/null)
fi

out=out/$(basename "$example")
rm -rf "$out" && mkdir -p "$out"
for side in base head; do
  "$bin" hydrate --applications "$example/$side/applications.yaml" --output "$out/$side" --charts-dir cache
done
"$bin" diff --base "$out/base" --head "$out/head" --output "$out/diff.html" \
  --repo example/gitops --pr-number 1 --pr-title "Ingress to Gateway API migration" \
  --base-ref main --head-ref feature/gateway-api --author playground
echo "open $(pwd)/$out/diff.html"
