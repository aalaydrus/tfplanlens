#!/usr/bin/env bash
# Regenerate the committed real plans and smoke reports locally, the same way
# the smoke workflow does, then commit the result yourself:
#
#   scripts/smoke-refresh.sh                 # plans, reports, rereview diff
#   SCREENSHOTS=1 scripts/smoke-refresh.sh   # also docs/screenshots (needs node)
#
# Writes testdata/plans/real/*.json, docs/smoke/** and, with SCREENSHOTS=1,
# docs/screenshots/**. Needs terraform >= 1.7. The aws-assume-role scenario also
# needs floci on :4566 (docker run -d -p 4566:4566 floci/floci:2.2.0; use a fresh
# container each run) and is skipped with a message otherwise. Set
# SMOKE_PLUGIN_DIR to a provider mirror to install providers offline.
#
# Why this is a script and not a workflow step: pushes made with a workflow's
# built-in token do not start other workflows, so a bot commit on a pull request
# has no CI results and would block required status checks.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

go build -o tfplanlens ./cmd/tfplanlens

for dir in testdata/real/*/; do
  scripts/smoke-plan.sh "$(basename "$dir")" "$work"
done

for plan in testdata/plans/real/*.json; do
  scenario=$(basename "$plan" | sed 's/-tf.*//')
  scripts/smoke-report.sh "$scenario" "$plan"
done

# Two real plans of the same state (see scripts/smoke-plan.sh, v3).
before=$(ls testdata/plans/real/rereview-tf*.json)
after=$(ls testdata/plans/real/rereview-after-tf*.json)
mkdir -p docs/smoke/rereview-diff
for spec in text:txt markdown:md json:json; do
  ./tfplanlens diff "$before" "$after" --format "${spec%%:*}" \
    --tfapply-account-id=111111111111 --out "docs/smoke/rereview-diff/diff.${spec##*:}" </dev/null
done

if [ "${SCREENSHOTS:-}" = 1 ]; then
  npm install --no-save --no-package-lock playwright
  npx playwright install --with-deps chromium
  SCREENSHOT_SCENARIOS=simple,complex node scripts/screenshot.mjs docs/smoke docs/screenshots
fi

echo "done. Review with: git status; git diff --stat"
echo "Then run the tests on the fresh plans: go test ./... -count=1"
