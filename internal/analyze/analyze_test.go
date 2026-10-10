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
	a := Analyze(load(t, "deletes-and-replaces"), nil, nil)
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
	a := Analyze(load(t, "for-each-shift"), nil, nil)
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

func groupsByKey(a *Analysis) map[string]Group {
	m := map[string]Group{}
	for _, g := range a.Groups {
		m[g.Key] = g
	}
	return m
}

func TestTargetAccountSources(t *testing.T) {
	p := load(t, "runner-cross-account")
	a := Analyze(p, nil, &Options{ApplyAccount: "999999999999", ApplySource: SourceFlag})
	g := groupsByKey(a)
	for key, want := range map[string][2]string{
		"aws":         {"999999999999", SourceCallerIdentity},  // prior_state aws_caller_identity
		"aws.prod":    {"111111111111", SourceRoleARN},         // literal assume_role.role_arn
		"aws.staging": {"222222222222", SourceRoleARNVariable}, // resolved through plan variables
		"aws.sandbox": {"999999999999", SourceApplyAccount},    // no assume_role: runs as the apply account
	} {
		if g[key].Account != want[0] || g[key].Source != want[1] {
			t.Errorf("%s: %s via %s, want %s via %s", key, g[key].Account, g[key].Source, want[0], want[1])
		}
	}
	if g["aws"].Cross || g["aws.sandbox"].Cross || !g["aws.prod"].Cross || !g["aws.staging"].Cross {
		t.Errorf("cross flags: %v %v %v %v", g["aws"].Cross, g["aws.sandbox"].Cross, g["aws.prod"].Cross, g["aws.staging"].Cross)
	}
	if a.ApplyAccount.ID != "999999999999" || a.ApplyAccount.Source != SourceFlag || len(a.Notices) != 0 {
		t.Errorf("apply account %+v notices %v", a.ApplyAccount, a.Notices)
	}
}

func TestUnresolvedApplyAccount(t *testing.T) {
	a := Analyze(load(t, "runner-cross-account"), nil, nil)
	g := groupsByKey(a)
	// Without an apply account the no-assume_role alias cannot be resolved,
	// and nothing is flagged as cross-account.
	if g["aws.sandbox"].Resolved || g["aws.sandbox"].Source != SourceUnresolved || g["aws.sandbox"].Account != "unresolved (alias: sandbox)" {
		t.Errorf("sandbox: %+v", g["aws.sandbox"])
	}
	for k, grp := range g {
		if grp.Cross {
			t.Errorf("%s flagged cross-account with unknown apply account", k)
		}
	}
	if a.ApplyAccount.Resolved || a.ApplyAccount.Source != SourceUnresolved || len(a.Notices) != 1 {
		t.Errorf("apply account %+v notices %v", a.ApplyAccount, a.Notices)
	}
}

