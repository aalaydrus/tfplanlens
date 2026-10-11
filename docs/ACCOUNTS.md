# Accounts: where it runs vs where resources land

Plans are often applied from a runner in an automation account that assumes roles into target accounts. The report shows both:

- **Apply account** ("executed from"): where `terraform apply` runs.
- **Target account**, per provider, with the source that decided it. Every resource whose target differs from the apply account is flagged.

You usually do not need to configure anything: both default to what the plan's own provider configuration says. The plan JSON does not include the backend, so anything it cannot tell you goes in `tfplanlens.config.yaml` (see [Config file](USAGE.md#config-file)).

## Apply account

Highest precedence first:

| # | Source | How |
|---|---|---|
| 1 | flag | `--tfapply-account-id 999999999999` |
| 2 | env var | `TFPLANLENS_TFAPPLY_ACCOUNT_ID=999999999999` |
| 3 | config file | `tfapply_account_id: "999999999999"` |
| 4 | inferred | the account of the **default (unaliased) provider**, if that provider has no `assume_role` (found as in the target-account table below) |
| 5 | unknown | the report says so in one notice; cross-account flags are off |

Nothing is asked and nothing is written to stderr. Use `--fail-on unresolved` in CI if an unknown account should fail the build.

## Target account

Resolved per provider, first match wins; the report shows which source was used:

| | Source | Shown as |
|---|---|---|
| a | `--account alias=id` (repeatable; overrides everything below) | `flag` |
| b | `accounts:` in the config file | `config` |
| c | `assume_role.role_arn`, literal or resolved through the plan's `variables` | `role_arn`, `role_arn_variable` |
| d | a single value in `allowed_account_ids` | `allowed_account_ids` |
| e | `account_id` of an `aws_caller_identity` data source in `prior_state` | `caller_identity` |
| f | providers with no `assume_role` run as the apply account | `apply_account` |
| g | otherwise | `unresolved` |

A resource whose own ARN-typed attributes (`arn`, `*_arn`, `*_arns`) name a different account than its provider's target gets a warning.
