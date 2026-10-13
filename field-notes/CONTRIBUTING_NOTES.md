# Contributing notes

How we have been working on tfplanlens; the habits that caught real bugs.

## Build and test

```
go build -o tfplanlens ./cmd/tfplanlens
go vet ./... && go test ./... -count=1
GOOS=windows go vet ./...        # CI also runs Windows
go test ./cmd/... -run TestGolden -update   # regenerate goldens, then READ the diff
```

Go 1.23 (`go.mod` says `go 1.23.0`). CI installs whatever `go.mod` asks for, so
a dependency bump to `go 1.25.0` once stayed green; `TestGoModKeepsTheMinimumGoVersion`
now fails it. To move the minimum on purpose, change that test and every doc that
names "1.23".
Dependencies are deliberately few: cobra, terraform-json, yaml.v3, go-difflib,
`x/term`; `x/net` is test-only (HTML tokenizer).

## The loop that works: real plan first

1. Make a real `terraform show -json` plan (`scripts/smoke-plan.sh <scenario>`,
   or `simulation/regenerate.sh`), not a hand-edited one.
2. Run the tool, compare with what Terraform says.
3. Fix, and add a test that **fails on the old code** (revert the fix and
   check). Assert on addresses, actions and counts, not byte-exact output,
   for real plans (resource ids change per run); goldens are for hand fixtures.
4. Look at it: HTML in a browser (Playwright, screenshots), TUI in a pty.

Several bugs only showed up this way (null JSON arrays, wave badges on
no-ops, Bubble Tea's terminal query, secrets appearing in plan JSON).

## Pull requests

- Branch names start with `feat/` (not `claude/`).
- One feature per PR; do not open a PR until asked. The user can ask for
  several features in one PR; say so in the description, by section.
- In the description, list only what was verified, with the command or CI run;
  paste `go test` output where it matters; say plainly what was **not**
  verified. No check marks for things that were not run.
- A merged PR is finished: new work goes on a new branch from the default.
- Never put a model name in commits, code or PR text.

## Generated files: regenerate after a change that affects output

```
go test ./cmd/... -run TestGolden -update      # goldens, then READ git diff testdata
simulation/demo.sh                              # simulation/output (also tested)
for p in testdata/plans/real/*.json; do scripts/smoke-report.sh "$(basename $p | sed 's/-tf.*//')" "$p"; done   # docs/smoke, no Terraform needed
```

`docs/readme/html-report.png` (README and About page) and `docs/screenshots` are
taken by hand with Playwright against a built report; they go stale silently.

## Conventions

- Arrays in JSON output are `[]`, never `null` (`TestJSONArraysAreNeverNull`).
- Renderers consume only `analyze.Analysis`, never raw plan JSON.
- Sensitive values are never printed (`TestSimulationNeverPrintsSecretValues`).
- Report only what the plan says; when the plan cannot tell (a local value, the
  backend), say so in a note instead of guessing.
- Keep the CLI small: new flags need a reason; settings go in
  `tfplanlens.config.yaml`.

## Testing against AWS without AWS

floci (a local AWS emulator): `docker run -d --name floci --network host -v /var/run/docker.sock:/var/run/docker.sock -u root floci/floci:latest`.
Use a fresh container per run. See `TERRAFORM_GOTCHAS.md` for its quirks and
`scripts/floci-seed.sh` for seeding IAM roles. Offline provider install:
`SMOKE_PLUGIN_DIR` / `SIMULATION_PLUGIN_DIR` pointing at a provider mirror.

## Adding a note

Found a surprising plan, a provider quirk or an idea? Add it to
`TERRAFORM_GOTCHAS.md` or `WISHLIST.md` with the evidence (a file path and the
versions). Mark it `verified` only if you reproduced it.
