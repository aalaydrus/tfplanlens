#!/usr/bin/env bash
# Seed floci (a local AWS emulator) with the IAM roles the aws-assume-role
# smoke scenario assumes. floci enforces IAM, so a role must exist, in its own
# account, with a trust policy that lets the runner's account assume it. floci
# does not check request signatures and takes the account id from a 12-digit
# access key id, so plain curl is enough: no AWS CLI, no real credentials.
#
#   scripts/floci-seed.sh [endpoint]      # default http://localhost:4566
set -euo pipefail

endpoint=${1:-${AWS_ENDPOINT_URL:-http://localhost:4566}}
runner_account=999999999999

create_role() { # <account> <role name>
  local account=$1 name=$2
  local trust
  trust=$(printf '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::%s:root"},"Action":"sts:AssumeRole"}]}' "$runner_account")
  local out
  out=$(curl -sS -m 20 -X POST "$endpoint/" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "Authorization: AWS4-HMAC-SHA256 Credential=$account/20260101/us-east-1/iam/aws4_request, SignedHeaders=host, Signature=unused" \
    --data-urlencode "Action=CreateRole" --data-urlencode "Version=2010-05-08" \
    --data-urlencode "RoleName=$name" --data-urlencode "AssumeRolePolicyDocument=$trust")
  case "$out" in
    *"<RoleName>$name</RoleName>"*|*EntityAlreadyExists*) echo "role $name in account $account: ok" ;;
    *) echo "could not create role $name in $account: $out" >&2; exit 1 ;;
  esac
}

create_role 111111111111 prod-deployer
create_role 222222222222 staging-deployer
