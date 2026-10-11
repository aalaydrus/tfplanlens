# Terraform and provider gotchas

Everything here was seen in real `terraform show -json` output unless marked
`hypothesis`. Versions: Terraform 1.16.5, `hashicorp/aws` 6.68.0 (the
simulation) and `hashicorp/local` 2.9.1, `terraform_data` (the smoke scenarios).

## Reading plan JSON

- **Secrets are in plain text.** `variables.<name>.value`, the before/after
  values of resources, and prior state hold sensitive values; the
  `*_sensitive` maps only say which keys are sensitive. A plan file is as
  secret as the state. tfplanlens never prints them (`TestSimulationNeverPrintsSecretValues`),
  but never commit a real plan. *verified: `simulation/assets/input/plan.json`*
- **The backend is not in the plan.** So the account `terraform apply` runs
  from cannot be read from the plan; it comes from the config file, a flag or
  an inference. *verified*
- **`replace_triggered_by` is not in the configuration JSON**, and `local.*`
  references cannot be resolved, so dependency waves are a lower bound. A
  `replace_triggered_by` replace shows reason `replace_by_triggers`. *verified:
  `testdata/real/reasons`*
- **No top-level plan mode.** Refresh-only and destroy plans are recognized by
  what is in them (refresh-only has no `resource_changes`). *verified*
- **`forget` is an action** (a `removed` block); moved resources carry
  `previous_address`. *verified*
- **Module resources' `provider_config_key`** is the resolved root key (for
  example `aws.prod`), not the module's local name. *verified*
- **`assume_role` is a list** (`[{"role_arn": {...}}]`) whose value is either
  `constant_value` or `references`. In `terraform-json`, `ConstantValue` is a
  sentinel when references exist, so check references first. *verified*
- **`configuration` exposes dependency edges** as `references` lists on
  expressions (and `depends_on`, `count_expression`, `for_each_expression`,
  module call `expressions`). That is where waves come from. *verified*

- **Unmarked secrets are printed by Terraform itself, and tfplanlens used to
  repeat them.** A secret in an attribute Terraform does not mark sensitive, or
  inside `user_data`, is in the plan JSON and in `terraform plan` output;
  tfplanlens additionally decodes base64/gzip `user_data`, which turned an
  obscured secret into readable lines. Now redacted by name and by shape
  (`docs/SECRETS.md`). *verified: a crafted plan, `testdata/plans/secret-lookalikes.json`*

## Noise from the AWS provider

- **Drift that nobody made.** On refresh the provider fills legacy attributes:
  `aws_iam_role.inline_policy` and `aws_s3_bucket.versioning` show up as
  drift, and nearly every resource shows `tags: null -> {}`. We hide the tags
  case; the others are listed as drift. *verified: simulation, Drift section*
- **A replace sets computed-optional attributes to `null`**
  (`stream_enabled`, `table_class`, `tags` on `aws_dynamodb_table`), so a
  replace lists `x: false -> null` lines that are not real changes. *verified*

## Edge cases where the plan is technically right but misleading

- **Rotating a secret is a replace, not an update.** Changing `secret_string`
  on `aws_secretsmanager_secret_version` forces replacement in provider 6.68.0
  (`delete, create`, reason `replace_because_cannot_update`), and the new
  version gets a new `version_id`. We list it under Danger like any replace,
  though in practice it is a routine rotation. Also `secret_binary` shows as
  "(sensitive value changed)" although it is empty before and null after.
  *verified: simulation v2*. See the wishlist item on benign replaces.
- **Removing one for_each key and adding another looks like a rename**, and
  the delete is real: a queue `bulk` removed while `audit` is added gives the
  "possible key rename" warning. *verified: simulation v2*
- **Changing a DynamoDB primary key** (`hash_key`/`range_key`) forces
  replacement of the table. *verified: simulation v2*
- **Mixed instances of one resource** (create one key, delete another) are
  one resource in the waves; we show it as apply work. *verified*

## floci (the local AWS emulator we test against)

- The account id is taken from a 12-digit access key id.
- `AssumeRole` is denied until the roles exist (`scripts/floci-seed.sh`).
- State from a previous run breaks the next one: use a fresh container.
- S3 is addressed as `<bucket>.localhost:4566` and `<account>.localhost:4566`;
  where `*.localhost` does not resolve, add `/etc/hosts` entries.
- It supports the resource types in the simulation (S3, DynamoDB, SQS, SNS,
  SSM, Secrets Manager, IAM); anything else is untested.

## Tooling

- **Bubble Tea queries the terminal at import time** (`init()` asks for the
  background color), which stalled `tfplanlens report` for 5 seconds on
  terminals that do not answer. We use `golang.org/x/term` instead.
  *verified in a pty*
- `/dev/null` is not a terminal: use `x/term`'s `IsTerminal`, not a character
  device check.
- `go get` of a newer `x/net` bumps `go.mod`'s `go` line; pin `x/net`.
