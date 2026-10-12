# Playbook: open a PR

- Only when asked. One feature per PR, branch from the default branch; a merged
  PR is finished, new work gets a new branch.
- Before pushing: `go vet ./... && go test ./... -count=1` and
  `GOOS=windows go vet ./...`.
- Description: what changed and why; **Verified** (the commands, with output
  where it matters, or CI run links); **Not verified** (say it plainly: CI not
  yet run, no real AWS, anything you skipped). No check marks for things that
  were not run.
- Commit messages: imperative, why not what. No model names anywhere in the
  repo, commits or PR text.
- Generated files: if you changed outputs, regenerate goldens
  (`-update`) and `simulation/demo.sh` and say so.
- Do not commit real plans or state: they contain secrets in plain text.
- Nothing commits to your branch behind your back: if you changed `testdata/real`,
  run `scripts/smoke-refresh.sh` and commit the regenerated plans and reports.
