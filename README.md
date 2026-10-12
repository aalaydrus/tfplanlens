# tfplanlens

Read a Terraform plan the way a reviewer needs to: deletes and replaces first, with the reason in plain English.

It works **offline** on the JSON from `terraform show -json`: no AWS credentials, no Terraform binary, no network, no LLM. Output is deterministic. Values Terraform marks sensitive are never printed, and secret-looking values it does not mark are hidden too (best effort: [what is and is not hidden](docs/SECRETS.md)).

![tfplanlens HTML report: a Danger side panel next to summary tiles and a filterable table](docs/readme/html-report.png)

**Where to start:** open **Quickstart** below (about 30 seconds), or look at a finished example in [`simulation/`](simulation/rundown/EXAMPLE_RUNDOWN.md) without installing anything.

<details>
<summary><b>Quickstart</b>: install and read your first plan</summary>

```sh
git clone https://github.com/aalaydrus/tfplanlens && cd tfplanlens
go install ./cmd/tfplanlens            # Go 1.23+, no releases yet; put $(go env GOPATH)/bin on PATH

terraform plan -out plan.bin
terraform show -json plan.bin > plan.json
tfplanlens report plan.json
```

</details>

<details>
<summary><b>What you get</b></summary>

- **Danger first, ranked**: every delete and replace with why, a STATEFUL badge, what depends on it (a lower bound), deletion protection and final snapshot where the plan says, and a warning when a `count`/`for_each` change looks like an accidental shift. Stateful and unprotected come first, then by dependents.
- **Which account**: where `terraform apply` runs vs where each resource lands, and how that was decided.
- **Short**: long values become the lines that changed; unknown values are counted, not printed.
- **Also**: drift, moves, imports, apply waves, and a diff between two plans.

</details>

<details>
<summary><b>Commands</b></summary>

```sh
tfplanlens report plan.json -f html -o report.html      # one offline HTML file
tfplanlens report plan.json -f markdown -o report.md    # a PR comment
tfplanlens report plan.json --fail-on delete,replace:aws_db_instance   # exit 1 in CI
tfplanlens report a.json b.json                         # several plans, one report
tfplanlens diff before.json after.json                  # what the new push changed
tfplanlens tui plan.json                                # explore interactively
```

Settings (apply account, default formats, `--fail-on` rules) go in an optional [`tfplanlens.config.yaml`](tfplanlens.config.yaml).

</details>

<details>
<summary><b>For agents</b>: navigate this repo cheaply</summary>

Working on tfplanlens with an AI agent? Point it at [`AGENTS_START_HERE.md`](field-notes/Author-Notes/AGENTS_START_HERE.md): a short routing page (about 4 KB) that sends each kind of task to the one file it needs, instead of reading the whole repo. There are also ready-made [skills](field-notes/Author-Notes/SKILLS/README.md) for Claude and Kiro, and notes on [searching the docs with qmd](field-notes/Author-Notes/refs/SEARCH_WITH_QMD.md) (an experiment: optional, not required).

</details>

**More:** [Usage and flags](docs/USAGE.md) · [Accounts](docs/ACCOUNTS.md) · [Secrets](docs/SECRETS.md) · [GitHub Pages](docs/PAGES.md) · [Why not just read `terraform plan`?](docs/WHY.md) · [JSON schema](docs/json-schema.md) · [Development](docs/DEVELOPMENT.md) · [Working notes](field-notes/) · [SPEC](SPEC.md)

MIT. [Contributing](CONTRIBUTING.md) · [Security](SECURITY.md) · [Third-party notices](THIRD_PARTY_NOTICES.md). Not affiliated with HashiCorp or AWS.
