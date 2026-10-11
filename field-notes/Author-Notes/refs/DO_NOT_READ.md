# Do not read these whole (and how to peek instead)

| Path | Why | Peek with |
|---|---|---|
| `testdata/plans/**/*.json`, `simulation/assets/input/*.json` | one-line JSON, 80–90 KB each | `jq` below |
| `simulation/assets/input/terraform.tfstate` | 600 lines of state | `jq '.resources[].type' ...` |
| `testdata/golden/*` | generated | regenerate with `-update`, read `git diff testdata` |
| `simulation/output/*`, `docs/smoke/**` | generated (the latter by a CI bot) | `./simulation/demo.sh`; ignore |
| `README.md` (260 lines), `SPEC.md` (230) | user/spec docs | `grep -n '^## ' FILE`, then `sed -n 'A,Bp'` |
| `internal/render/html/report.html`, `style.css`, `app.js` | only for HTML work | |
| `node_modules/`, `tfplanlens` (binary), `go.sum` | noise | |

## `jq` recipes for a plan

```
jq '.resource_changes[] | {address, actions: .change.actions, reason: .action_reason}' plan.json
jq '.resource_changes[] | select(.change.actions != ["no-op"]) | .address' plan.json
jq '.resource_changes[] | select(.address=="aws_x.y") | .change | {before, after, after_unknown}' plan.json
jq -c '.configuration.root_module.resources[] | {address, refs: ([.expressions // {} | .. | .references? // empty | .[]] | unique)}' plan.json
jq '{terraform_version, resource_drift: (.resource_drift|length), vars: .variables}' plan.json
```

Careful: plan JSON holds secret values in plain text (`variables`, before/after
of sensitive attributes). Do not paste raw values into issues, PRs or chats.
