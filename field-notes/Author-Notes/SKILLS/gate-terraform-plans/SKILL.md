---
name: gate-terraform-plans
description: Set up or explain a CI gate that blocks a Terraform plan with risky changes. Use when the user wants a pipeline step to fail on deletes, replaces, drift, unresolved accounts, or on specific resource types (for example a database), or asks what the tfplanlens exit codes mean. Uses `tfplanlens report --fail-on` and the config file.
---

# Gate Terraform plans in CI with tfplanlens

`tfplanlens report` prints the report **first**, then exits non-zero when a rule
matches, so a pipeline can show the report and still block.

## Exit codes

`0` no rule matched; `1` a rule matched (the matches are on stderr);
`2` an error (bad file, flag or config). Treat 2 as a broken step, not a block.

## Rules (`--fail-on`, comma-separated)

| Rule | Matches |
|---|---|
| `delete` | resources the plan deletes (not replaces) |
| `replace` | resources the plan replaces |
| `replace:aws_db_instance` | replaces of one type (`delete:<type>` likewise) |
| `drift` | changes made outside Terraform |
| `unresolved` | an account that could not be determined |

Rules are checked on the whole plan even if the report hides some types
(`ignore_types`).

## Put the rules in the config file

`tfplanlens.config.yaml` next to where the step runs (the flag overrides it):

```yaml
tfapply_account_id: "999999999999"   # account terraform apply runs from
fail_on: [delete, "replace:aws_dynamodb_table"]
format: [markdown]
```

## A pipeline step (shell)

```
terraform plan -out=plan.bin
terraform show -json plan.bin > plan.json     # contains secrets: never upload as an artifact
tfplanlens report plan.json -f markdown -o report.md || status=$?
# post report.md as the PR comment here
exit "${status:-0}"
```

To block only on what a push *added*, compare with the last reviewed plan:
`tfplanlens diff old.json new.json --fail-on-new-danger`.

## Choosing rules

Start with what is irreversible: `delete`, and `replace:<type>` for each
stateful type the team cares about. Tell the user a replace of a secret
version (a password rotation) is routine and would trip a bare `replace` rule:
scope replace rules by type. Do not add rules the user did not ask for.
The tool has no published releases yet: the pipeline must build or provide the
binary itself.
