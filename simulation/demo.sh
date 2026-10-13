#!/usr/bin/env bash
# Run tfplanlens on the simulation plans and write every output to
# simulation/output. No Terraform, AWS or network needed: the plans are
# committed in assets/input.
#
#   simulation/demo.sh            # builds ../tfplanlens first if tfplanlens is not on PATH
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
cd "$here"
if command -v tfplanlens >/dev/null 2>&1; then
  bin=tfplanlens
else
  (cd .. && go build -o tfplanlens ./cmd/tfplanlens)
  bin=../tfplanlens
fi

plan=assets/input/plan.json
revised=assets/input/plan-revised.json
mkdir -p output

# The proposed change, in every format. tfplanlens.config.yaml (next to this
# script) supplies the apply account.
for spec in text:txt markdown:md html:html json:json; do
  "$bin" report "$plan" --format "${spec%%:*}" --no-color --out "output/report.${spec##*:}"
done

# The short, ranked list for a reviewer or a code review agent.
"$bin" report "$plan" --format brief --out output/brief.txt

# The revised change, and what changed between the two.
"$bin" report "$revised" --no-color --out output/report-revised.txt
"$bin" diff "$plan" "$revised" --out output/diff.txt
"$bin" diff "$plan" "$revised" --format markdown --out output/diff.md

# A CI gate: exit code 1 when the plan deletes or replaces a DynamoDB table or
# deletes anything. The report is still written first.
set +e
"$bin" report "$plan" --no-color --fail-on delete,replace:aws_dynamodb_table > /dev/null 2> output/gate-stderr.txt
echo "proposed plan, gate exit code: $?" | tee output/gate-exit.txt
"$bin" report "$revised" --no-color --fail-on delete,replace:aws_dynamodb_table > /dev/null 2> /dev/null
echo "revised plan,  gate exit code: $?" | tee -a output/gate-exit.txt