// Precedence: --account flag > config accounts > role_arn > allowed_account_ids
// > caller identity > apply account > unresolved.
func TestTargetAccountPrecedence(t *testing.T) {
	p := &plan.Plan{Providers: map[string]plan.Provider{
		"aws.x": {
			Key: "aws.x", Name: "aws", Alias: "x", AssumeRole: true,
			AssumeRoleARN: "arn:aws:iam::333333333333:role/r", RoleARNVia: plan.ViaLiteral,
			AllowedAccountIDs: []string{"444444444444"},
			CallerIdentityIDs: []string{"555555555555"},
		},
	}}
	cfg := &config.Config{Accounts: map[string]string{"x": "222222222222"}}
	steps := []struct {
		name string
		cfg  *config.Config
		opts *Options
		want string
		src  string
	}{
		{"flag", cfg, &Options{AccountOverrides: map[string]string{"x": "111111111111"}}, "111111111111", SourceFlag},
		{"flag by full key", cfg, &Options{AccountOverrides: map[string]string{"aws.x": "111111111111"}}, "111111111111", SourceFlag},
		{"config", cfg, &Options{}, "222222222222", SourceConfig},
		{"role_arn", &config.Config{}, &Options{}, "333333333333", SourceRoleARN},
	}
	for _, s := range steps {
		got := resolveAccount(p, s.cfg, s.opts, "aws.x")
		if got.label != s.want || got.source != s.src {
			t.Errorf("%s: %s via %s, want %s via %s", s.name, got.label, got.source, s.want, s.src)
		}
	}

	// Peel layers off the provider one at a time.
	pr := p.Providers["aws.x"]
	pr.AssumeRoleARN = ""
	p.Providers["aws.x"] = pr
	if got := resolveAccount(p, &config.Config{}, &Options{}, "aws.x"); got.label != "444444444444" || got.source != SourceAllowedAccountID {
		t.Errorf("allowed_account_ids: %+v", got)
	}
	pr.AllowedAccountIDs = []string{"444444444444", "666666666666"} // more than one is ambiguous
	p.Providers["aws.x"] = pr
	if got := resolveAccount(p, &config.Config{}, &Options{}, "aws.x"); got.label != "555555555555" || got.source != SourceCallerIdentity {
		t.Errorf("caller identity: %+v", got)
	}
	pr.CallerIdentityIDs = nil
	p.Providers["aws.x"] = pr
	// assume_role present but unresolvable: must NOT fall back to the apply account.
	if got := resolveAccount(p, &config.Config{}, &Options{ApplyAccount: "999999999999"}, "aws.x"); got.resolved || got.source != SourceUnresolved {
		t.Errorf("assume_role without a resolvable ARN fell back: %+v", got)
	}
	pr.AssumeRole = false
	p.Providers["aws.x"] = pr
	if got := resolveAccount(p, &config.Config{}, &Options{ApplyAccount: "999999999999"}, "aws.x"); got.label != "999999999999" || got.source != SourceApplyAccount {
		t.Errorf("apply account: %+v", got)
	}
	if got := resolveAccount(p, &config.Config{}, &Options{}, "aws.x"); got.resolved || got.label != "unresolved (alias: x)" {
		t.Errorf("unresolved: %+v", got)
	}
}

func TestAccountFlagOverridesResolvedValue(t *testing.T) {
	p := load(t, "runner-cross-account")
	a := Analyze(p, nil, &Options{
		ApplyAccount:     "999999999999",
		AccountOverrides: map[string]string{"prod": "444444444444"},
	})
	if g := groupsByKey(a)["aws.prod"]; g.Account != "444444444444" || g.Source != SourceFlag {
		t.Errorf("override ignored: %+v", g)
	}
	// The override changes the ARN-mismatch warnings too: the prod role's own
	// ARN names 111111111111, which now disagrees with the provider.
	var warned bool
	for _, c := range a.Changes {
		if c.Address == "aws_iam_role.deployer" && len(c.Warnings) == 1 {
			warned = true
		}
	}
	if !warned {
		t.Error("expected an ARN mismatch warning after override")
	}
}

func TestARNMismatchWarnings(t *testing.T) {
	a := Analyze(load(t, "runner-cross-account"), nil, &Options{ApplyAccount: "999999999999", ApplySource: SourceFlag})
	got := map[string][]Warning{}
	for _, c := range a.Changes {
		if len(c.Warnings) > 0 {
			got[c.Address] = c.Warnings
		}
	}
	if len(got) != 1 {
		t.Fatalf("warnings on %v, want only aws_ssm_parameter.config", got)
	}
	w := got["aws_ssm_parameter.config"]
	if len(w) != 1 || w[0].Path != "kms_key_arn" || w[0].Kind != "arn_account_mismatch" ||
		!strings.Contains(w[0].Message, "111111111111") || !strings.Contains(w[0].Message, "222222222222") {
		t.Errorf("warning: %+v", w)
	}
	if a.Summary.Warnings != 1 {
		t.Errorf("summary warnings = %d", a.Summary.Warnings)
	}
}

func TestIsARNKey(t *testing.T) {
	for k, want := range map[string]bool{"arn": true, "role_arn": true, "subnet_arns": true, "name": false, "arnold": false, "learn": false} {
		if isARNKey(k) != want {
			t.Errorf("%q: got %v", k, !want)
		}
	}
}

func TestDriftHighRiskAndKnownAfterApply(t *testing.T) {
	a := Analyze(load(t, "drift"), nil, nil)
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
	if got := Analyze(load(t, "basic-create"), nil, nil).Summary.KnownAfterApply; got != 4 {
		t.Errorf("known after apply = %d, want 4", got)
	}
}

func TestMovedImportForget(t *testing.T) {
	a := Analyze(load(t, "moved-import-removed"), nil, nil)
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
	a := Analyze(load(t, "deletes-and-replaces"), cfg, nil)
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
