#!/usr/bin/env bash
# Hydrates the example's base and head trees and diffs them into out/diff.html.
#
#   ./test/smoke/run.sh              # the committed example
#   ./test/smoke/run.sh local/mine   # your own scenario under test/smoke/local/ (gitignored)
#
# A scenario directory holds base/applications.yaml and head/applications.yaml
# whose source paths are relative to test/smoke/.
set -euo pipefail
cd "$(dirname "$0")"

example=${1:-example}
bin=${ARGOCD_DIFF_BIN:-../../bin/argocd-diff}
if [ ! -x "$bin" ]; then
  echo "building bin/argocd-diff" >&2
  (cd ../.. && make build >/dev/null)
fi

out=out/$(basename "$example")
rm -rf "$out" && mkdir -p "$out"
for side in base head; do
  "$bin" hydrate --applications "$example/$side/applications.yaml" --output "$out/$side" --charts-dir cache
done
"$bin" --base "$out/base" --head "$out/head" --output "$out/diff.html" \
  --base-applications "$example/base/applications.yaml" --head-applications "$example/head/applications.yaml" \
  --summary "$out/diff.md" \
  --repo example/gitops --pr-number 1 --pr-title "Ingress to Gateway API migration" \
  --base-ref main --head-ref feature/gateway-api --author smoke
echo "open $(pwd)/$out/diff.html"
