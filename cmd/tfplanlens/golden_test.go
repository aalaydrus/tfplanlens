package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/golden")

// Formats covered by golden files (SPEC section 7); html has its own checks.
var goldenFormats = map[string]string{"text": "txt", "markdown": "md", "json": "json"}

type goldenCase struct {
	name string   // golden file stem
	plan string   // fixture name
	args []string // extra report args
}

func goldenCases(t *testing.T) []goldenCase {
	t.Helper()
	plans, err := filepath.Glob("../../testdata/plans/*.json")
	if err != nil || len(plans) == 0 {
		t.Fatalf("no fixtures found: %v", err)
	}
	var cases []goldenCase
	for _, p := range plans {
		n := strings.TrimSuffix(filepath.Base(p), ".json")
		cases = append(cases, goldenCase{name: n, plan: n})
	}
	// The runner fixture also runs with the apply account known, and with a
	// flag overriding a value the plan resolves by itself.
	cases = append(cases,
		goldenCase{"runner-cross-account.apply", "runner-cross-account", []string{"--tfapply-account-id", "999999999999"}},
		goldenCase{"runner-cross-account.override", "runner-cross-account", []string{"--tfapply-account-id", "999999999999", "--account", "prod=444444444444"}},
	)
	return cases
}

// TestGolden runs `report --format <f> --no-color` on every fixture and
// compares stdout with testdata/golden/<name>.<ext>. Regenerate with
// `go test ./cmd/... -run TestGolden -update`, then review the diff.
func TestGolden(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	for _, gc := range goldenCases(t) {
		for format, ext := range goldenFormats {
			t.Run(gc.name+"/"+format, func(t *testing.T) {
				var out, errb bytes.Buffer
				root := newRootCmd(strings.NewReader(""), &out, &errb)
				args := []string{"report", "../../testdata/plans/" + gc.plan + ".json", "--format", format, "--no-color"}
				root.SetArgs(append(args, gc.args...))
				if err := root.Execute(); err != nil {
					t.Fatalf("%v\n%s", err, errb.String())
				}
				golden := filepath.Join("../../testdata/golden", gc.name+"."+ext)
				if *update {
					if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
						t.Fatal(err)
					}
					return
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatalf("missing golden file (run with -update): %v", err)
				}
				if !bytes.Equal(out.Bytes(), want) {
					t.Errorf("output differs from %s (run with -update to accept)", golden)
				}
			})
		}
	}
}

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"version"}, strings.NewReader(""), &out, &out); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out.String()) != version {
		t.Errorf("got %q", out.String())
	}
}

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

const plans = "../../testdata/plans/"

func TestFailOnDelete(t *testing.T) {
	if code, _, _ := runCLI("report", plans+"deletes-and-replaces.json", "--no-color", "--fail-on", "delete"); code != exitFailOn {
		t.Errorf("deletes present: exit %d, want %d", code, exitFailOn)
	}
	if code, _, _ := runCLI("report", plans+"basic-create.json", "--no-color", "--fail-on", "delete"); code != exitOK {
		t.Errorf("no deletes: exit %d, want %d", code, exitOK)
	}
}

func TestFailOnScopedAndDrift(t *testing.T) {
	cases := []struct {
		plan, rule string
		want       int
	}{
		{"deletes-and-replaces", "replace:aws_db_instance", exitFailOn},
		{"deletes-and-replaces", "replace:aws_s3_bucket", exitOK},
		{"drift", "drift", exitFailOn},
		{"basic-create", "drift", exitOK},
		{"basic-create", "delete,replace,drift", exitOK},
	}
	for _, c := range cases {
		if code, _, _ := runCLI("report", plans+c.plan+".json", "--no-color", "--fail-on", c.rule); code != c.want {
			t.Errorf("%s --fail-on %s: exit %d, want %d", c.plan, c.rule, code, c.want)
		}
	}
}

func TestFailOnPrintsReportAndBadRuleIsError(t *testing.T) {
	code, out, _ := runCLI("report", plans+"drift.json", "--no-color", "--fail-on", "replace")
	if code != exitFailOn || !strings.Contains(out, "== Danger ==") {
		t.Errorf("exit %d, report printed: %v", code, strings.Contains(out, "== Danger =="))
	}
	if code, _, _ := runCLI("report", plans+"drift.json", "--fail-on", "explode"); code != exitError {
		t.Errorf("bad rule: exit %d, want %d", code, exitError)
	}
}

