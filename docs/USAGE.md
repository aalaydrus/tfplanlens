# Usage

Install, every command, flag and setting. For the short version see the [README](../README.md).

## Install

tfplanlens is installed from a local clone; there are no published releases and no Homebrew formula.

```sh
git clone https://github.com/aalaydrus/tfplanlens
cd tfplanlens
go install ./cmd/tfplanlens        # installs to $(go env GOPATH)/bin (or $GOBIN)
tfplanlens version                 # e.g. "dev (8814cf7)": the git revision it was built from
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`. To update, `git pull` and run `go install ./cmd/tfplanlens` again. To build a binary in place instead: `go build -o tfplanlens ./cmd/tfplanlens`.

## Commands and flags

Pass `-` as the plan to read stdin: `terraform show -json plan.bin | tfplanlens report -`.

```
tfplanlens report <plan.json|-> [more plans...] [flags]
tfplanlens diff <old.json> <new.json> [-f text|markdown|json] [-o path] [--fail-on-new-danger]
tfplanlens tui <plan.json> [more plans...]
tfplanlens version
```

Flags of `report` (`tui` takes `--config`, `--tfapply-account-id` and `--account`; `diff` also takes `-f`, `-o` and `--fail-on-new-danger`):

```
  -f, --format <text|html|markdown|json>   output format; repeatable (default text)
  -o, --out <path>                         write to a file instead of stdout
  -c, --config <path>                      config file (default ./tfplanlens.config.yaml if present)
      --fail-on <rules>                    exit 1 on: delete, replace, drift, unresolved, or kind:type
      --tfapply-account-id <id>            account terraform apply runs from
      --account <alias=id>                 target account for a provider alias; repeatable
      --no-color
```

Exit codes: `0` ok, `1` a `--fail-on` rule matched, `2` error.

`--fail-on` takes comma-separated rules: `delete`, `replace`, `drift`, `unresolved` (any account the report could not determine), and `kind:type` to scope the first three, e.g. `replace:aws_db_instance`.

## Config file

Everything else lives in an optional `tfplanlens.config.yaml` in the directory you run from (or the file given with `-c`). Flags override it. [tfplanlens.config.yaml](../tfplanlens.config.yaml) in this repo is a commented template with every key; copy it next to your Terraform code.

```yaml
tfapply_account_id: "999999999999"   # where terraform apply runs (else inferred from the default provider)
accounts:                            # provider alias or key (aws.prod) -> account
  prod: "111111111111"
format: [text]                       # default output format(s)
fail_on: [delete, "replace:aws_db_instance"]
max_line: 120                        # terminal truncation width
stateful_types: [aws_rds_cluster]    # added to the built-in list
long_value_keys: [custom_script]     # added to the built-in list
ignore_types: []                     # hidden from the report, never from --fail-on
```

Unknown keys and invalid values are errors, so a typo cannot silently do nothing.

## Output formats

| Format | What you get |
|---|---|
| `text` | colored terminal report: Summary, Danger, Drift, Changes by account, Changes by module |
| `markdown` | for PR comments: summary table, Danger expanded, the rest in `<details>` per account then module; capped at 60,000 characters |
| `html` | one self-contained file (no CDN, works offline, dark mode): Danger panel, tiles, chart, searchable and groupable table, per-resource detail drawers |
| `json` | the analysis as stable JSON, see [docs/json-schema.md](json-schema.md) |

Every format shows the apply account and each resource's target account with the source that decided it.

## What the reports look like

The HTML report is one file with no external requests (works offline, dark mode). Each resource opens a drawer with its attribute diffs; this is the decoded 400-line `user_data` change:

![Detail drawer showing a decoded user_data diff: 2 lines removed, 2 added, of 401](readme/html-drawer-user-data.png)

The markdown report is built for PR comments: a summary table, the Danger section expanded, everything else in collapsible `<details>` per account then module, capped at 60,000 characters. This is the top of the one produced for that same plan (the full report is in [readme/sample-pr-comment.md](readme/sample-pr-comment.md)):

