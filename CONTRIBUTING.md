# Contributing

Thanks for helping. tfplanlens is a small Go CLI that reads `terraform show -json`
plans offline.

## The one rule: never commit a real plan or state file

Plan JSON and Terraform state hold **secret values in plain text** (variables,
sensitive attributes, passwords). Before every commit, check that you are not
adding `plan.json`, `plan.bin`, `*.tfstate` or `*.tfvars` from a real environment.
`.gitignore` blocks them except under `testdata/` and `simulation/`, which hold
fabricated examples. If you need a plan to reproduce a bug, generate one from a
throwaway configuration (see `testdata/real/` and `simulation/regenerate.sh`) or
write a minimal one by hand. If you committed a real one by accident, tell a
maintainer **before** pushing: it must be removed from history, not just deleted.

## Set up

```sh
git clone https://github.com/aalaydrus/tfplanlens && cd tfplanlens
go build -o tfplanlens ./cmd/tfplanlens
go vet ./... && go test ./... -count=1        # about 3 seconds; Go 1.23+
```

You do not need Terraform, AWS or Docker for the tests.

## Contribute from a fork

1. Fork the repository and branch from `main`.
2. Make one focused change, with a test that fails without it.
3. Run `go vet ./...`, `go test ./... -count=1` and `GOOS=windows go vet ./...`.
4. Open a pull request. Say what changed, what you verified (commands or CI),
   and what you did not verify.

CI runs the tests on Linux, macOS and Windows plus `staticcheck`. Pull requests
from forks run with a read-only token, so some jobs that write results back are
skipped for them.

## Where things are

- [`field-notes/Author-Notes/AGENTS_START_HERE.md`](field-notes/Author-Notes/AGENTS_START_HERE.md)
  is the short map for people and AI agents alike: what is where, which test
  guards what, and a playbook for common tasks (fix a bug, add a flag, add an
  output field, add a real-plan scenario).
- [`docs/`](docs/USAGE.md) are the user docs; [`SPEC.md`](SPEC.md) is the design.
- [`field-notes/`](field-notes/README.md) records plan and provider gotchas and a
  wishlist of ideas, a good place to find something to work on.

## Conventions

- Reproduce with a real plan first, then fix, then prove the new test fails on the
  old code.
- Never print sensitive values, and keep the redaction (`internal/redact`) in
  step: a new output path that shows plan values must go through it; a new
  credential format is a pattern plus a test in `internal/redact`.
- Never print sensitive values; JSON arrays are `[]`, never `null`; say "unknown"
  when the plan cannot tell instead of guessing. Tests enforce these.
- Keep the CLI small: prefer a config key to a flag, and a default to either.
- Do not add dependencies lightly (the list is short on purpose).
- Generated files (goldens in `testdata/golden`, `simulation/output`) are
  regenerated, not hand-edited; see the playbooks.

## License

By contributing you agree that your contribution is licensed under the MIT
License of this project.
