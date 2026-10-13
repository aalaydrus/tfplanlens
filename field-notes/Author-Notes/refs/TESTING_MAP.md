# Testing map: what guards what

Run all: `go test ./... -count=1` (about 3 s). One package: `go test ./internal/analyze`.
One test: `-run TestName -v`.

## Three kinds of input

| Kind | Where | Used for | Assert how |
|---|---|---|---|
| Hand-written fixtures | `testdata/plans/*.json` | exact output of tricky shapes | **goldens** (`testdata/golden/`), byte for byte |
| Real Terraform plans | `testdata/plans/real/*-tf*.json` (made by `scripts/smoke-plan.sh`) | does the tool match real Terraform | addresses, actions, counts only (ids change per run) |
| Simulation plans | `simulation/assets/input/` | the AWS demo | committed outputs must match (`TestSimulationOutputsAreCurrent`) |

Golden update: `go test ./cmd/... -run TestGolden -update`, then
**read `git diff testdata`**: the diff must be exactly the change you meant.
Unit tests that build `plan.Plan`/`Analysis` by hand are in each package.

## By area

| Area | Tests (file) |
|---|---|
| Every plan x every format works | `TestEveryPlanEveryFormat`, `TestGolden` (`cmd/tfplanlens/golden_test.go`) |
| Danger first, reasons, long values | `TestDangerBeforeAnyChangeListing`, `TestReasonText`, `TestDiffChangedLinesPlusContext` |
| Secret-looking values redacted | `TestUnmarkedSecretsNeverReachAnyOutput`, `TestRedactionIsCountedAndExplained`, `TestRedactionCanBeSwitchedOffAndDecodingToo`, `TestRedactionLeavesTheRealPlansAlone` (`cmd/tfplanlens/redact_test.go`), unit tests in `internal/redact`; offline/no-writes: `scripts/verify-offline.sh` (manual, Linux) |
| Sensitive never printed | `TestSensitiveValuesNeverPrinted`, `TestSensitiveValuesNeverAppear` (html), `TestSensitiveNeverPrinted` (markdown), `TestSimulationNeverPrintsSecretValues` |
| JSON shape | `TestJSONArraysAreNeverNull` (`real_plans_test.go`; add new array keys to `arrayKeys`) |
| Accounts | `apply_account_test.go`, `internal/analyze/analyze_test.go`, `internal/plan/provider_test.go` |
| `--fail-on` | `TestFailOnDelete`, `TestFailOnScopedAndDrift`, `TestFailOnUnresolved` |
| CLI surface | `TestCLISurfaceIsLean` (allowed `report` flags), `TestShortFlags`, `TestConfigFileSettingsAndFlagOverrides` |
| Waves, cycles, instances | `internal/analyze/waves_test.go`, `internal/plan/deps_test.go`, `TestRealWavesFromConfiguration` |
| `-f brief` and `--top` | `internal/analyze/brief_test.go`, `TestBriefFormatListsRankedItemsAndHonoursTop`, `TestTopOnlyAppliesToBriefAndMustBeSane`, `TestBriefStillHonoursFailOn`, the `*.brief.txt` goldens, `simulation/output/brief.txt` |
| Safeguards (deletion protection, final snapshot) | `internal/analyze/safeguards_test.go` (`TestRealPlanDynamoDBTableReplaceShowsProtectionOff`, `TestSafeguardsFromBeforeValues`, `TestUnprotectedDeletesRankBeforeProtectedOnes`) |
| Dependents and Danger ranking | `internal/analyze/dependents_test.go` (`TestFindingsCarryWhatDependsOnThem`, `TestDangerIsRankedByStatefulThenDependents`) |
| diff | `internal/diff/diff_test.go`, `TestRealPlanDiff` |
| Merge (several plans) | `internal/analyze/merge_test.go`, `cmd/tfplanlens/multi_test.go` |
| TUI | `internal/tui/tui_test.go` (model, key parser), `cmd/tfplanlens/tui_test.go` |
| HTML is offline and escaped | `internal/render/html/html_test.go` (`TestEveryFixtureRendersValidSelfContainedHTML`, `TestHostilePlanDataIsEscaped`) |
| Markdown cap and escaping | `internal/render/markdown/markdown_test.go` |
| Big plans | `TestLargePlanPerformance` (generates 5,000 resources) |
| Dependency policy | `TestNoBubbleTeaDependency` |
| These agent docs | `TestAgentDocs*` (`cmd/tfplanlens/agentdocs_test.go`) |

## CI

`ci.yml`: vet + test on Linux/macOS/Windows (Windows is why `.gitattributes`
pins LF). `smoke.yml`: regenerates every real plan with real Terraform, runs the
real-plan tests on them, renders every format, screenshots, comments on the PR and
uploads the outputs (it commits nothing; `scripts/smoke-refresh.sh` refreshes the
committed copies).
