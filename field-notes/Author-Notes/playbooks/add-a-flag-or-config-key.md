# Playbook: add a CLI flag, command or config key

The CLI is deliberately small. Prefer a config key over a flag, and a default
over either. A new `report` flag also needs `TestCLISurfaceIsLean` updated
(the `want` map in `cmd/tfplanlens/apply_account_test.go`), which is the
point: it forces the conversation.

**Flag**: `cmd/tfplanlens/main.go` (`newReportCmd`, `newDiffCmd`, `newTUICmd`
register flags on `reportOptions`; `runReport` / `loadAnalysis` consume them).
Precedence is flag > env > config > default; follow `explicitApplyAccount`
(`apply_account.go`) as the pattern.

**Config key**: `internal/config/config.go`: add the field with a `yaml` tag
and validation in `validate()` (unknown keys are errors on purpose); example
and comment in `tfplanlens.config.yaml`; `docs/USAGE.md` ("Config file", flags) and the README command list if it is a headline feature;
`config_test.go` + a precedence test next to `TestConfigFileSettingsAndFlagOverrides`.

**Command**: `newRootCmd` registers commands. Reuse `loadAll`
(`multi.go`) so several plans, labels and `--config` work for free. Errors are
returned (exit 2); a rule match returns `*failOnError` (exit 1) after the
output is written.

Always: help text with an example, a test that drives it through `runCLI(...)`
(`cmd/tfplanlens/golden_test.go` has the helper), `docs/USAGE.md`, and
[../refs/CLI_REFERENCE.md](../refs/CLI_REFERENCE.md).
