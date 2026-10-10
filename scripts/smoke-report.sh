#!/usr/bin/env bash
# Run tfplanlens on one real plan in every output format.
#
#   scripts/smoke-report.sh <scenario> <plan.json> [out-root]
#
# Writes docs/smoke/<scenario>/report.{txt,html,md,json} with an apply account
# given, plus report-no-apply-account.txt from a run WITHOUT one.
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

# No apply account given: nothing is asked and nothing is written to stderr.
# The report must finish (exit 0) and say where the apply account came from:
# inferred from the default provider when the plan allows it (aws-assume-role),
# otherwise a notice that it is unknown.
timeout 60 "$bin" report "$plan" --format text --no-color \
  --out "$out/report-no-apply-account.txt" </dev/null 2> "$out/stderr.tmp"
test ! -s "$out/stderr.tmp"
rm -f "$out/stderr.tmp"
grep -qE "Apply account unknown|source: inferred from the default provider" "$out/report-no-apply-account.txt"
echo "wrote $out"
