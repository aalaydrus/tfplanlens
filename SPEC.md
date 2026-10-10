# tfplanlens: SPEC

> Read a Terraform plan the way a reviewer needs to, so a clean-looking plan stops failing at apply.

## 1. Problem

`terraform plan` output is hard to review safely:

- Deletes and replaces are buried in long, noisy output.
- Very long attribute values (EC2 `user_data`, IAM policy JSON) are unreadable on one line.
- Unclear which account or provider each resource lands in.
- No summary of what the plan does overall.
- Changes made outside Terraform (drift) are easy to miss.
- Apply order and dependencies are not visible.

## 2. Goals and non-goals

**Goals**
- Offline analysis of `terraform show -json plan.out`. No AWS credentials, no Terraform binary needed at runtime.
- Make dangerous changes (delete, replace, drift) impossible to miss, and explain *why* they happen in plain English.
- Single static binary, fast on plans with 10k+ resources.
- Deterministic output. No LLM calls anywhere.

**Non-goals (v0.1)**
- Not a policy engine (use Checkov/OPA for that).
- No AWS API calls (planned later as an opt-in `--aws` mode).
- No IAM blast-radius analysis (separate future feature).
- No apply-time features.

## 3. Language and stack

- **Go 1.23+**
- `github.com/hashicorp/terraform-json` for plan types. If a field is missing from the library (for example `action_reason`), fall back to parsing the raw JSON for that field.
- `github.com/spf13/cobra` for the CLI
- `github.com/pmezard/go-difflib` (or `sergi/go-diff`) for text diffs
- `html/template` with `embed` for the HTML report (single self-contained file: inline CSS and JS, no CDN)
- Standard library for everything else. Keep dependencies minimal.

## 4. CLI

```
tfplanlens report <plan.json> [flags]    # main command
tfplanlens version
```

Input may be a file path or `-` for stdin.

Flags for `report`:
- `--format text|html|markdown|json` (repeatable; default `text`)
- `--out <path>` (for html, markdown, json; default stdout for text and markdown)
- `--config <path>` (default `.tfplanlens.yaml` if present)
- `--fail-on <rules>` comma-separated, e.g. `delete,replace,drift,replace:aws_db_instance`
- `--max-line <n>` truncation width for terminal output (default 120)
- `--no-color`
- `--tfapply-account-id <id>`: the 12-digit account terraform apply runs from (see 5.5)
- `--account <alias>=<id>` (repeatable): target account for a provider alias; beats every other source (see 5.5)
- `--no-prompt`: never ask interactively for the apply account
- `--require-apply-account`: exit `2` instead of continuing when the apply account is unresolved (for CI)

Exit codes: `0` ok, `1` a `--fail-on` rule matched, `2` error (bad input, unsupported format).

## 5. Features for v0.1

### 5.1 Summary
- Counts by action: create, update, replace, delete, read, no-op.
- Counts by resource type and by module.
- Count of "known after apply" attributes (just a number, never listed inline).
- Whether the plan is a refresh-only or destroy plan (`mode`, `applyable`, `errored`).

### 5.2 Danger section (shown first, always)
For every resource with a delete or replace action, show:
- Address, type, account/provider group.
- **Plain-English reason** from `action_reason`:
  - `delete_because_no_resource_config`: "Removed from config"
  - `delete_because_wrong_repetition`: "Resource switched between count/for_each/single"
  - `delete_because_count_index`: "count shrank; index N no longer exists"
  - `delete_because_each_key`: "for_each key removed: <key>"
  - `delete_because_no_module`: "Whole module removed or its call changed"
  - `replace_because_cannot_update`: "Attribute change forces replacement" (list attributes from `replace_paths`)
  - `replace_because_tainted`: "Resource is tainted"
  - `replace_by_request`: "Replacement requested with -replace"
  - Unknown or missing reason: show the raw value.
- Warn loudly when a delete is in a **count/for_each shift** (likely index-shift accident), since this is the classic "state mismatch deletes resources" case.
- Mark resources whose type is on a built-in "stateful" list (databases, S3 buckets, EBS volumes, KMS keys, Route53 zones, EFS, DynamoDB tables, Elasticache, OpenSearch) with a **STATEFUL** badge.

### 5.3 Drift section
- Read `resource_drift` and show what changed outside Terraform since the last apply: address, attributes changed, before and after values (using the long-value handling in 5.4).
- Drift on a resource that is also being replaced or deleted is flagged as a high-risk finding.

