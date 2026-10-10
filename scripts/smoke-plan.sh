#!/usr/bin/env bash
# Generate a REAL Terraform plan for a smoke-test scenario.
#
#   scripts/smoke-plan.sh <scenario> [work-root]
#
# Applies testdata/real/<scenario>/v1, swaps in v2 (keeping state and
# .terraform), plans, and writes the `terraform show -json` output to
# testdata/plans/real/<scenario>-tf<terraform version>.json.
#
# A scenario directory may also contain:
#   between.sh  run in the work directory after v1 is applied and before v2
#               is swapped in: out-of-band changes, taints, state surgery.
#               $SCENARIO_DIR points at the scenario directory.
#   plan-args   extra arguments for `terraform plan`, e.g. -destroy,
#               -refresh-only or -replace=terraform_data.x
#
# Needs terraform >= 1.7. Most scenarios use only the built-in terraform_data
# resource (no providers, no credentials); `drift` and `refresh-only` also use
# the credential-free hashicorp/local provider. Set SMOKE_PLUGIN_DIR to a
# provider mirror directory to install providers offline.
set -euo pipefail

scenario=${1:?usage: smoke-plan.sh <scenario> [work-root]}
root=$(cd "$(dirname "$0")/.." && pwd)
dir="$root/testdata/real/$scenario"
work="${2:-$root/work}/$scenario"

init_args=()
if [ -n "${SMOKE_PLUGIN_DIR:-}" ]; then
  init_args=("-plugin-dir=$SMOKE_PLUGIN_DIR")
fi

rm -rf "$work"
mkdir -p "$work"
cp -R "$dir/v1/." "$work/"
cd "$work"

terraform init -input=false -no-color ${init_args[@]+"${init_args[@]}"}
terraform apply -auto-approve -input=false -no-color

if [ -f "$dir/between.sh" ]; then
  SCENARIO_DIR="$dir" bash "$dir/between.sh"
fi

# Swap the configuration but KEEP terraform.tfstate, .terraform and any
# terraform.tfvars written by between.sh.
find . -maxdepth 1 -name '*.tf' -delete
rm -rf modules
cp -R "$dir/v2/." .
terraform init -input=false -no-color ${init_args[@]+"${init_args[@]}"}

plan_args=()
if [ -f "$dir/plan-args" ]; then
  read -r -a plan_args < "$dir/plan-args"
fi
terraform plan -input=false -no-color ${plan_args[@]+"${plan_args[@]}"} -out=plan.bin
terraform show -json plan.bin > plan.json

# sed (not head) reads all of the output: with pipefail, head closing the pipe early kills terraform with SIGPIPE.
version=$(terraform version | sed -n '1s/^Terraform v//p')
mkdir -p "$root/testdata/plans/real"
cp plan.json "$root/testdata/plans/real/$scenario-tf$version.json"
echo "wrote testdata/plans/real/$scenario-tf$version.json"
