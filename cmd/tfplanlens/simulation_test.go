package main

import (
	"os"
	"strings"
	"testing"
)

// simulation/output is what simulation/demo.sh writes, and the rundown quotes
// it. Fail when the committed files and the tool disagree, so the example
// cannot go stale. Fix: run simulation/demo.sh and commit.
func TestSimulationOutputsAreCurrent(t *testing.T) {
	const dir = "../../simulation/"
	cfg := "--config=" + dir + "tfplanlens.config.yaml"
	plan, revised := dir+"assets/input/plan.json", dir+"assets/input/plan-revised.json"
	t.Setenv(envApplyAccount, "")

	for _, tc := range []struct {
		file string
		args []string
	}{
		{"report.txt", []string{"report", plan, "-f", "text", "--no-color"}},
		{"report.md", []string{"report", plan, "-f", "markdown"}},
		{"report.html", []string{"report", plan, "-f", "html"}},
		{"report.json", []string{"report", plan, "-f", "json"}},
		{"report-revised.txt", []string{"report", revised, "--no-color"}},
		{"diff.txt", []string{"diff", plan, revised}},
		{"diff.md", []string{"diff", plan, revised, "-f", "markdown"}},
	} {
		code, out, errs := runCLI(append(tc.args, cfg)...)
		if code != exitOK {
			t.Fatalf("%s: exit %d: %s", tc.file, code, errs)
		}
		want, err := os.ReadFile(dir + "output/" + tc.file)
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != out {
			t.Errorf("simulation/output/%s is out of date: run simulation/demo.sh", tc.file)
		}
	}
}

// The facts the rundown tells the reader to notice.
func TestSimulationTellsTheStoryInTheRundown(t *testing.T) {
	const dir = "../../simulation/"
	cfg := "--config=" + dir + "tfplanlens.config.yaml"
	t.Setenv(envApplyAccount, "")

	_, out, _ := runCLI("report", dir+"assets/input/plan.json", "--no-color", cfg)
	for _, want := range []string{
		"REPLACE aws_dynamodb_table.orders [STATEFUL]",
		"REPLACE aws_secretsmanager_secret_version.db",
		"secret_string: (sensitive value changed)  [forces replacement]",
		`DELETE aws_sqs_queue.orders["bulk"]`,
		"target account: 123456789012 (source: allowed_account_ids)",
		"Executed from (apply account): 999999999999 (source: config)",
		"apply wave 2:",
		"aws_sqs_queue.orders [for_each]: audit (create), bulk (delete), priority (no-op), standard (update)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report is missing %q", want)
		}
	}

	// The gate blocks the proposal and lets the revision through.
	if code, _, errs := runCLI("report", dir+"assets/input/plan.json", "--fail-on", "delete,replace:aws_dynamodb_table", cfg); code != exitFailOn || !strings.Contains(errs, "aws_dynamodb_table.orders") {
		t.Errorf("proposed plan: exit %d %s", code, errs)
	}
	if code, _, _ := runCLI("report", dir+"assets/input/plan-revised.json", "--fail-on", "delete,replace:aws_dynamodb_table", cfg); code != exitOK {
		t.Errorf("revised plan: exit %d", code)
	}

	_, d, _ := runCLI("diff", dir+"assets/input/plan.json", dir+"assets/input/plan-revised.json", cfg)
	if !strings.Contains(d, "== No longer deleted or replaced ==\naws_dynamodb_table.orders\n") || !strings.Contains(d, "~ replace -> update  aws_dynamodb_table.orders") {
		t.Errorf("diff:\n%s", d)
	}
}

// plan.json and the state hold the (fake) passwords in plain text, as real
// plan JSON does. No output may repeat them.
func TestSimulationNeverPrintsSecretValues(t *testing.T) {
	const dir = "../../simulation/"
	cfg := "--config=" + dir + "tfplanlens.config.yaml"
	plan, revised := dir+"assets/input/plan.json", dir+"assets/input/plan-revised.json"
	t.Setenv(envApplyAccount, "")

	raw, _ := os.ReadFile(plan)
	for _, secret := range []string{"initial-password-v1", "rotated-password-v2"} {
		if !strings.Contains(string(raw), secret) {
			t.Fatalf("test is vacuous: plan.json no longer contains %q", secret)
		}
	}
	for _, args := range [][]string{
		{"report", plan, "-f", "text"}, {"report", plan, "-f", "markdown"}, {"report", plan, "-f", "html"}, {"report", plan, "-f", "json"},
		{"report", revised, "-f", "json"}, {"diff", plan, revised}, {"diff", plan, revised, "-f", "markdown"}, {"diff", plan, revised, "-f", "json"},
	} {
		code, out, errs := runCLI(append(args, cfg)...)
		if code != exitOK {
			t.Fatalf("%v: exit %d %s", args, code, errs)
		}
		for _, secret := range []string{"initial-password-v1", "rotated-password-v2", "orders_app"} {
			if strings.Contains(out, secret) {
				t.Errorf("%v prints %q", args, secret)
			}
		}
	}
}
