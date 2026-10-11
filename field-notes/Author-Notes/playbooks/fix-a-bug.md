# Playbook: fix a bug

1. **Reproduce with a real plan.** If the bug is about a Terraform/provider
   shape, get the real JSON: `scripts/smoke-plan.sh <scenario>` or a scenario
   under `testdata/real/` (see [add-a-real-plan-scenario.md](add-a-real-plan-scenario.md)).
   Query it with `jq` ([../refs/DO_NOT_READ.md](../refs/DO_NOT_READ.md)); do not
   open it.
2. **Find the owner** in [../refs/CODE_MAP.md](../refs/CODE_MAP.md). Wrong value
   in the report -> `internal/analyze`; wrong parse of the plan -> `internal/plan`;
   wrong layout/escaping -> the renderer; wrong flag/exit code -> `cmd/tfplanlens`.
3. **Write the failing test first** in the same package, asserting the
   behavior (addresses, actions, counts, text snippets), not whole outputs.
   Run it and watch it fail.
4. **Fix** the smallest thing. Keep arrays non-nil, never print sensitive values.
5. **Prove the test guards the fix**: revert the fix (`git stash` the source
   file), confirm the test fails, restore.
6. `go vet ./... && go test ./... -count=1`. If goldens changed, they must differ
   only by what you intended (`git diff testdata`).
7. If the bug taught us something about Terraform or a provider, add an entry
   to [../../TERRAFORM_GOTCHAS.md](../../TERRAFORM_GOTCHAS.md) (`verified` only if
   reproduced).
8. PR: [open-a-pr.md](open-a-pr.md).
