package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/config"
)

const runnerPlan = "../../testdata/plans/runner-cross-account.json"

func TestApplyAccountPrecedence(t *testing.T) {
	cfg := &config.Config{TFApplyAccountID: "333333333333"}

	t.Setenv(envApplyAccount, "222222222222")
	id, src, err := explicitApplyAccount(&reportOptions{applyAccount: "111111111111"}, cfg)
	if err != nil || id != "111111111111" || src != analyze.SourceFlag {
		t.Errorf("flag should win: %q %q %v", id, src, err)
	}
	id, src, err = explicitApplyAccount(&reportOptions{}, cfg)
	if err != nil || id != "222222222222" || src != analyze.SourceEnv {
		t.Errorf("env should beat config: %q %q %v", id, src, err)
	}
	t.Setenv(envApplyAccount, "")
	id, src, err = explicitApplyAccount(&reportOptions{}, cfg)
	if err != nil || id != "333333333333" || src != analyze.SourceConfig {
		t.Errorf("config: %q %q %v", id, src, err)
	}
	// Nothing given: left for the analysis to infer from the plan.
	if id, src, err = explicitApplyAccount(&reportOptions{}, &config.Config{}); err != nil || id != "" || src != "" {
		t.Errorf("none given: %q %q %v", id, src, err)
	}
}

func TestApplyAccountInvalidValuesAreErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		o   *reportOptions
		env string
		cfg *config.Config
	}{
		"flag":   {&reportOptions{applyAccount: "1234"}, "", &config.Config{}},
		"env":    {&reportOptions{}, "abc", &config.Config{}},
		"config": {&reportOptions{}, "", &config.Config{TFApplyAccountID: "12345678901x"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(envApplyAccount, tc.env)
			if _, _, err := explicitApplyAccount(tc.o, tc.cfg); err == nil {
				t.Error("want error")
			}
		})
	}
}

// With no flag, env var or config, the apply account is the account of the
// plan's default provider (here from an aws_caller_identity in prior_state),
// and nothing is asked or warned about.
func TestApplyAccountInferredFromDefaultProvider(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	code, out, errs := runCLI("report", runnerPlan, "--no-color")
	if code != exitOK || errs != "" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	for _, want := range []string{
		"Executed from (apply account): 999999999999 (source: inferred from the default provider via caller_identity)",
		"aws.sandbox: target account 999999999999 (source: apply_account)", // no assume_role: runs as the apply account
		"aws.prod: target account 111111111111 (source: role_arn) [DIFFERENT from apply account]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "NOTICE") {
		t.Error("a resolved apply account needs no notice")
	}

	// An explicit value always beats the inferred one.
	_, out, _ = runCLI("report", runnerPlan, "--no-color", "--tfapply-account-id", "444444444444")
	if !strings.Contains(out, "Executed from (apply account): 444444444444 (source: flag)") {
		t.Errorf("flag did not override the inferred account:\n%s", out)
	}
}

// When the plan says nothing about the runner, the report says so in one
// notice. No prompt, no stderr noise, exit 0.
func TestApplyAccountUnresolvedIsQuiet(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	code, out, errs := runCLI("report", plans+"basic-create.json", "--no-color")
	if code != exitOK || errs != "" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	if !strings.Contains(out, "Executed from (apply account): unresolved") || !strings.Contains(out, "set tfapply_account_id in tfplanlens.config.yaml") {
		t.Errorf("unresolved apply account not explained:\n%s", out)
	}
}

func TestFailOnUnresolved(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	if code, _, errs := runCLI("report", plans+"basic-create.json", "--fail-on", "unresolved"); code != exitFailOn || !strings.Contains(errs, "unresolved matched apply account") {
		t.Errorf("unresolved accounts: exit %d, stderr %q", code, errs)
	}
	// Everything resolvable from the plan itself: nothing to fail on.
	if code, _, errs := runCLI("report", runnerPlan, "--fail-on", "unresolved"); code != exitOK {
		t.Errorf("resolved accounts: exit %d, stderr %q", code, errs)
	}
	if _, _, errs := runCLI("report", runnerPlan, "--fail-on", "unresolved:aws_x"); !strings.Contains(errs, "takes no type") {
		t.Errorf("stderr %q", errs)
	}
}

