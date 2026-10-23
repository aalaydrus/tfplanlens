# Development: releases and smoke tests

Other working notes: [`field-notes/`](../field-notes/README.md).

## Releases

A tag `v*` runs `.github/workflows/release.yml`: GoReleaser builds linux, darwin and windows binaries (amd64 and arm64) with the current stable Go, writes `checksums.txt` and creates the GitHub release. The Homebrew formula is generated but only uploaded when `HOMEBREW_TAP_GITHUB_TOKEN` is set and the `aalaydrus/homebrew-tap` repository exists; until then it is skipped. CI runs `goreleaser check` and a snapshot build on every pull request.

To cut a release: merge to `main`, wait for CI, then `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`. Then check the run, the release's six archives and checksums, and that `tfplanlens version` prints `X.Y.Z`. The generated changelog lists commit subjects; the hand-written notes for a release live in `docs/releases/`. The first release is v0.1.0.

## Smoke tests on real plans

`.github/workflows/smoke.yml` runs tfplanlens against real Terraform plans. For each scenario under `testdata/real/`, `scripts/smoke-plan.sh` applies `v1`, swaps in `v2`, and runs `terraform plan` (optionally with extra flags from `plan-args`, and an out-of-band step from `between.sh`). It then renders every format, runs the Go tests against the fresh plans, takes screenshots (`scripts/screenshot.mjs`), posts the reports on the pull request, and uploads everything as the `smoke-outputs` artifact. Nothing is committed back to the branch. To update the committed plans and reports (`testdata/plans/real`, `docs/smoke`, `docs/screenshots`), run `scripts/smoke-refresh.sh` (add `SCREENSHOTS=1` for the screenshots) and commit the result: pushes made with a workflow's own token do not start other workflows, so a bot commit on a pull request would have no CI results and block required checks.

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

## Workflow actions are pinned

Every `uses:` in `.github/workflows` is a full commit SHA with the version in a
comment, because a tag like `@v4` can be moved to different code by whoever owns
that repository. `.github/dependabot.yml` opens a weekly pull request to bump the
pins (and the Go modules); read the upstream diff before merging it. To pin a new
action: `git ls-remote --tags https://github.com/OWNER/REPO 'refs/tags/vX*'`, then
`uses: OWNER/REPO@<40-char sha> # vX.Y.Z`. `TestWorkflowActionsArePinnedToCommitSHAs`
fails on an unpinned one. Still floating on purpose in the smoke test:
`terraform_version: 1.x` and goreleaser `~> v2.9`.

