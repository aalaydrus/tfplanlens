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
	// multi-account: the default provider says nothing about the account.
	a := Analyze(load(t, "multi-account"), nil, nil)
	g := groupsByKey(a)
	if g["aws.mystery"].Resolved || g["aws.mystery"].Source != SourceUnresolved || g["aws.mystery"].Account != "unresolved (alias: mystery)" {
		t.Errorf("mystery: %+v", g["aws.mystery"])
	}
	// Without an apply account nothing is flagged as cross-account.
	for k, grp := range g {
		if grp.Cross {
			t.Errorf("%s flagged cross-account with unknown apply account", k)
		}
	}
	if a.ApplyAccount.Resolved || a.ApplyAccount.Source != SourceUnresolved || len(a.Notices) != 1 {
		t.Errorf("apply account %+v notices %v", a.ApplyAccount, a.Notices)
	}
}

// With no apply account given, it is the account of the default provider.
func TestApplyAccountInferredFromDefaultProvider(t *testing.T) {
	a := Analyze(load(t, "runner-cross-account"), nil, nil)
	if got := a.ApplyAccount; got.ID != "999999999999" || got.Source != SourceInferred || got.Via != SourceCallerIdentity || !got.Resolved {
		t.Fatalf("apply account %+v", got)
	}
	if got := a.ApplyAccount.Describe(); got != "source: inferred from the default provider via caller_identity" {
		t.Errorf("describe %q", got)
	}
	g := groupsByKey(a)
	if g["aws.sandbox"].Account != "999999999999" || g["aws.sandbox"].Source != SourceApplyAccount || !g["aws.prod"].Cross || g["aws.sandbox"].Cross {
		t.Errorf("groups %+v", g)
	}
	if len(a.Notices) != 0 {
		t.Errorf("notices %v", a.Notices)
	}
	// An explicit apply account wins over the inferred one.
	b := Analyze(load(t, "runner-cross-account"), nil, &Options{ApplyAccount: "444444444444", ApplySource: SourceFlag})
	if b.ApplyAccount.ID != "444444444444" || b.ApplyAccount.Source != SourceFlag || b.ApplyAccount.Via != "" {
		t.Errorf("explicit: %+v", b.ApplyAccount)
	}
}

