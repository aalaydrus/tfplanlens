package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/golden")

// Formats covered by golden files (SPEC section 7); html has its own checks.
var goldenFormats = map[string]string{"text": "txt", "markdown": "md", "json": "json", "brief": "brief.txt"}

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
	if got := strings.TrimSpace(out.String()); !strings.HasPrefix(got, version) {
		t.Errorf("got %q", got)
	}
}

func TestFormatVersion(t *testing.T) {
	stamped := func(rev, modified string) *debug.BuildInfo {
		return &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: rev}, {Key: "vcs.modified", Value: modified}}}
	}
	cases := []struct {
		name, v string
		info    *debug.BuildInfo
		want    string
	}{
		{"release build keeps its version", "0.1.0", stamped("8814cf78ccac", "false"), "0.1.0"},
		{"local clean build shows the revision", "dev", stamped("8814cf78ccacf67c4ed02c2b12918c8d066b0d90", "false"), "dev (8814cf7)"},
		{"local dirty build says so", "dev", stamped("8814cf78ccac", "true"), "dev (8814cf7, modified)"},
		{"no VCS info", "dev", &debug.BuildInfo{}, "dev"},
		{"no build info", "dev", nil, "dev"},
	}
	for _, c := range cases {
		if got := formatVersion(c.v, c.info); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// SPEC 9: `report` runs on every fixture, in every format, with no errors.
// This covers the hand-written fixtures and the real Terraform plans.
func TestEveryPlanEveryFormat(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	var plans []string
	for _, pattern := range []string{"../../testdata/plans/*.json", "../../testdata/plans/real/*.json"} {
		m, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		plans = append(plans, m...)
	}
	if len(plans) < 15 {
		t.Fatalf("only %d plans found", len(plans))
	}
	for _, plan := range plans {
		for _, format := range []string{"text", "markdown", "json", "html"} {
			if code, out, errs := runCLI("report", plan, "--format", format, "--no-color"); code != exitOK || out == "" {
				t.Errorf("%s --format %s: exit %d, %d bytes: %s", filepath.Base(plan), format, code, len(out), errs)
			}
		}
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

// SPEC 9, criterion 4: deletes and replaces appear in the Danger section
// before any change listing, in text, markdown and html.
func TestDangerBeforeAnyChangeListing(t *testing.T) {
	t.Setenv(envApplyAccount, "")
	var plansWithDanger []string
	for _, pattern := range []string{"../../testdata/plans/*.json", "../../testdata/plans/real/*.json"} {
		m, _ := filepath.Glob(pattern)
		plansWithDanger = append(plansWithDanger, m...)
	}
	markers := map[string]struct{ danger, listing string }{
		"text":     {"== Danger ==", "== Changes by account =="},
		"markdown": {"### Danger", "### Changes by account"},
		"html":     {`id="danger"`, `id="rows"`},
	}
	checked := 0
	for _, plan := range plansWithDanger {
		_, jsonOut, _ := runCLI("report", plan, "-f", "json")
		if !strings.Contains(jsonOut, `"findings": [`+"\n"+`    {`) {
			continue // no deletes or replaces: nothing to order
		}
		checked++
		for format, m := range markers {
			_, out, _ := runCLI("report", plan, "-f", format, "--no-color")
			d, l := strings.Index(out, m.danger), strings.Index(out, m.listing)
			if d < 0 || l < 0 || d > l {
				t.Errorf("%s --format %s: danger at %d, change listing at %d", filepath.Base(plan), format, d, l)
			}
		}
	}
	if checked < 8 {
		t.Errorf("only %d plans with findings checked", checked)
	}
}

func TestBriefFormatListsRankedItemsAndHonoursTop(t *testing.T) {
	const dir = "../../simulation/"
	plan := dir + "assets/input/plan.json"
	cfg := "--config=" + dir + "tfplanlens.config.yaml"
	t.Setenv(envApplyAccount, "")

	code, out, errs := runCLI("report", plan, "-f", "brief", "--top", "2", cfg)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	for _, want := range []string{"showing 2 of 5 needing attention", "1. REPLACE aws_dynamodb_table.orders [STATEFUL]", `2. DELETE aws_sqs_queue.orders["bulk"]`, "3 more not shown"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "3. ") {
		t.Errorf("--top 2 printed a third item:\n%s", out)
	}
	if _, all, _ := runCLI("report", plan, "-f", "brief", "--top", "0", cfg); !strings.Contains(all, "5. UPDATE aws_secretsmanager_secret.db") {
		t.Errorf("--top 0 must list everything:\n%s", all)
	}
}

func TestTopOnlyAppliesToBriefAndMustBeSane(t *testing.T) {
	plan := plans + "basic-create.json"
	if code, _, errs := runCLI("report", plan, "--top", "3"); code != exitError || !strings.Contains(errs, "--top only applies to -f brief") {
		t.Errorf("--top with the default format: exit %d, %q", code, errs)
	}
	if code, _, errs := runCLI("report", plan, "-f", "brief", "--top", "-1"); code != exitError || !strings.Contains(errs, "--top must be") {
		t.Errorf("--top -1: exit %d, %q", code, errs)
	}
	if code, out, _ := runCLI("report", plan, "-f", "brief"); code != exitOK || !strings.Contains(out, "Nothing needs attention") {
		t.Errorf("a plan with nothing to flag: exit %d\n%s", code, out)
	}
}

func TestBriefStillHonoursFailOn(t *testing.T) {
	code, out, _ := runCLI("report", plans+"deletes-and-replaces.json", "-f", "brief", "--fail-on", "delete")
	if code != exitFailOn || !strings.Contains(out, "tfplanlens brief") {
		t.Errorf("exit %d, brief written: %v", code, strings.Contains(out, "tfplanlens brief"))
	}
}
