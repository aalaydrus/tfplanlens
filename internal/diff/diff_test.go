package diff

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

func change(addr, action string, attrs ...analyze.Attr) analyze.Change {
	return analyze.Change{Address: addr, Action: action, Account: "1", Attrs: attrs}
}

func val(path, before, after string) analyze.Attr {
	return analyze.Attr{Path: path, Kind: "value", Before: json.RawMessage(before), After: json.RawMessage(after)}
}

func an(cs ...analyze.Change) *analyze.Analysis {
	return &analyze.Analysis{Changes: cs}
}

func addrs(es []Entry) []string {
	out := []string{}
	for _, e := range es {
		out = append(out, e.Address)
	}
	return out
}

func TestCompareClassifiesEveryResource(t *testing.T) {
	oldA := an(
		change("same", "update", val("a", `"1"`, `"2"`)),
		change("gone", "create"),
		change("defused", "replace"),
		change("shifts", "update", val("a", `"1"`, `"2"`)),
		change("noop", "no-op"),
	)
	newA := an(
		change("same", "update", val("a", `"1"`, `"2"`)),
		change("new", "create"),
		change("defused", "update"),
		change("shifts", "update", val("a", `"1"`, `"3"`)),
		change("noop", "no-op"),
		change("danger", "delete"),
	)
	r := Compare(oldA, newA)
	if got := addrs(r.Added); !reflect.DeepEqual(got, []string{"danger", "new"}) {
		t.Errorf("added %v", got)
	}
	if got := addrs(r.Removed); !reflect.DeepEqual(got, []string{"gone"}) {
		t.Errorf("removed %v", got)
	}
	if got := addrs(r.Changed); !reflect.DeepEqual(got, []string{"defused", "shifts"}) {
		t.Errorf("changed %v", got)
	}
	if r.Unchanged != 1 {
		t.Errorf("unchanged %d, want 1 (plain no-ops do not count)", r.Unchanged)
	}
	if !reflect.DeepEqual(r.NewDanger, []string{"danger"}) || !reflect.DeepEqual(r.Defused, []string{"defused"}) {
		t.Errorf("danger %v defused %v", r.NewDanger, r.Defused)
	}
	if got := r.Changed[1].Differences; len(got) != 1 || !strings.Contains(got[0], `attribute a: was "1" => "2", now "1" => "3"`) {
		t.Errorf("differences %v", got)
	}
	if !r.Dangerous() {
		t.Error("Dangerous() = false")
	}
}

func TestNoopToChangeIsAddedAndChangeToNoopIsRemoved(t *testing.T) {
	r := Compare(an(change("a", "no-op"), change("b", "delete")), an(change("a", "delete"), change("b", "no-op")))
	if !reflect.DeepEqual(addrs(r.Added), []string{"a"}) || !r.Added[0].Danger {
		t.Errorf("added %+v", r.Added)
	}
	if !reflect.DeepEqual(addrs(r.Removed), []string{"b"}) || !r.Removed[0].Defused {
		t.Errorf("removed %+v", r.Removed)
	}
}

func TestIdenticalPlansHaveNoDifferences(t *testing.T) {
	a := an(change("x", "update", val("p", `1`, `2`)), change("y", "delete"))
	r := Compare(a, a)
	if !r.Empty() || r.Unchanged != 2 || r.Dangerous() {
		t.Errorf("%+v", r)
	}
	var b bytes.Buffer
	RenderText(&b, r)
	if !strings.Contains(b.String(), "No differences: 2 resource(s)") {
		t.Errorf("text:\n%s", b.String())
	}
}

func TestDifferenceKinds(t *testing.T) {
	sens := func(p string) analyze.Attr { return analyze.Attr{Path: p, Kind: "sensitive"} }
	o := change("r", "replace", val("gone", `1`, `2`), val("flip", `1`, `2`), sens("secret"))
	o.Reason, o.Account = "replace_because_cannot_update", "111"
	o.Attrs[1].ForcesReplace = true
	n := change("r", "update", val("flip", `1`, `2`), val("fresh", `1`, `2`), sens("secret"))
	n.Account = "222"
	got := Compare(an(o), an(n)).Changed[0].Differences
	want := []string{
		"action: replace -> update",
		"reason: replace_because_cannot_update -> (none)",
		"target account: 111 -> 222",
		"attribute flip: forces replacement true -> false",
		"attribute now changes: fresh",
		"attribute no longer changes: gone",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSensitiveValuesNeverCompared(t *testing.T) {
	s1 := analyze.Attr{Path: "p", Kind: "sensitive"}
	r := Compare(an(change("r", "update", s1)), an(change("r", "update", s1)))
	if !r.Empty() {
		t.Errorf("%+v", r.Changed)
	}
}

func TestPlanLevelAndDriftDifferences(t *testing.T) {
	o, n := an(), an()
	o.Summary.Mode, n.Summary.Mode = "normal", "destroy"
	o.Summary.TerraformVersion, n.Summary.TerraformVersion = "1.9.0", "1.10.0"
	o.ApplyAccount.ID, n.ApplyAccount.ID = "111", ""
	o.Drift = []analyze.DriftItem{{Address: "d1"}, {Address: "d2"}}
	n.Drift = []analyze.DriftItem{{Address: "d2"}, {Address: "d3"}}
	r := Compare(o, n)
	if len(r.Plan) != 3 || r.Plan[0] != "plan mode: normal -> destroy" || r.Plan[2] != "apply account: 111 -> (none)" {
		t.Errorf("plan %v", r.Plan)
	}
	if !reflect.DeepEqual(r.DriftNew, []string{"d3"}) || !reflect.DeepEqual(r.DriftGone, []string{"d1"}) {
		t.Errorf("drift %v %v", r.DriftNew, r.DriftGone)
	}
}

func TestJSONArraysAreNeverNull(t *testing.T) {
	var b bytes.Buffer
	if err := RenderJSON(&b, Compare(an(), an())); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"plan", "added", "removed", "changed", "new_danger", "defused", "drift_new", "drift_gone"} {
		if _, ok := m[k].([]any); !ok {
			t.Errorf("%s is %T, want an array", k, m[k])
		}
	}
}

func TestMarkdownEscapesAndMarks(t *testing.T) {
	r := Compare(an(), an(change("a`b|c", "delete")))
	var b bytes.Buffer
	if err := RenderMarkdown(&b, r); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"<!-- tfplanlens-diff -->", "**New danger**", "``a`b|c``", "<details>"} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown missing %q:\n%s", want, out)
		}
	}
}
