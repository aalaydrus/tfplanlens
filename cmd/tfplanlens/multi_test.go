package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeriveLabels(t *testing.T) {
	for name, tc := range map[string]struct {
		in   []string
		want []string
	}{
		"file names":             {[]string{"vpc.json", "app.json"}, []string{"vpc", "app"}},
		"directory per unit":     {[]string{"live/vpc/plan.json", "live/app/plan.json"}, []string{"vpc", "app"}},
		"same dir name":          {[]string{"prod/vpc/plan.json", "stage/vpc/plan.json"}, []string{"prod/vpc", "stage/vpc"}},
		"dot slash and tfplan":   {[]string{"./a/tfplan.json", "./b/tfplan.json"}, []string{"a", "b"}},
		"explicit wins":          {[]string{"net=x/plan.json", "y/plan.json"}, []string{"net", "y"}},
		"only generic file name": {[]string{"plan.json", "other/plan.json"}, []string{"plan", "other"}},
	} {
		paths, labels, explicit := make([]string, len(tc.in)), make([]string, len(tc.in)), make([]bool, len(tc.in))
		for i, s := range tc.in {
			paths[i], labels[i], explicit[i] = splitLabel(s)
		}
		got, err := deriveLabels(paths, labels, explicit)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v %v, want %v", name, got, err, tc.want)
		}
	}
	if _, err := deriveLabels([]string{"a/plan.json", "a/plan.json"}, []string{"", ""}, []bool{false, false}); err == nil || !strings.Contains(err.Error(), "label=path") {
		t.Errorf("identical paths: %v", err)
	}
}

func TestSplitLabelLeavesPathsWithEqualsAlone(t *testing.T) {
	for in, want := range map[string][3]any{
		"net=a.json":        {"a.json", "net", true},
		"dir=x/plan.json":   {"x/plan.json", "dir", true},
		"dir/a=b/plan.json": {"dir/a=b/plan.json", "", false},
		"plain.json":        {"plain.json", "", false},
		"=a.json":           {"=a.json", "", false},
	} {
		p, l, e := splitLabel(in)
		if p != want[0] || l != want[1] || e != want[2] {
			t.Errorf("%q: %q %q %v, want %v", in, p, l, e, want)
		}
	}
}

// The same real plan twice: every address exists in both plans, so a merge
// that did not prefix them would collide.
func TestRealMultiPlanReport(t *testing.T) {
	files := realPlans(t, "complex")
	simple, _ := filepath.Glob("../../testdata/plans/real/simple-tf*.json")
	if len(simple) == 0 {
		t.Skip("no simple plan")
	}
	t.Setenv(envApplyAccount, "")
	code, out, errs := runCLI("report", "x="+files[0], "y="+files[0], "s="+simple[0], "--format", "json", "--tfapply-account-id", "111111111111")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	var m struct {
		Summary struct {
			Total   int
			Actions map[string]int
		}
		Units   []struct{ Label string }
		Changes []struct{ Address, Unit string }
		Waves   []struct {
			Phase     string
			Number    int
			Resources []struct{ Address string }
		}
	}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Units) != 3 {
		t.Errorf("units %+v", m.Units)
	}
	seen := map[string]bool{}
	for _, c := range m.Changes {
		if seen[c.Address] {
			t.Errorf("duplicate address %s", c.Address)
		}
		seen[c.Address] = true
		if !strings.HasPrefix(c.Address, c.Unit+":") {
			t.Errorf("%s is not prefixed with its unit %s", c.Address, c.Unit)
		}
	}
	for _, want := range []string{"x:terraform_data.db", "y:terraform_data.db", "s:terraform_data.db"} {
		if !seen[want] {
			t.Errorf("missing %s", want)
		}
	}
	// Two copies of complex plus simple: counts add up.
	single, _ := analyzeReal(t, files[0])
	if m.Summary.Actions["delete"] != 2*single.Summary.Actions["delete"]+1 {
		t.Errorf("deletes %d, single complex has %d", m.Summary.Actions["delete"], single.Summary.Actions["delete"])
	}
	w2 := map[string]bool{}
	for _, w := range m.Waves {
		if w.Phase == "apply" && w.Number == 2 {
			for _, r := range w.Resources {
				w2[r.Address] = true
			}
		}
	}
	if !w2["x:terraform_data.lb"] || !w2["y:terraform_data.lb"] {
		t.Errorf("apply wave 2 should hold lb of both copies: %v", w2)
	}

	for _, format := range []string{"text", "markdown", "html"} {
		code, out, errs := runCLI("report", "x="+files[0], "y="+files[0], "--format", format, "--no-color", "--tfapply-account-id", "111111111111")
		if code != exitOK || !strings.Contains(out, "x:terraform_data.db") || !strings.Contains(out, "y:terraform_data.db") {
			t.Errorf("%s: exit %d %s", format, code, errs)
		}
	}
	_, html, _ := runCLI("report", "x="+files[0], "y="+files[0], "--format", "html", "--tfapply-account-id", "111111111111")
	if !strings.Contains(html, `<option value="unit">plan</option>`) || !strings.Contains(html, `data-unit="y"`) {
		t.Errorf("html has no group-by-plan")
	}
	_, single1, _ := runCLI("report", files[0], "--format", "html", "--tfapply-account-id", "111111111111")
	if strings.Contains(single1, `value="unit"`) {
		t.Errorf("single-plan html offers group by plan")
	}
}

func TestMultiPlanFailOnNamesThePlan(t *testing.T) {
	files := realPlans(t, "complex")
	simple, _ := filepath.Glob("../../testdata/plans/real/simple-tf*.json")
	if len(simple) == 0 {
		t.Skip("no simple plan")
	}
	code, _, errs := runCLI("report", "a="+files[0], "b="+simple[0], "--fail-on", "delete:terraform_data", "--tfapply-account-id", "111111111111")
	if code != exitFailOn || !strings.Contains(errs, "a:terraform_data.worker[2]") || !strings.Contains(errs, "b:terraform_data.old") {
		t.Errorf("exit %d: %s", code, errs)
	}
}

func TestMultiPlanRejectsStdinAndBadLabels(t *testing.T) {
	files := realPlans(t, "complex")
	if code, _, errs := runCLI("report", files[0], "-"); code == exitOK || !strings.Contains(errs, "single plan") {
		t.Errorf("stdin with several plans: exit %d %s", code, errs)
	}
	if code, _, errs := runCLI("report", files[0], files[0]); code == exitOK || !strings.Contains(errs, "label=path") {
		t.Errorf("identical paths: exit %d %s", code, errs)
	}
	if code, _, errs := runCLI("report", files[0], "nope.json"); code == exitOK || !strings.Contains(errs, "nope.json") {
		t.Errorf("missing file should name it: exit %d %s", code, errs)
	}
}
