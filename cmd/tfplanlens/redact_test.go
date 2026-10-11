package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// secret-lookalikes.json holds secrets Terraform did NOT mark sensitive: a
// secret-named attribute, an env var, credentials in a URL, a private key and
// NAME=value lines inside base64 user_data. None may reach any output.
const lookalikePlan = "../../testdata/plans/secret-lookalikes.json"

var lookalikeSecrets = []string{
	"pw-OLD-unmarked", "pw-NEW-unmarked", "tok-OLD-unmarked", "tok-NEW-unmarked", "hunter-URL-OLD", "hunter-URL-NEW",
	"dbpass-OLD", "dbpass-NEW", "AKIAIOSFODNN7EXAMPLE", "MIIEfakeprivatekeymaterial", "BBBBfakeprivatekeymaterialCCCC",
}

// Things that merely look like they might be secrets and must stay visible.
var lookalikeBenign = []string{
	"Rotates the token every 30 days", "Rotates the token every 60 days", "token_validity_days", "echo deploying v1",
	"echo deploying v2", "LOG_LEVEL", "debug", "secret:app-AbCdEf", "[REDACTED]",
}

func TestUnmarkedSecretsNeverReachAnyOutput(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	for _, f := range []string{"text", "markdown", "html", "json"} {
		code, out, errs := runCLI("report", lookalikePlan, "-f", f, "--no-color", "--tfapply-account-id", "111111111111")
		if code != exitOK {
			t.Fatalf("%s: exit %d %s", f, code, errs)
		}
		for _, s := range lookalikeSecrets {
			if strings.Contains(out, s) {
				t.Errorf("%s output contains %q", f, s)
			}
		}
		if f != "json" { // the JSON escapes quotes, so only check the readable formats for the benign text
			for _, b := range lookalikeBenign {
				if !strings.Contains(out, b) {
					t.Errorf("%s output lost %q", f, b)
				}
			}
		}
	}
	// The HTML report carries the decoded values for its "expand full value"
	// toggle; they must be redacted too.
	_, html, _ := runCLI("report", lookalikePlan, "-f", "html", "--tfapply-account-id", "111111111111")
	if strings.Contains(html, "dbpass-") || strings.Contains(html, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("the HTML expand-full-value data contains a decoded secret")
	}
	// diff and the tui model share the analysis.
	code, d, _ := runCLI("diff", lookalikePlan, lookalikePlan, "-f", "json", "--tfapply-account-id", "111111111111")
	for _, s := range lookalikeSecrets {
		if code != exitOK || strings.Contains(d, s) {
			t.Errorf("diff output (exit %d) contains %q", code, s)
		}
	}
}

func TestRedactionIsCountedAndExplained(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	_, out, _ := runCLI("report", lookalikePlan, "-f", "json", "--tfapply-account-id", "111111111111")
	var a struct {
		Summary struct {
			Redacted int `json:"redacted"`
		}
		Notices []string
	}
	if err := json.Unmarshal([]byte(out), &a); err != nil {
		t.Fatal(err)
	}
	if a.Summary.Redacted == 0 || len(a.Notices) == 0 || !strings.Contains(strings.Join(a.Notices, " "), "redact_secrets: false") {
		t.Errorf("the report must say it hid something and how to show it: %+v", a)
	}
	// A plan with nothing to hide says nothing.
	_, plain, _ := runCLI("report", "../../testdata/plans/basic-create.json", "-f", "json", "--tfapply-account-id", "111111111111")
	if strings.Contains(plain, `"redacted"`) {
		t.Error("redacted must be absent when nothing was hidden")
	}
}

func TestRedactionCanBeSwitchedOffAndDecodingToo(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	dir := t.TempDir()
	off := filepath.Join(dir, "off.yaml")
	if err := os.WriteFile(off, []byte("redact_secrets: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, out, _ := runCLI("report", lookalikePlan, "--no-color", "--config", off, "--tfapply-account-id", "111111111111")
	for _, s := range []string{"dbpass-OLD", "AKIAIOSFODNN7EXAMPLE", "hunter-URL-OLD", "MIIEfakeprivatekeymaterial"} {
		if !strings.Contains(out, s) {
			t.Errorf("with redact_secrets: false, %q should be shown", s)
		}
	}
	// With the name rule off the attribute shows its values again too.
	if !strings.Contains(out, "pw-OLD-unmarked") {
		t.Error("redact_secrets: false should also show secret-named attributes")
	}

	noDecode := filepath.Join(dir, "nodecode.yaml")
	if err := os.WriteFile(noDecode, []byte("decode_values: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, out, _ = runCLI("report", lookalikePlan, "--no-color", "--config", noDecode, "--tfapply-account-id", "111111111111")
	if strings.Contains(out, "echo deploying v1") || strings.Contains(out, "dbpass-") {
		t.Error("decode_values: false must leave base64 undecoded")
	}
	// Redaction still applies to what is shown.
	for _, s := range lookalikeSecrets {
		if strings.Contains(out, s) {
			t.Errorf("decode_values: false leaked %q", s)
		}
	}
}

// Real plans: redaction must not change anything for them (no false positives
// on the plans we ship), unless the plan really holds secret-looking values.
func TestRedactionLeavesTheRealPlansAlone(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	files, _ := filepath.Glob("../../testdata/plans/real/*.json")
	files = append(files, "../../simulation/assets/input/plan.json", "../../simulation/assets/input/plan-revised.json")
	for _, f := range files {
		code, out, errs := runCLI("report", f, "-f", "json", "--tfapply-account-id", "111111111111")
		if code != exitOK {
			t.Fatalf("%s: exit %d %s", f, code, errs)
		}
		if strings.Contains(out, `"redacted"`) {
			t.Errorf("%s: unexpected redaction (false positive?): see summary.redacted", filepath.Base(f))
		}
	}
}
