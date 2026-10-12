#!/usr/bin/env bash
# Build the GitHub Pages demo site: the committed example reports plus an index.
#
#   scripts/build-demo-site.sh <out-dir>
#
# Only copies files that are already committed. Nothing is generated from a
# live plan, so the demo never contains anything that is not in the repository.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
out="${1:?usage: build-demo-site.sh <out-dir>}"

# site path -> committed report
reports=(
  "orders-platform.html:simulation/output/report.html"
  "assume-role.html:docs/smoke/aws-assume-role/report.html"
  "drift.html:docs/smoke/drift/report.html"
  "destroy.html:docs/smoke/destroy/report.html"
  "complex.html:docs/smoke/complex/report.html"
  "import.html:docs/smoke/import/report.html"
  "refresh-only.html:docs/smoke/refresh-only/report.html"
  "module-provider.html:docs/smoke/module-provider/report.html"
  "reasons.html:docs/smoke/reasons/report.html"
  "simple.html:docs/smoke/simple/report.html"
)

rm -rf "$out"
mkdir -p "$out/reports"
cp "$root/docs/demo/index.html" "$out/index.html"
for r in "${reports[@]}"; do
  dest="${r%%:*}"
  src="$root/${r#*:}"
  [ -f "$src" ] || { echo "missing $src" >&2; exit 1; }
  cp "$src" "$out/reports/$dest"
done
touch "$out/.nojekyll"
echo "demo site written to $out ($(find "$out/reports" -name '*.html' | wc -l) reports)"