func TestConfigApplyAccountAndAccountFlag(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	cfg := filepath.Join(t.TempDir(), "c.yaml")
	// Unquoted on purpose: YAML reads these as integers.
	if err := os.WriteFile(cfg, []byte("tfapply_account_id: 888888888888\naccounts:\n  sandbox: 555555555555\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCLI("report", runnerPlan, "--no-color", "-c", cfg, "--account", "staging=777777777777")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	for _, want := range []string{
		"Executed from (apply account): 888888888888 (source: config)",
		"aws.sandbox: target account 555555555555 (source: config)",
		"aws.staging: target account 777777777777 (source: flag)",
		"aws.prod: target account 111111111111 (source: role_arn)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestBadAccountFlag(t *testing.T) {
	for _, v := range []string{"prod", "=111111111111", "prod=12345"} {
		if code, _, _ := runCLI("report", runnerPlan, "--account", v); code != exitError {
			t.Errorf("--account %q: exit %d, want %d", v, code, exitError)
		}
	}
}

func writeConfig(t *testing.T, yaml string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "tfplanlens.config.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// The config file can hold the run's settings; flags override it.
func TestConfigFileSettingsAndFlagOverrides(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	deletes := plans + "deletes-and-replaces.json"

	cfg := writeConfig(t, "format: [json]\nfail_on: [delete]\nmax_line: 40\n")
	code, out, _ := runCLI("report", deletes, "-c", cfg)
	if code != exitFailOn {
		t.Errorf("fail_on from config: exit %d", code)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("format from config: output starts %.30q", out)
	}

	// Flags win: text instead of json, and a different --fail-on rule.
	code, out, _ = runCLI("report", deletes, "-c", cfg, "-f", "text", "--no-color", "--fail-on", "drift")
	if code != exitOK || !strings.Contains(out, "== Danger ==") {
		t.Errorf("flag overrides: exit %d, output %.40q", code, out)
	}

	// max_line truncates terminal output.
	_, out, _ = runCLI("report", deletes, "-c", writeConfig(t, "max_line: 40\n"), "--no-color")
	for _, line := range strings.Split(out, "\n") {
		if n := utf8.RuneCountInString(line); n > 40 {
			t.Errorf("%d-rune line with max_line 40: %q", n, line)
			break
		}
	}
}

func TestBadConfigValuesAreErrors(t *testing.T) {
	for name, yaml := range map[string]string{
		"unknown key": "tfapply_account: 123456789012\n",
		"bad format":  "format: [pdf]\n",
		"negative":    "max_line: -1\n",
		"bad fail_on": "fail_on: [explode]\n",
	} {
		if code, _, errs := runCLI("report", plans+"basic-create.json", "-c", writeConfig(t, yaml)); code != exitError {
			t.Errorf("%s: exit %d (%s)", name, code, errs)
		}
	}
}

func TestShortFlags(t *testing.T) {
	out := filepath.Join(t.TempDir(), "r.json")
	if code, stdout, errs := runCLI("report", plans+"basic-create.json", "-f", "json", "-o", out); code != exitOK || stdout != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, errs)
	}
	if b, err := os.ReadFile(out); err != nil || !strings.Contains(string(b), `"schema_version": 1`) {
		t.Errorf("%v %q", err, b)
	}
}

// The CLI stays small: no completion command, and only these flags. (--top
// was added for -f brief; adding a flag means changing this list on purpose.)
func TestCLISurfaceIsLean(t *testing.T) {
	code, out, _ := runCLI("--help")
	if code != exitOK || strings.Contains(out, "completion") {
		t.Errorf("exit %d; completion command present:\n%s", code, out)
	}
	_, out, _ = runCLI("report", "--help")
	want := map[string]bool{"format": true, "out": true, "config": true, "fail-on": true, "tfapply-account-id": true, "account": true, "no-color": true, "top": true, "help": true}
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		name := strings.Fields(strings.ReplaceAll(trimmed, ",", " "))
		var long string
		for _, w := range name {
			if strings.HasPrefix(w, "--") {
				long = strings.TrimPrefix(w, "--")
			}
		}
		if !want[long] {
			t.Errorf("unexpected flag --%s", long)
		}
		delete(want, long)
	}
	if len(want) != 0 {
		t.Errorf("flags missing from --help: %v", want)
	}
}

// Found by the real-plan smoke test: /dev/null is a character device, so a
// character-device check treats it as a terminal.
func TestDevNullIsNotATerminal(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("/dev/null must not count as a terminal")
	}
	if isTerminal(strings.NewReader("")) {
		t.Error("non-files are never terminals")
	}
	if useColor(f, false) {
		t.Error("no colour when output is /dev/null")
	}
}
