# tfplanlens

Read a Terraform plan the way a reviewer needs to. Deletes and replaces first, with the reason in plain English.

It works **offline** on the JSON from `terraform show -json`: no AWS credentials, no Terraform binary, no network, no LLM. Deterministic output, and sensitive values are never printed.

![tfplanlens HTML report: Danger panel, summary tiles and a filterable table](docs/readme/html-report.png)

## Quickstart

```sh
git clone https://github.com/aalaydrus/tfplanlens && cd tfplanlens
go install ./cmd/tfplanlens            # Go 1.23+, no releases yet; put $(go env GOPATH)/bin on PATH

terraform plan -out plan.bin
terraform show -json plan.bin > plan.json
tfplanlens report plan.json
```

## What you get

- **Danger first**: every delete and replace with why, a STATEFUL badge, and a warning when a `count`/`for_each` change looks like an accidental shift.
- **Which account**: where `terraform apply` runs vs where each resource lands, and how that was decided.
- **Short**: long values become the lines that changed; unknown values are counted, not printed.
- **Also**: drift, moves, imports, apply waves, and a diff between two plans.

## Commands

```sh
tfplanlens report plan.json -f html -o report.html      # one offline HTML file
tfplanlens report plan.json -f markdown -o report.md    # a PR comment
tfplanlens report plan.json --fail-on delete,replace:aws_db_instance   # exit 1 in CI
tfplanlens report a.json b.json                         # several plans, one report
tfplanlens diff before.json after.json                  # what the new push changed
tfplanlens tui plan.json                                # explore interactively
```

Settings (apply account, default formats, `--fail-on` rules) go in an optional [`tfplanlens.config.yaml`](tfplanlens.config.yaml).

## Try it without AWS

[`simulation/`](simulation/) has a made-up AWS stack with real plans and every output: start with [`EXAMPLE_RUNDOWN.md`](simulation/rundown/EXAMPLE_RUNDOWN.md).

## More

[Usage and flags](docs/USAGE.md) · [Accounts](docs/ACCOUNTS.md) · [Why not just read `terraform plan`?](docs/WHY.md) · [JSON schema](docs/json-schema.md) · [Development and smoke tests](docs/DEVELOPMENT.md) · [Working notes](field-notes/) · [SPEC](SPEC.md)

MIT
