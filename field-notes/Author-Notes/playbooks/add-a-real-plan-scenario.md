# Playbook: add a real-plan scenario

Use when a Terraform behavior must be proven on a real plan.

1. `testdata/real/<name>/v1/main.tf` (applied first), `v2/main.tf` (planned).
   Optional: `v3/` (a second plan, written as `<name>-after-tf<ver>.json`),
   `between.sh` (out-of-band changes), `plan-args` (e.g. `-destroy`), `env`,
   `requires-floci` (AWS provider against the emulator). Most scenarios use
   only `terraform_data`, no credentials.
2. Generate: `scripts/smoke-plan.sh <name>` (needs terraform >= 1.7; offline
   providers via `SMOKE_PLUGIN_DIR`). It writes
   `testdata/plans/real/<name>-tf<version>.json`.
3. Look at it with `jq` ([../refs/DO_NOT_READ.md](../refs/DO_NOT_READ.md)).
4. Test in `cmd/tfplanlens/real_plans_test.go`: use `realPlans(t, "<name>")`
   and `analyzeReal`; assert addresses/actions/counts, never bytes (ids differ
   each CI run). Skip, do not fail, when the file is absent.
5. CI regenerates every scenario (`smoke.yml` loops over `testdata/real/*/`),
   runs the real-plan tests on the fresh plans and uploads the outputs, but
   commits nothing: commit the plan files yourself (`scripts/smoke-refresh.sh`
   regenerates `testdata/plans/real`, `docs/smoke` and optionally screenshots).
6. AWS shapes: floci setup and quirks in
   [../../TERRAFORM_GOTCHAS.md](../../TERRAFORM_GOTCHAS.md); the bigger worked
   example is `simulation/regenerate.sh`.