func TestIgnoreTypesHidesFromReportNotFailOn(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(cfg, []byte("ignore_types: [aws_db_instance]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCLI("report", plans+"deletes-and-replaces.json", "--no-color", "--config", cfg, "--fail-on", "replace:aws_db_instance")
	if code != exitFailOn {
		t.Errorf("exit %d, want %d", code, exitFailOn)
	}
	if strings.Contains(out, "aws_db_instance") {
		t.Error("ignored type still in report")
	}
}

func TestStdinAndErrors(t *testing.T) {
	data, err := os.ReadFile(plans + "basic-create.json")
	if err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := run([]string{"report", "-", "--no-color"}, bytes.NewReader(data), &out, &errb); code != exitOK {
		t.Errorf("stdin: exit %d: %s", code, errb.String())
	}
	if code := run([]string{"report", "-"}, strings.NewReader("{not json"), &out, &errb); code != exitError {
		t.Errorf("bad json: exit %d", code)
	}
	if code := run([]string{"report", "-"}, strings.NewReader(`{"format_version":"9.0"}`), &out, &errb); code != exitError {
		t.Errorf("unsupported format: exit %d", code)
	}
	if code, _, _ := runCLI("report", "does-not-exist.json"); code != exitError {
		t.Errorf("missing file: exit %d", code)
	}
}

func TestSensitiveValuesNeverPrinted(t *testing.T) {
	for _, f := range []string{"text", "json", "markdown", "html"} {
		_, out, _ := runCLI("report", plans+"sensitive.json", "--no-color", "--format", f)
		for _, secret := range []string{"old-secret-pw", "new-secret-pw", "s3cr3t-value"} {
			if strings.Contains(out, secret) {
				t.Errorf("%s output leaks %q", f, secret)
			}
		}
		if !strings.Contains(out, "sensitive") { // the marker, never the value
			t.Errorf("%s output has no sensitive marker", f)
		}
	}
}

func TestOutFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.json")
	if code, out, _ := runCLI("report", plans+"basic-create.json", "--format", "json", "--out", path); code != exitOK || out != "" {
		t.Fatalf("exit %d, stdout %q", code, out)
	}
	if b, err := os.ReadFile(path); err != nil || !strings.Contains(string(b), `"schema_version": 1`) {
		t.Errorf("out file: %v %q", err, b)
	}
}

// TestLargePlanPerformance generates a 5,000-resource plan (not committed)
// and checks that analyzing and rendering stays under 2 seconds.
func TestLargePlanPerformance(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"format_version":"1.2","terraform_version":"1.9.5","resource_changes":[`)
	for i := 0; i < 5000; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		action := `["update"]`
		if i%10 == 0 {
			action = `["delete","create"]`
		}
		fmt.Fprintf(&sb, `{"address":"module.m%d.aws_instance.i[%d]","module_address":"module.m%d","mode":"managed","type":"aws_instance","name":"i","index":%d,"provider_name":"registry.terraform.io/hashicorp/aws","action_reason":"replace_because_cannot_update","change":{"actions":%s,"before":{"ami":"a","tags":{"n":"%d"}},"after":{"ami":"b","tags":{"n":"%d"}},"after_unknown":{"id":true},"replace_paths":[["ami"]]}}`,
			i%50, i, i%50, i, action, i, i+1)
	}
	sb.WriteString(`]}`)

	for _, format := range []string{"text", "json", "markdown", "html"} {
		start := time.Now()
		var out, errb bytes.Buffer
		code := run([]string{"report", "-", "--no-color", "--format", format}, strings.NewReader(sb.String()), &out, &errb)
		if code != exitOK {
			t.Fatalf("%s: exit %d: %s", format, code, errb.String())
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("%s took %v, want < 2s", format, d)
		}
		if format == "markdown" && out.Len() > 60000 {
			t.Errorf("markdown is %d bytes, want <= 60000", out.Len())
		}
	}
}