<details>
<summary>Sample PR comment (summary and Danger section)</summary>

**tfplanlens plan report**

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 1 |
| update | 4 |
| replace | 1 |
| delete | 3 |
| no-op | 6 |
| forget | 1 |
| Drift | 0 |
| Known after apply | 10 attribute(s) |

**Danger**

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `terraform_data.db` | `Attribute change forces replacement: triggers_replace` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.subnet["b"]` | `for_each key removed: b`<br>**WARNING:** `for_each key removed while sibling instances change (possible key rename: terraform_data.subnet["d"])` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.worker[2]` | `count shrank; index 2 no longer exists` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.worker[3]` | `count shrank; index 3 no longer exists` | `111111111111` (source: apply_account) |

</details>

## Waves and instance groups

Every report ends with apply and destroy **waves** worked out from the
configuration's references and `depends_on`: wave 1 has no dependency on another
changing resource, wave 2 depends only on wave 1, and so on (destroys run in
reverse). Terraform applies in parallel, so this shows possible parallelism, not a
strict order, and it is a lower bound because `local.*` references and
`replace_triggered_by` are not in the plan JSON (the report says so when it
applies). A dependency cycle is flagged prominently. `count` and `for_each`
resources are listed once with their keys, and the HTML report can group the table
by resource.

## Several plans in one report

```
tfplanlens report network/plan.json app/plan.json -f html -o all.html
tfplanlens report net=a.json app=b.json --fail-on delete
tfplanlens tui network/plan.json app/plan.json
```

Each plan is analyzed on its own (its own accounts, waves and dependencies), then merged: every address is prefixed with the plan's label (`vpc:aws_vpc.main`), so identical addresses in different plans never collide, counts are added up, and a **Plans** table shows each plan's mode, apply account and counts. A plan's label is its file name, or its directory when the file is called `plan.json`/`tfplan.json` (more path elements are added until the labels differ), or whatever you give with `label=path`. `--fail-on` is checked per plan and names the plan. In HTML and the TUI you can group by plan.

Limits: dependencies *between* plans (one root module reading another's outputs) are not in the plan JSON, so waves are only exact within a plan (the report says so); text inside a warning (such as a possible key rename) names the address without the plan prefix; `-` (stdin) works only with a single plan; `diff` still compares exactly two plans.

## What changed since the last plan

```
tfplanlens diff before.json after.json
tfplanlens diff before.json after.json -f markdown -o diff.md
tfplanlens diff before.json after.json --fail-on-new-danger
```

Compares two plans of the same configuration by resource address and tells you only what the new push changed: resources that are **only in the new plan**, ones that **no longer change**, and ones that change **differently** (action, reason, target account, and which attributes). It calls out anything the new plan now deletes or replaces that the old one did not (`--fail-on-new-danger` exits 1 on that, after printing the diff), anything that is no longer deleted or replaced, and drift that appeared or went away. A plain no-op counts as "not changing". A resource the new plan moved (a `moved` block) from an address the old plan listed is treated as the same resource. Formats: text, markdown (for a PR comment) and json. Sensitive values are never compared, so two sensitive changes to the same attribute count as equal.

## Interactive terminal view

```
tfplanlens tui plan.json
```

An expandable tree of the changes: `→`/enter expands a resource to its reason, warnings and attribute diffs, and `count`/`for_each` instances sit under one parent row. `/` searches, `f` cycles the action filter, `g` groups by account or module, `n` shows plain no-ops, `w` switches to the waves view, `e`/`c` expand or collapse everything, `q` quits. It needs a terminal (use `report` for pipes and files) and takes the same `--config`, `--tfapply-account-id` and `--account` options as `report`.

Account resolution (apply account, target account, sources) is in [ACCOUNTS.md](ACCOUNTS.md).