### 5.4 Long-value handling
For any string attribute longer than 200 characters, or any known long-value key (`user_data`, `user_data_base64`, `policy`, `assume_role_policy`, `inline_policy`, `container_definitions`, `definition`, `body`):
1. If the value is valid base64, decode it. If it is gzip, decompress it.
2. If the result is valid JSON, pretty-print it with sorted keys. For IAM policy documents, additionally render one line per statement: `Effect Action(s) -> Resource(s)`.
3. Compute a line-level diff between before and after.
4. Display only changed lines plus 2 lines of context. Show a header like `user_data: 3 lines changed of 412 (+2 -1)`.
5. Never print the full value in the terminal or markdown output. The HTML report has an "expand full value" toggle.

### 5.5 Account / provider grouping
Plans are often run from a runner in an automation account that assumes roles into target accounts, so two different accounts matter and the report shows both:

- **Apply account** ("executed from"): where `terraform apply` runs.
- **Target account**: where a resource lands, resolved per provider config key (provider and alias).

Group `resource_changes` by `provider_config_key`. Show per-group counts, and flag every resource whose target account differs from the apply account.

**Apply account** precedence, highest first:
1. `--tfapply-account-id <12-digit id>`
2. env var `TFPLANLENS_TFAPPLY_ACCOUNT_ID`
3. `tfapply_account_id` in `.tfplanlens.yaml`
4. Interactive prompt: "Which account ID does terraform apply run from?", validated as 12 digits, with an offer to save the answer to the config file. Only when stdin and stdout are both terminals and `--no-prompt` is not set. The question goes to stderr, never into the report.
5. Unresolved: print a warning on stderr and continue. Cross-account flags are off and providers without `assume_role` stay unresolved. With `--require-apply-account`, exit `2` instead.

An invalid value (not 12 digits) from the flag, env var or config is an error (exit `2`). The source used is recorded and shown (`flag`, `env`, `config`, `prompt`, `unresolved`).

**Target account** is resolved per provider config key, first match wins, and the source is recorded and shown in every report format:

| | Source | Recorded as |
|---|---|---|
| a | `--account <alias>=<id>` (repeatable; matches the alias or the full key such as `aws.prod`; `default` matches the unaliased provider) | `flag` |
| b | `accounts:` mapping in `.tfplanlens.yaml` (same keys) | `config` |
| c | `assume_role.role_arn`, literal, or a plain variable reference resolved through the plan's `variables`; the account ID is parsed from the ARN | `role_arn`, `role_arn_variable` |
| d | `allowed_account_ids`, when it holds exactly one value (literal or resolved through `variables`) | `allowed_account_ids` |
| e | `account_id` of an `aws_caller_identity` data source in `prior_state` that uses this provider, when exactly one distinct value | `caller_identity` |
| f | A provider with **no** `assume_role` runs as the apply account | `apply_account` |
| g | Otherwise label it `unresolved (alias: <name>)` | `unresolved` |

A provider that has `assume_role` but whose role ARN cannot be resolved never falls back to (f); it is unresolved.

**Warnings.** When an ARN-typed attribute of a resource (`arn`, or a name ending in `_arn` / `_arns`) contains an account ID that differs from its provider's resolved target account, show a warning naming the attribute, the ARN's account, the target account, and the source. Sensitive and unknown values are skipped, and so are resources that are not changing.

### 5.6 Moves, imports, removals
- Resources with `previous_address` display as **moved** (not delete and create).
- Resources with `change.importing` display as **import**.
- Resources removed via a `removed` block (forget, not destroy) display as **removed from state only**.

### 5.7 Sensitive values
- Where `before_sensitive` or `after_sensitive` marks a value, show `(sensitive value changed)` and never print the value.

### 5.8 Output formats
- **text**: colored terminal output with sections in this order: Summary, Danger, Drift, Changes by account, Changes by module. Collapse long lists (show the first 10, then "+N more").
- **html**: one self-contained file with a dashboard: summary tiles, action donut or bar chart (inline SVG, no libraries), a Danger panel at the top, filterable and searchable table of all changes, per-resource detail drawers with attribute diffs, grouping toggle (account / module / type). Works offline, supports dark mode.
- **markdown**: for PR comments. Summary table, Danger section expanded, everything else inside `<details>` blocks per account then module. Hard cap 60,000 characters (GitHub comment limit is 65,536). If exceeded, truncate and add a note.
- **json**: a stable, documented schema of the analysis result (versioned with `"schema_version": 1`).