func TestInferApplyAccountRules(t *testing.T) {
	provider := func(pr plan.Provider) *plan.Plan {
		return &plan.Plan{Providers: map[string]plan.Provider{pr.Key: pr}}
	}
	cases := []struct {
		name string
		pr   plan.Provider
		cfg  *config.Config
		want string
		via  string
	}{
		{"allowed_account_ids", plan.Provider{Key: "aws", Name: "aws", AllowedAccountIDs: []string{"123456789012"}}, nil, "123456789012", SourceAllowedAccountID},
		{"caller identity", plan.Provider{Key: "aws", Name: "aws", CallerIdentityIDs: []string{"123456789012"}}, nil, "123456789012", SourceCallerIdentity},
		{"accounts: default", plan.Provider{Key: "aws", Name: "aws"}, &config.Config{Accounts: map[string]string{"default": "210987654321"}}, "210987654321", SourceConfig},
		// A default provider that assumes a role says nothing about where the
		// runner lives.
		{"assume_role default provider", plan.Provider{Key: "aws", Name: "aws", AssumeRole: true, AssumeRoleARN: "arn:aws:iam::111111111111:role/x", RoleARNVia: plan.ViaLiteral, CallerIdentityIDs: []string{"123456789012"}}, nil, "", ""},
		{"ambiguous caller identity", plan.Provider{Key: "aws", Name: "aws", CallerIdentityIDs: []string{"123456789012", "210987654321"}}, nil, "", ""},
		{"nothing known", plan.Provider{Key: "aws", Name: "aws"}, nil, "", ""},
	}
	for _, c := range cases {
		cfg := c.cfg
		if cfg == nil {
			cfg = &config.Config{}
		}
		id, via := inferApplyAccount(provider(c.pr), cfg, &Options{})
		if id != c.want || via != c.via {
			t.Errorf("%s: %q via %q, want %q via %q", c.name, id, via, c.want, c.via)
		}
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

// Found by the real-plan smoke test: known-after-apply attributes were listed
// with their (often huge) old values.
func TestUnknownAttributesAreCountedNotListed(t *testing.T) {
	attrs := diffAttrs(
		map[string]any{"id": "old-id", "output": strings.Repeat("x", 500), "name": "a"},
		map[string]any{"name": "b"},
		nil, nil, map[string]any{"id": true, "output": true}, nil, nil)
	if len(attrs) != 1 || attrs[0].Path != "name" {
		t.Errorf("attrs = %+v, want only name", attrs)
	}
}

// Found by the real-plan smoke test: output = [a, b, c] becoming unknown also
// reported output[0], output[1], output[2] as "-> (absent)".
func TestMarkedLeafCollapsesTheOtherSidesChildren(t *testing.T) {
	attrs := diffAttrs(
		map[string]any{"output": []any{"a", "b", "c"}, "input": []any{"a", "b"}},
		map[string]any{"input": []any{"a", "x"}},
		nil, nil, map[string]any{"output": true}, nil, nil)
	if len(attrs) != 1 || attrs[0].Path != "input[1]" {
		t.Errorf("attrs = %+v, want only input[1]", attrs)
	}
}

// Found by the real-plan smoke test: terraform_data.output mirrors a
// sensitive input but Terraform does not mark the copy sensitive, so the
// secret was printed in clear.
func TestSecretCopiedIntoUnmarkedAttributeIsMasked(t *testing.T) {
	attrs := diffAttrs(
		map[string]any{"input": "hunter2", "output": "hunter2", "note": "hi"},
		map[string]any{"input": "hunter3", "output": "hunter3", "note": "bye"},
		map[string]any{"input": true}, map[string]any{"input": true}, nil, nil, nil)
	for _, at := range attrs {
		if at.Before != nil || at.After != nil {
			if at.Path != "note" {
				t.Errorf("%s printed values: %s -> %s", at.Path, at.Before, at.After)
			}
		}
		if strings.Contains(string(at.Before)+string(at.After), "hunter") {
			t.Errorf("secret leaked via %s", at.Path)
		}
	}
	kinds := map[string]string{}
	for _, at := range attrs {
		kinds[at.Path] = at.Kind
	}
	if kinds["input"] != "sensitive" || kinds["output"] != "sensitive" || kinds["note"] != "value" {
		t.Errorf("kinds = %v", kinds)
	}
	// Short values are not matched: a sensitive "1" must not mask every "1".
	short := diffAttrs(
		map[string]any{"pw": "1", "count": "1"}, map[string]any{"pw": "2", "count": "2"},
		map[string]any{"pw": true}, map[string]any{"pw": true}, nil, nil, nil)
	for _, at := range short {
		if at.Path == "count" && at.Kind != "value" {
			t.Errorf("short secret over-masked: %+v", at)
		}
	}
}

func TestKnownAfterApplyCountShownInLabel(t *testing.T) {
	c := Change{Action: "update", KnownAfterApply: 2}
	if got := c.Label(); got != "update, 2 known after apply" {
		t.Errorf("label %q", got)
	}
}

func TestForEachRenameWarningIsNotCalledAnIndexShift(t *testing.T) {
	a := Analyze(load(t, "for-each-shift"), nil, nil)
	for _, f := range a.Findings {
		if f.Reason == "delete_because_each_key" && f.IndexShift && strings.Contains(f.Warning, "index") {
			t.Errorf("for_each warning mentions an index shift: %s", f.Warning)
		}
	}
}

// Found by the real AWS plan: the provider reads back "{}" where state had
// null for tags, so every resource showed up as drifted (and a replaced one as
// "high risk"). Terraform's own plan output does not report it.
func TestNullVersusEmptyCollectionIsNotDrift(t *testing.T) {
	mk := func(addr string, action plan.Action, before, after any) plan.Change {
		return plan.Change{Address: addr, Type: "aws_sqs_queue", Mode: "managed", Action: action, Before: before, After: after}
	}
	p := &plan.Plan{
		Changes: []plan.Change{mk("aws_sqs_queue.q", plan.ActionDelete, map[string]any{"name": "q"}, nil)},
		Drift: []plan.Change{
			mk("aws_sqs_queue.noise", plan.ActionUpdate, map[string]any{"tags": nil, "list": []any{}}, map[string]any{"tags": map[string]any{}, "list": nil}),
			mk("aws_sqs_queue.q", plan.ActionUpdate, map[string]any{"tags": nil}, map[string]any{"tags": map[string]any{}}), // also being deleted by the plan
			mk("aws_sqs_queue.real", plan.ActionUpdate, map[string]any{"visibility": 30.0, "tags": nil}, map[string]any{"visibility": 60.0, "tags": map[string]any{}}),
			mk("aws_sqs_queue.gone", plan.ActionDelete, map[string]any{"name": "gone"}, nil),
		},
	}
	a := Analyze(p, nil, nil)
	got := map[string][]string{}
	for _, d := range a.Drift {
		for _, at := range d.Attrs {
			got[d.Address] = append(got[d.Address], at.Path)
		}
		if _, ok := got[d.Address]; !ok {
			got[d.Address] = nil
		}
	}
	if len(a.Drift) != 2 || got["aws_sqs_queue.real"] == nil || len(got["aws_sqs_queue.real"]) != 1 || got["aws_sqs_queue.real"][0] != "visibility" {
		t.Errorf("drift = %v, want only 'real' (visibility) and 'gone'", got)
	}
	if _, ok := got["aws_sqs_queue.gone"]; !ok {
		t.Error("a delete is always real drift")
	}
	// The null-vs-empty entry must not mark the deleted resource high risk.
	if a.Findings[0].DriftAlso {
		t.Error("finding flagged as drifted because of null-vs-empty noise")
	}
	// ...and --fail-on drift sees only real drift.
	noiseOnly := &plan.Plan{Drift: p.Drift[:2]}
	if m := CheckFailOn(noiseOnly, Analyze(noiseOnly, nil, nil), []Rule{{Kind: "drift"}}); len(m) != 0 {
		t.Errorf("--fail-on drift matched noise: %v", m)
	}
	if m := CheckFailOn(p, a, []Rule{{Kind: "drift"}}); len(m) != 2 {
		t.Errorf("--fail-on drift matches = %v, want real + gone", m)
	}
}
