#!/usr/bin/env bash
# Run tfplanlens on one real plan in every output format.
#
#   scripts/smoke-report.sh <scenario> <plan.json> [out-root]
#
# Writes docs/smoke/<scenario>/report.{txt,html,md,json} with an apply account
# given, plus report-no-apply-account.txt from a run WITHOUT one: stdin is not
# a terminal there, so tfplanlens must warn and continue, never prompt or hang.
set -euo pipefail

scenario=${1:?usage: smoke-report.sh <scenario> <plan.json> [out-root]}
plan=${2:?plan file}
root=$(cd "$(dirname "$0")/.." && pwd)
out="${3:-$root/docs/smoke}/$scenario"
bin=${TFPLANLENS:-$root/tfplanlens}

mkdir -p "$out"
for spec in text:txt html:html markdown:md json:json; do
  format=${spec%%:*}
  ext=${spec##*:}
  timeout 60 "$bin" report "$plan" --format "$format" --no-color \
    --tfapply-account-id=111111111111 --out "$out/report.$ext" </dev/null
done

# No apply account: must finish (exit 0) with a warning on stderr.
timeout 60 "$bin" report "$plan" --format text --no-color \
  --out "$out/report-no-apply-account.txt" </dev/null 2> "$out/no-apply-account.stderr"
grep -q "apply account is unresolved" "$out/no-apply-account.stderr"
echo "wrote $out"
