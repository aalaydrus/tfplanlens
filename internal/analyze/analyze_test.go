package analyze

import (
	"os"
	"strings"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/config"
	"github.com/aalaydrus/tfplanlens/internal/plan"
)

func TestParseARNAccount(t *testing.T) {
	cases := map[string]string{
		"arn:aws:iam::111111111111:role/deployer": "111111111111",
		"arn:aws-cn:iam::222222222222:role/a:b":   "222222222222",
		"arn:aws:s3:::bucket":                     "",
		"arn:aws:iam::12345:role/x":               "",
		"arn:aws:iam::11111111111a:role/x":        "",
		"not-an-arn":                              "",
		"":                                        "",
	}
	for in, want := range cases {
		if got := ParseARNAccount(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestReasonText(t *testing.T) {
	cases := []struct {
		reason, index string
		hasIdx        bool
		paths         []string
		want          string
	}{
		{"delete_because_no_resource_config", "", false, nil, "Removed from config"},
		{"delete_because_wrong_repetition", "", false, nil, "Resource switched between count/for_each/single"},
		{"delete_because_count_index", "3", true, nil, "count shrank; index 3 no longer exists"},
		{"delete_because_each_key", "bob", true, nil, "for_each key removed: bob"},
		{"delete_because_no_module", "", false, nil, "Whole module removed or its call changed"},
		{"replace_because_cannot_update", "", false, []string{"a", "b.c"}, "Attribute change forces replacement: a, b.c"},
		{"replace_because_tainted", "", false, nil, "Resource is tainted"},
		{"replace_by_request", "", false, nil, "Replacement requested with -replace"},
		{"brand_new_reason", "", false, nil, "brand_new_reason"},
		{"", "", false, nil, "(no reason reported)"},
	}
	for _, c := range cases {
		if got := ReasonText(c.reason, c.index, c.hasIdx, c.paths); got != c.want {
			t.Errorf("%q: got %q, want %q", c.reason, got, c.want)
		}
	}
}

func load(t *testing.T, name string) *plan.Plan {
	t.Helper()
	f, err := os.Open("../../testdata/plans/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := plan.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEveryReasonRenderedInDangerSection(t *testing.T) {
	a := Analyze(load(t, "deletes-and-replaces"), nil)
	if len(a.Findings) != 12 {
		t.Fatalf("%d findings, want 12", len(a.Findings))
	}
	for _, f := range a.Findings {
		if f.ReasonText == "" {
			t.Errorf("%s: no reason text", f.Address)
		}
	}
	// replace_paths come through the raw-JSON fallback.
	for _, f := range a.Findings {
		if f.Address == "aws_db_instance.main" {
			if strings.Join(f.ReplacePaths, ",") != "storage_encrypted" || !f.Stateful {
				t.Errorf("db finding: %+v", f)
			}
		}
	}
}

func TestIndexShiftDetection(t *testing.T) {
	a := Analyze(load(t, "for-each-shift"), nil)
	got := map[string]bool{}
	for _, f := range a.Findings {
		got[f.Address] = f.IndexShift
	}
	want := map[string]bool{
		"aws_instance.fleet[2]":      true,
		`aws_iam_user.team["carol"]`: false,
		"aws_s3_bucket.shards[1]":    false,
	}
	for addr, w := range want {
		if g, ok := got[addr]; !ok || g != w {
			t.Errorf("%s: shift=%v (present %v), want %v", addr, g, ok, w)
		}
	}
}

func TestAccountResolutionOrder(t *testing.T) {
	p := load(t, "multi-account")
	a := Analyze(p, &config.Config{Accounts: map[string]string{
		"mystery": "333333333333",
		"prod":    "999999999999", // loses to the literal role ARN
	}})
	got := map[string]Group{}
	for _, g := range a.Groups {
		got[g.Key] = g
	}
	for key, want := range map[string]string{
		"aws.prod":    "111111111111",
		"aws.shared":  "222222222222",
		"aws.mystery": "333333333333",
		"aws":         "unresolved (alias: default)",
	} {
		if got[key].Account != want {
			t.Errorf("%s: %q, want %q", key, got[key].Account, want)
		}
	}
	if got["aws.prod"].Source != "role_arn" || got["aws.mystery"].Source != "config" {
		t.Errorf("sources: %q %q", got["aws.prod"].Source, got["aws.mystery"].Source)
	}
	if !got["aws.prod"].Cross || got["aws"].Cross {
		t.Error("cross-account flag wrong")
	}

	a = Analyze(p, nil)
	for _, g := range a.Groups {
		if g.Key == "aws.mystery" && g.Account != "unresolved (alias: mystery)" {
			t.Errorf("fallback label %q", g.Account)
		}
	}
}

func TestDriftHighRiskAndKnownAfterApply(t *testing.T) {
	a := Analyze(load(t, "drift"), nil)
	risk := map[string]bool{}
	for _, d := range a.Drift {
		risk[d.Address] = d.HighRisk
	}
	if !risk["aws_security_group.web"] || risk["aws_instance.api"] {
		t.Errorf("risk %v", risk)
	}
	if !a.Findings[0].DriftAlso {
		t.Error("finding not flagged as drifted")
	}
	if got := Analyze(load(t, "basic-create"), nil).Summary.KnownAfterApply; got != 4 {
		t.Errorf("known after apply = %d, want 4", got)
	}
}

func TestMovedImportForget(t *testing.T) {
	a := Analyze(load(t, "moved-import-removed"), nil)
	by := map[string]Change{}
	for _, c := range a.Changes {
		by[c.Address] = c
	}
	if by["aws_s3_bucket.new_name"].MovedFrom != "aws_s3_bucket.old_name" || by["aws_s3_bucket.new_name"].Action != "no-op" {
		t.Errorf("moved: %+v", by["aws_s3_bucket.new_name"])
	}
	if !by["aws_iam_role.imported"].Import || !by["aws_db_instance.forgotten"].StateOnly {
		t.Error("import / state-only flags missing")
	}
	if len(a.Findings) != 0 {
		t.Error("moves, imports and forgets are not danger findings")
	}
}

func TestIgnoreTypesAndStatefulConfig(t *testing.T) {
	cfg := &config.Config{IgnoreTypes: []string{"aws_sqs_queue"}, StatefulTypes: []string{"aws_sns_topic"}}
	a := Analyze(load(t, "deletes-and-replaces"), cfg)
	if a.Summary.Hidden != 1 || len(a.Findings) != 11 {
		t.Errorf("hidden %d findings %d", a.Summary.Hidden, len(a.Findings))
	}
	for _, f := range a.Findings {
		if f.Type == "aws_sns_topic" && !f.Stateful {
			t.Error("config stateful_types ignored")
		}
	}
}

func TestParseRules(t *testing.T) {
	r, err := ParseRules("delete, replace:aws_db_instance,drift")
	if err != nil || len(r) != 3 || r[1].Type != "aws_db_instance" {
		t.Fatalf("%v %v", r, err)
	}
	for _, bad := range []string{"nope", "delete:", "replace:x,boom"} {
		if _, err := ParseRules(bad); err == nil {
			t.Errorf("%q should fail", bad)
		}
	}
}
