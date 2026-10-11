#!/usr/bin/env bash
# Regenerate simulation/assets/input from the Terraform in simulation/assets/terraform.
#
#   simulation/regenerate.sh [work-dir]
#
# Needs terraform >= 1.7 and floci, a local AWS emulator (no real AWS access):
#   docker run -d --name floci -p 4566:4566 floci/floci
# The AWS provider addresses S3 as <bucket>.localhost:4566 (and <account>.localhost for tags), which most systems
# resolve to 127.0.0.1 by themselves; if yours does not, add
#   127.0.0.1 orders-receipts-prod.localhost 123456789012.localhost
# to /etc/hosts.
# Set SIMULATION_PLUGIN_DIR to a provider mirror to install the AWS provider
# offline. The output is the same up to resource ids and timestamps.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
tf="$here/assets/terraform"
out="$here/assets/input"
work="${1:-$(mktemp -d)}"

# A 12-digit access key id is the account id for floci: this is the workload account.
export AWS_ENDPOINT_URL=${AWS_ENDPOINT_URL:-http://localhost:4566}
export AWS_ACCESS_KEY_ID=123456789012 AWS_SECRET_ACCESS_KEY=test AWS_REGION=us-east-1
export AWS_EC2_METADATA_DISABLED=true TF_IN_AUTOMATION=1

init_args=()
if [ -n "${SIMULATION_PLUGIN_DIR:-}" ]; then init_args=("-plugin-dir=$SIMULATION_PLUGIN_DIR"); fi

mkdir -p "$work" "$out"
cd "$work"

# 1. Deploy v1: this is the production state the plans are made against.
# The database password is a fake value; the proposed change rotates it.
export TF_VAR_db_password=initial-password-v1
cp "$tf/v1-deployed/main.tf" .
terraform init -input=false -no-color ${init_args[@]+"${init_args[@]}"} >/dev/null
terraform apply -auto-approve -input=false -no-color >/dev/null
cp terraform.tfstate "$out/terraform.tfstate"

# 2. Plan the proposed change (v2), then the revised one (v3), against the same state.
export TF_VAR_db_password=rotated-password-v2
for spec in v2-proposed:plan.json v3-revised:plan-revised.json; do
  cp "$tf/${spec%%:*}/main.tf" main.tf
  terraform plan -input=false -no-color -out=plan.bin >/dev/null
  terraform show -json plan.bin > "$out/${spec##*:}"
done
echo "wrote $out/{terraform.tfstate,plan.json,plan-revised.json}"
