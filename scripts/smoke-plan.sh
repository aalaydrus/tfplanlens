#!/usr/bin/env bash
# Generate a REAL Terraform plan for a smoke-test scenario.
#
#   scripts/smoke-plan.sh <scenario> [work-root]
#
# Applies testdata/real/<scenario>/v1, swaps in v2 (keeping state and
# .terraform), plans, and writes the `terraform show -json` output to
# testdata/plans/real/<scenario>-tf<terraform version>.json.
# Needs terraform >= 1.7 on PATH. Scenarios only use the built-in
# terraform_data resource: no providers, no credentials.
set -euo pipefail

scenario=${1:?usage: smoke-plan.sh <scenario> [work-root]}
root=$(cd "$(dirname "$0")/.." && pwd)
work="${2:-$root/work}/$scenario"

rm -rf "$work"
mkdir -p "$work"
cp -R "$root/testdata/real/$scenario/v1/." "$work/"
cd "$work"

terraform init -input=false -no-color
terraform apply -auto-approve -input=false -no-color

# Swap the configuration but KEEP terraform.tfstate and .terraform.
find . -maxdepth 1 -name '*.tf' -delete
rm -rf modules
cp -R "$root/testdata/real/$scenario/v2/." .
terraform init -input=false -no-color

terraform plan -input=false -no-color -out=plan.bin
terraform show -json plan.bin > plan.json

version=$(terraform version | head -1 | sed 's/^Terraform v//')
mkdir -p "$root/testdata/plans/real"
cp plan.json "$root/testdata/plans/real/$scenario-tf$version.json"
echo "wrote testdata/plans/real/$scenario-tf$version.json"
