package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/config"
)

const runnerPlan = "../../testdata/plans/runner-cross-account.json"

func TestApplyAccountPrecedence(t *testing.T) {
	cfg := &config.Config{TFApplyAccountID: "333333333333"}
	var warn bytes.Buffer

	t.Setenv(envApplyAccount, "222222222222")
	id, src, err := resolveApplyAccount(&reportOptions{applyAccount: "111111111111"}, cfg, terminal{}, &warn)
	if err != nil || id != "111111111111" || src != analyze.SourceFlag {
		t.Errorf("flag should win: %q %q %v", id, src, err)
	}
	id, src, err = resolveApplyAccount(&reportOptions{}, cfg, terminal{}, &warn)
	if err != nil || id != "222222222222" || src != analyze.SourceEnv {
		t.Errorf("env should beat config: %q %q %v", id, src, err)
	}
	t.Setenv(envApplyAccount, "")
	id, src, err = resolveApplyAccount(&reportOptions{}, cfg, terminal{}, &warn)
	if err != nil || id != "333333333333" || src != analyze.SourceConfig {
		t.Errorf("config: %q %q %v", id, src, err)
	}
	if warn.Len() != 0 {
		t.Errorf("unexpected warning %q", warn.String())
	}
}

func TestApplyAccountInvalidValuesAreErrors(t *testing.T) {
	var warn bytes.Buffer
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
			if _, _, err := resolveApplyAccount(tc.o, tc.cfg, terminal{}, &warn); err == nil {
				t.Error("want error")
			}
		})
	}
}

func tty(input string) (terminal, *bytes.Buffer) {
	var prompt bytes.Buffer
	return terminal{in: strings.NewReader(input), prompt: &prompt, stdinTTY: true, stdoutTTY: true}, &prompt
}

func TestApplyAccountPrompt(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	path := filepath.Join(t.TempDir(), ".tfplanlens.yaml")
	if err := os.WriteFile(path, []byte("# my config\naccounts:\n  prod: \"111111111111\""), 0o644); err != nil {
		t.Fatal(err)
	}
	term, prompt := tty("not-an-id\n123456789012\ny\n")
	id, src, err := resolveApplyAccount(&reportOptions{config: path}, &config.Config{}, term, &bytes.Buffer{})
	if err != nil || id != "123456789012" || src != analyze.SourcePrompt {
		t.Fatalf("%q %q %v", id, src, err)
	}
	if !strings.Contains(prompt.String(), "Which account ID does terraform apply run from?") || !strings.Contains(prompt.String(), "not a 12-digit") {
		t.Errorf("prompt text: %q", prompt.String())
	}
	saved, _ := os.ReadFile(path)
	if !strings.Contains(string(saved), "# my config") || !strings.Contains(string(saved), "accounts:") {
		t.Errorf("existing config lost: %q", saved)
	}
	cfg, err := config.Load(path)
	if err != nil || cfg.TFApplyAccountID != "123456789012" || cfg.Accounts["prod"] != "111111111111" {
		t.Errorf("saved config does not load back: %+v %v", cfg, err)
	}

	// Declining to save leaves the file alone.
	path2 := filepath.Join(t.TempDir(), "c.yaml")
	term, _ = tty("123456789012\nn\n")
	if id, _, _ := resolveApplyAccount(&reportOptions{config: path2}, &config.Config{}, term, &bytes.Buffer{}); id != "123456789012" {
		t.Errorf("id %q", id)
	}
	if _, err := os.Stat(path2); err == nil {
		t.Error("config written without consent")
	}
}

func TestApplyAccountNeverPromptsWhenNotInteractive(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	cases := map[string]struct {
		term terminal
		o    reportOptions
	}{
		"stdin not a TTY": {terminal{stdinTTY: false, stdoutTTY: true}, reportOptions{}},
		"stdout piped":    {terminal{stdinTTY: true, stdoutTTY: false}, reportOptions{}},
		"--no-prompt":     {terminal{stdinTTY: true, stdoutTTY: true}, reportOptions{noPrompt: true}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var prompt, warn bytes.Buffer
			c.term.in = strings.NewReader("123456789012\ny\n")
			c.term.prompt = &prompt
			id, src, err := resolveApplyAccount(&c.o, &config.Config{}, c.term, &warn)
			if err != nil || id != "" || src != analyze.SourceUnresolved {
				t.Errorf("%q %q %v", id, src, err)
			}
			if prompt.Len() != 0 {
				t.Errorf("prompted: %q", prompt.String())
			}
			if !strings.Contains(warn.String(), "apply account is unresolved") {
				t.Errorf("no warning: %q", warn.String())
			}
		})
	}
}

func TestPromptSkippedWhenEmptyAnswer(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	term, _ := tty("\n")
	var warn bytes.Buffer
	id, src, err := resolveApplyAccount(&reportOptions{}, &config.Config{}, term, &warn)
	if err != nil || id != "" || src != analyze.SourceUnresolved || warn.Len() == 0 {
		t.Errorf("%q %q %v %q", id, src, err, warn.String())
	}
}

func TestRequireApplyAccount(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	code, out, errs := runCLI("report", runnerPlan, "--no-color", "--require-apply-account")
	if code != exitError || out != "" || !strings.Contains(errs, "--require-apply-account") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if code, _, errs := runCLI("report", runnerPlan, "--no-color", "--require-apply-account", "--tfapply-account-id", "999999999999"); code != exitOK {
		t.Errorf("resolved: exit %d: %s", code, errs)
	}
	t.Setenv(envApplyAccount, "999999999999")
	if code, _, errs := runCLI("report", runnerPlan, "--no-color", "--require-apply-account"); code != exitOK {
		t.Errorf("env: exit %d: %s", code, errs)
	}
}

func TestUnresolvedApplyAccountWarnsAndContinues(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	code, out, errs := runCLI("report", runnerPlan, "--no-color")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(errs, "warning: apply account is unresolved") {
		t.Errorf("stderr %q", errs)
	}
	if !strings.Contains(out, "Executed from (apply account): unresolved") {
		t.Errorf("report does not say the apply account is unresolved:\n%s", out)
	}
}

func TestConfigApplyAccountAndAccountFlag(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	cfg := filepath.Join(t.TempDir(), "c.yaml")
	// Unquoted on purpose: YAML reads these as integers.
	if err := os.WriteFile(cfg, []byte("tfapply_account_id: 999999999999\naccounts:\n  sandbox: 555555555555\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCLI("report", runnerPlan, "--no-color", "--config", cfg, "--account", "staging=777777777777")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	for _, want := range []string{
		"Executed from (apply account): 999999999999 (source: config)",
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