### 5.9 Config file `.tfplanlens.yaml`
```yaml
tfapply_account_id: "999999999999"   # account terraform apply runs from (5.5)
accounts:                # provider alias (or key such as aws.prod) -> target account
  prod-alias: "111111111111"
  shared-alias: "222222222222"
stateful_types:         # added to the built-in list
  - aws_rds_cluster
long_value_keys:        # added to the built-in list
  - custom_script
ignore_types: []        # resource types to hide from the report (never from --fail-on)
```

## 6. Architecture

```
cmd/tfplanlens/main.go
internal/plan/        # load and normalize plan JSON into internal model
internal/analyze/     # summary, danger findings, drift, grouping
internal/valuediff/   # long-value decode and diff
internal/render/
    text/
    html/             # templates embedded with go:embed
    markdown/
    jsonout/
internal/config/
testdata/plans/       # fixture plans (see 7)
testdata/golden/      # expected outputs
```

Internal model: `Analysis{ Summary, Findings[], Drift[], Groups[], Changes[] }`. Renderers consume only `Analysis`, never raw plan JSON, so v0.2 features (waves and so on) only touch `analyze`.

## 7. Testing

- **Fixtures** in `testdata/plans/`, each a `terraform show -json` style file. Hand-author them if no real plans are available. Required fixtures:
  1. `basic-create.json`: simple creates and updates
  2. `deletes-and-replaces.json`: covers every `action_reason`
  3. `for-each-shift.json`: for_each key removed, and a count index shift
  4. `long-user-data.json`: base64 user_data change with 400+ lines
  5. `iam-policy-change.json`: assume_role_policy and policy diff
  6. `drift.json`: populated `resource_drift`
  7. `multi-account.json`: two provider aliases with assume_role ARNs
  8. `moved-import-removed.json`
  9. `sensitive.json`
  10. `large.json`: 5,000+ generated resources (generate in test, don't commit)
- **Golden tests** for text, markdown and json output. Update with `go test -update`.
- HTML: test that the output is valid, self-contained (no `http://` or `https://` asset references) and contains expected finding text.
- Unit tests for base64/gzip detection, the policy statement renderer, and ARN account parsing.
- Performance test: `large.json` must analyze and render in under 2 seconds.
- CI never needs AWS or Terraform.

## 8. CI and release

- GitHub Actions: `go vet`, `staticcheck`, `go test ./...` on Linux, macOS and Windows.
- `goreleaser` on tag `v*`: binaries for linux/darwin/windows (amd64, arm64), checksums, and a Homebrew tap formula.
- A reusable GitHub Action (`action.yml`) that runs `tfplanlens report` and posts or updates a single PR comment (identified by a hidden marker).
- README includes: 30-second quickstart, a screenshot or GIF of the HTML report, a sample PR comment, and a "why not just read terraform plan?" section.

## 9. Acceptance criteria for v0.1

- [ ] `tfplanlens report fixture.json` runs on all fixtures with no errors.
- [ ] Every `action_reason` is rendered with a plain-English explanation.
- [ ] A 400-line `user_data` change is displayed as only its changed lines plus context.
- [ ] Deletes and replaces appear in the Danger section before anything else in text, html and markdown.
- [ ] Resources are grouped by account, with an `unresolved` fallback.
- [ ] `--fail-on delete` exits `1` when any delete exists, and `0` otherwise.
- [ ] HTML report opens offline and is a single file.
- [ ] Markdown output stays under 60,000 characters on `large.json`.
- [ ] README and LICENSE (MIT) are present.

## 10. Roadmap (do not build in v0.1)

**v0.2**
- Apply **waves** from the configuration's references and `depends_on`: wave 1 has no dependencies, wave 2 depends only on wave 1, and so on. Note that Terraform applies in parallel, so this shows possible parallelism, not a strict order.
- Detect and display dependency cycles.
- Group `count` and `for_each` instances under their parent with an expandable key list.
- Terminal UI (Bubble Tea) with expand/collapse and filters.

**v0.3**
- Multi-plan input (Terragrunt or monorepo), merged into one report.
- Opt-in `--aws` mode: read-only checks for external references to replaced or deleted resources (security group attachments, role trust, KMS grants, DNS records).
- Plan-to-plan diff: "what changed since the last time I reviewed this plan?"

**Later**
- IAM blast radius for roles and security groups in the plan.
- Kiro skill wrapper.
- Artifacts bucket for storing reports.
