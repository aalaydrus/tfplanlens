# Agents: start here

tfplanlens is a Go CLI that reads `terraform show -json` plans **offline** and
reports what matters: Danger (deletes/replaces), drift, target accounts, waves,
diffs between plans. It never touches AWS or Terraform. Read this page, then
open only the one ref or playbook your task points to.

## Route your task (read the right file, nothing else)

| You want to... | Read | Then edit |
|---|---|---|
| Fix a bug | [playbooks/fix-a-bug.md](playbooks/fix-a-bug.md) | the package in [refs/CODE_MAP.md](refs/CODE_MAP.md) |
| Show a new fact in reports (new field/section) | [playbooks/add-an-output-field.md](playbooks/add-an-output-field.md) | `internal/analyze`, then the renderers |
| Add a CLI flag, command or config key | [playbooks/add-a-flag-or-config-key.md](playbooks/add-a-flag-or-config-key.md) | `cmd/tfplanlens/main.go`, `internal/config` |
| Handle a Terraform/provider case we get wrong | [playbooks/add-a-real-plan-scenario.md](playbooks/add-a-real-plan-scenario.md) | `testdata/real/`, `scripts/smoke-plan.sh` |
| Understand data flow / the model | [refs/CODE_MAP.md](refs/CODE_MAP.md), [refs/DATA_MODEL.md](refs/DATA_MODEL.md) | |
| Know which test guards what | [refs/TESTING_MAP.md](refs/TESTING_MAP.md) | |
| Know every flag, exit code, config key | [refs/CLI_REFERENCE.md](refs/CLI_REFERENCE.md) | |
| Open a PR | [playbooks/open-a-pr.md](playbooks/open-a-pr.md) | |
| Use tfplanlens as an agent skill | [SKILLS/README.md](SKILLS/README.md) | |
| Check a Terraform/provider quirk or idea | [../TERRAFORM_GOTCHAS.md](../TERRAFORM_GOTCHAS.md), [../WISHLIST.md](../WISHLIST.md) | |

## Five facts that save the most tokens

1. **Data flow**: `plan.Load` (raw JSON → `plan.Plan`) → `analyze.Analyze`
   (→ `analyze.Analysis`, the only thing renderers see) → `internal/render/*`
   or `internal/diff` / `internal/tui`. Several plans: `analyze.Merge`.
2. **Renderers never read plan JSON**; new facts go in `analyze` first.
3. **Plan files are big single-line JSON (80–90 KB). Never `cat` them**; query
   with `jq` (recipes in [refs/DO_NOT_READ.md](refs/DO_NOT_READ.md)).
4. **Goldens are generated**: change code, run
   `go test ./cmd/... -run TestGolden -update`, then read `git diff testdata`.
5. **Real plan first**: reproduce with a real `terraform show -json` plan
   (`testdata/plans/real/`, `simulation/assets/input/`), not a hand edit.

## Commands

```
go build -o tfplanlens ./cmd/tfplanlens
go vet ./... && go test ./... -count=1        # ~3 s, run before every push
GOOS=windows go vet ./...                      # CI also runs Windows
go test ./internal/analyze -run TestWaves -v   # one package/test is fast
```

Go 1.23 (`go 1.23.0` in `go.mod`): do not let a dependency bump it. Few
dependencies on purpose (cobra, terraform-json, yaml.v3, go-difflib, x/term).

## Invariants (tests enforce these; do not break them)

- JSON arrays are `[]`, never `null` (`TestJSONArraysAreNeverNull`).
- Sensitive values are never printed anywhere (`TestSensitiveValuesNeverPrinted`,
  `TestSimulationNeverPrintsSecretValues`).
- Danger comes before any change listing (`TestDangerBeforeAnyChangeListing`).
- The CLI stays small: `TestCLISurfaceIsLean` lists the allowed `report` flags.
- Say "unknown" when the plan cannot tell (backend, `local.*`, cross-plan
  dependencies); do not guess.
- Docs that name files or commands are checked by `TestAgentDocs*`.

## Where the project's intent lives (read only if needed)

`SPEC.md` is long: read a section by header (`grep -n '^## ' SPEC.md`):
§5 features, §6 architecture, §9 v0.1 acceptance, §10 roadmap (status).
`docs/json-schema.md` is the output contract; `docs/acceptance.md` maps
criteria to tests; `README.md` is user-facing.

## Status

v0.1, v0.2 (waves, cycles, instance groups, `tui`) and v0.3 except `--aws`
are done: `diff`, multi-plan reports. Open ideas: [../WISHLIST.md](../WISHLIST.md).
