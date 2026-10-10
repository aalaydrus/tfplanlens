# tfplanlens

Read a Terraform plan the way a reviewer needs to, so a clean-looking plan stops failing at apply.

`tfplanlens` analyzes `terraform show -json` output offline: no AWS credentials, no Terraform binary, no network. See [SPEC.md](SPEC.md) for the full design.

```sh
terraform plan -out plan.out
terraform show -json plan.out > plan.json
tfplanlens report plan.json --tfapply-account-id 999999999999
```

## Accounts: where it runs vs where resources land

Plans are often applied from a runner in an automation account that assumes roles into target accounts. The report shows both:

- **Apply account** ("executed from"): where `terraform apply` runs.
- **Target account**, per provider, with the source that decided it. Every resource whose target differs from the apply account is flagged.

### Apply account

Highest precedence first:

| # | Source | How |
|---|---|---|
| 1 | flag | `--tfapply-account-id 999999999999` |
| 2 | env var | `TFPLANLENS_TFAPPLY_ACCOUNT_ID=999999999999` |
| 3 | config file | `tfapply_account_id: "999999999999"` in `.tfplanlens.yaml` |
| 4 | prompt | asks, and offers to save to `.tfplanlens.yaml`; only when stdin and stdout are terminals and `--no-prompt` is not set |
| 5 | unresolved | warning on stderr, report continues |

`--require-apply-account` exits `2` instead of continuing when it is unresolved (use it in CI). `--no-prompt` disables the prompt.

### Target account

Resolved per provider, first match wins; the report shows which source was used:

| | Source | Shown as |
|---|---|---|
| a | `--account alias=id` (repeatable; overrides everything below) | `flag` |
| b | `accounts:` in `.tfplanlens.yaml` | `config` |
| c | `assume_role.role_arn`, literal or resolved through the plan's `variables` | `role_arn`, `role_arn_variable` |
| d | a single value in `allowed_account_ids` | `allowed_account_ids` |
| e | `account_id` of an `aws_caller_identity` data source in `prior_state` | `caller_identity` |
| f | providers with no `assume_role` run as the apply account | `apply_account` |
| g | otherwise | `unresolved` |

A resource whose own ARN-typed attributes (`arn`, `*_arn`, `*_arns`) name a different account than its provider's target gets a warning.

## Flags

```
tfplanlens report <plan.json|-> [flags]

--format text|html|markdown|json   output format (repeatable)
--out <path>                write to a file instead of stdout
--config <path>             config file (default .tfplanlens.yaml if present)
--fail-on <rules>           delete, replace, drift, or kind:type, e.g. replace:aws_db_instance
--max-line <n>              terminal truncation width (default 120)
--no-color
--tfapply-account-id <id>   account terraform apply runs from
--account <alias>=<id>      target account for a provider alias (repeatable)
--no-prompt                 never ask for the apply account
--require-apply-account     exit 2 if the apply account is unresolved
```

Exit codes: `0` ok, `1` a `--fail-on` rule matched, `2` error.

## Output formats

| Format | What you get |
|---|---|
| `text` | colored terminal report: Summary, Danger, Drift, Changes by account, Changes by module |
| `markdown` | for PR comments: summary table, Danger expanded, the rest in `<details>` per account then module; capped at 60,000 characters |
| `html` | one self-contained file (no CDN, works offline, dark mode): Danger panel, tiles, chart, searchable and groupable table, per-resource detail drawers |
| `json` | the analysis as stable JSON, see [docs/json-schema.md](docs/json-schema.md) |

Every format shows the apply account and each resource's target account with the source that decided it.

## Releases

Tagging `v*` runs GoReleaser (`.goreleaser.yaml`): binaries for linux, darwin and windows (amd64, arm64), checksums, and a Homebrew formula pushed to `aalaydrus/homebrew-tap` when the `HOMEBREW_TAP_GITHUB_TOKEN` secret is set.

## Smoke tests on real plans

`.github/workflows/smoke.yml` runs tfplanlens against real Terraform plans (`scripts/smoke-plan.sh` applies `testdata/real/<scenario>/v1`, swaps in `v2`, and runs `terraform plan`; scenarios use only the built-in `terraform_data` resource, so no providers or credentials are needed). It renders every format, takes screenshots (`scripts/screenshot.mjs`), and commits the outputs under `docs/smoke` and `docs/screenshots`. To run it locally you need Terraform >= 1.7:

```sh
go build -o tfplanlens ./cmd/tfplanlens
scripts/smoke-plan.sh complex
scripts/smoke-report.sh complex testdata/plans/real/complex-tf*.json
```

## License

MIT
