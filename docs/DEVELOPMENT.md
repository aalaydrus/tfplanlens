# Development: releases and smoke tests

Other working notes: [`field-notes/`](../field-notes/README.md).

## Releases

There are no published releases. `.goreleaser.yaml` and `.github/workflows/release.yml` are in place (binaries for linux, darwin and windows, checksums, and an optional Homebrew formula if a tap and token are ever configured), but nothing is tagged. A tag `v*` would run them; CI already runs `goreleaser check` and a snapshot build on every pull request.

## Smoke tests on real plans

`.github/workflows/smoke.yml` runs tfplanlens against real Terraform plans. For each scenario under `testdata/real/`, `scripts/smoke-plan.sh` applies `v1`, swaps in `v2`, and runs `terraform plan` (optionally with extra flags from `plan-args`, and an out-of-band step from `between.sh`). It then renders every format, takes screenshots (`scripts/screenshot.mjs`), and commits the outputs under `docs/smoke` and `docs/screenshots`.

| Scenario | Covers |
|---|---|
| `simple` | create, update, replace (`triggers_replace`), delete (removed from config) |
| `complex` | `for_each` and `count` deletes, a `removed` block, a moved module, a sensitive variable, a 400-line base64 `user_data` |
| `destroy` | `plan -destroy` |
| `import` | an `import` block plus an update |
| `reasons` | `replace_by_triggers`, `replace_by_request` (`-replace`), `replace_because_tainted`, `delete_because_wrong_repetition`, `delete_because_no_module`, an implicit single-to-`count` move |
| `drift` | resources deleted or tampered with outside Terraform, then planned |
| `refresh-only` | `plan -refresh-only` (no `resource_changes` at all) |
| `module-provider` | a module handed `providers = { local = local.alt }` |
| `aws-assume-role` | the real AWS provider with `assume_role` (a literal and a variable-based role ARN), an alias without `assume_role`, an alias pinned with `allowed_account_ids`, a module handed an aliased provider, `aws_caller_identity` in `prior_state`, and a genuinely cross-account ARN. Runs against [floci](https://floci.io/floci/), a local AWS emulator, so it needs no AWS credentials. |

Most scenarios use only the built-in `terraform_data` resource; `drift`, `refresh-only` and `module-provider` also use the credential-free `hashicorp/local` provider. `aws-assume-role` uses the real AWS provider against floci (`docker run -p 4566:4566 floci/floci`); `scripts/smoke-plan.sh` seeds the IAM roles it assumes (`scripts/floci-seed.sh`) and skips the scenario with a message if floci is not running. Floci keeps state, so restart it between local runs of that scenario. To run one locally you need Terraform >= 1.7:

```sh
go build -o tfplanlens ./cmd/tfplanlens
scripts/smoke-plan.sh reasons
scripts/smoke-report.sh reasons testdata/plans/real/reasons-tf*.json
```

The tests in `cmd/tfplanlens/real_plans_test.go` assert on these plans by address, action and count (never byte-exact: every run produces new resource UUIDs).
