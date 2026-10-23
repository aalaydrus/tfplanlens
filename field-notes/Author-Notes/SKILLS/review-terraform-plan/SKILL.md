---
name: review-terraform-plan
description: Review a Terraform plan before it is applied. Use when the user asks what a terraform plan will do, whether it is safe to apply, what will be deleted or replaced, or to summarize a plan for a pull request. Runs the local tfplanlens CLI on a `terraform show -json` file and reports Danger, drift, target accounts and apply order.
---

# Review a Terraform plan with tfplanlens

tfplanlens reads a plan **offline** (no AWS, no Terraform run) and tells you what
matters first: deletes and replaces. Use it instead of reading raw plan output.

## 1. Make sure the tool exists

```
tfplanlens version
```

If it is not installed, download a release binary, or build it from a clone of the repo, and put it on PATH:
`go build -o tfplanlens ./cmd/tfplanlens` (Go 1.25+). There are no published
releases yet.

## 2. Get the plan as JSON

If the user gave you a plan JSON file, use it. Otherwise (and only if asked to
run Terraform):

```
terraform plan -out=plan.bin
terraform show -json plan.bin > plan.json
```

**Never print, paste or `cat` plan.json**: it is large and contains secret values
in plain text (variables, sensitive attributes). Only read tfplanlens output.

## 3. Run the report

```
tfplanlens report plan.json --no-color
```

Add `--tfapply-account-id <12 digits>` when you know which account
`terraform apply` runs from (a plan cannot say; if there is a
`tfplanlens.config.yaml` next to where you run, it supplies it). Useful variants:

```
tfplanlens report plan.json -f markdown -o report.md   # for a PR comment
tfplanlens report plan.json -f html -o report.html     # one self-contained file
tfplanlens report plan.json -f json                    # for further processing
tfplanlens report plan.json -f brief --top 10          # short and ranked: start here if tokens matter
```

`-f brief` is the cheapest way to review: only the changes that need attention,
most important first, each with the reasons. The first line says how many
qualified; raise `--top` (or `--top 0`) if some were left out. Use the full
`text` report when you need drift, accounts and waves too.

## 4. Read it in this order

1. **Danger**: every delete/replace with the plain-English reason. `[STATEFUL]`
   means data lives there (databases, buckets, keys): say so first. A
   "possible key rename" warning means a `for_each`/`count` key was removed
   while another changed: the delete may be an accident.
2. **Summary**: action counts, the apply account and its source, notices.
3. **Drift**: changes made outside Terraform. Some are provider refresh
   artifacts (`inline_policy`, `versioning`), not real changes; say when you
   think so but do not hide them.
4. **Changes by account**: where each resource lands; a flag says when the
   target account differs from the apply account.
5. **Waves**: what could run in parallel. It is a lower bound (locals and
   `replace_triggered_by` are invisible in the plan); never present it as a
   strict order.

## 5. Report back

Lead with the verdict in one line (what will be deleted/replaced, stateful or
not, which accounts). Then the few findings that need a human decision, each
with the address and the reason from the tool. Quote addresses exactly. Say
what the tool could not determine (unresolved accounts, unknown values).
If a notice says the tool "hid secret-looking values", say so: some values are
hidden on purpose (`docs/SECRETS.md`), and never try to recover them from the
plan file. A secret rotation shows as a replace of the secret version: it is routine, but
the tool lists it as Danger; say that if it applies.

Do not claim the plan is "safe". Say what the report shows.
