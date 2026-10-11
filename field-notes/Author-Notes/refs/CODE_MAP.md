# Code map

Lines are approximate; open a file by the function name, not the line.

```
cmd/tfplanlens/          the CLI (package main)
  main.go          commands: report, diff, tui, version; runReport, loadAnalysis
  multi.go         several plans: loadAll, labels (splitLabel, deriveLabels)
  apply_account.go apply-account flag/env/config resolution, --account parsing
internal/plan/     raw plan JSON -> plan.Plan (no analysis)
  plan.go          Load, normalize (one Change per resource instance), providers,
                   assume_role/allowed_account_ids/caller identity, mode detection
  deps.go          buildDeps: dependency graph from `configuration` (references,
                   depends_on, count/for_each, module calls)
internal/analyze/  plan.Plan -> Analysis (all the logic)
  analyze.go       Analyze: summary, findings, drift, groups, changes, waves, parents
  model.go         the Analysis types (the JSON contract); Change.Listed/Label
  attrs.go         attribute diff: flatten before/after, sensitive/unknown, long values
  account.go       ARN account parsing, arn_account_mismatch warnings
  failon.go        --fail-on rules: ParseRules, CheckFailOn
  reasons.go       action_reason -> plain English
  stateful.go      the built-in stateful type list
  waves.go         layering by dependency, Tarjan cycles, wave notes
  instances.go     count/for_each parents with key lists
  merge.go         Merge several analyses (labels, Units, prefixed addresses)
internal/valuediff/  long values: base64/gzip/JSON decode + line diff (valuediff.go),
                   IAM policy statement summary (policy.go)
internal/config/   tfplanlens.config.yaml (strict: unknown keys are errors)
internal/render/   one package per format, all take *analyze.Analysis
  text/ markdown/ html/ (report.html, style.css, app.js embedded) jsonout/
internal/diff/     Compare(old, new) + text/markdown/json renderers
internal/tui/      Model (pure Update/View, tested) + term.go (raw-mode loop, x/term)
testdata/plans/    hand-written fixtures (+ real/ real Terraform plans)
testdata/real/     Terraform configs that generate the real plans (v1, v2, v3)
testdata/golden/   expected outputs of the fixtures (generated; see -update)
scripts/           smoke-plan.sh (real plan), smoke-report.sh, floci-seed.sh, screenshot.mjs
simulation/        stub-AWS example with committed real plans and outputs
.github/workflows/ ci.yml (test, vet, windows), smoke.yml (real plans), release.yml
```

## Things that surprise people

- One `plan.Change` per **instance** (`aws_sqs_queue.orders["a"]`);
  `ConfigAddress` is the same without keys, and is what waves/graphs use.
- Plain no-ops are kept in `Analysis.Changes` but not "listed"
  (`Change.Listed()`); counts include them.
- `ignore_types` hides from the report but never from `--fail-on`
  (`CheckFailOn` takes the raw plan).
- Accounts are resolved per provider config key, with a recorded source; see
  SPEC §5.5 and `resolveAccount`.
- `tui`, `diff` and merged reports reuse `loadAnalysis`/`loadAll`; a new
  command that reads plans should too.
