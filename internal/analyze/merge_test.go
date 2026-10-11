package analyze

import (
	"reflect"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

func unit(addr string, action plan.Action, apply string) *Analysis {
	return Analyze(&plan.Plan{
		Changes: []plan.Change{chg(addr, "", action), chg("dep.x", "", "create")},
		Graph:   graph(map[string][]string{addr: {"dep.x"}}),
	}, nil, &Options{ApplyAccount: apply, ApplySource: SourceFlag})
}

func TestMergeKeepsIdenticalAddressesApart(t *testing.T) {
	m := Merge([]string{"vpc", "app"}, []*Analysis{unit("a.x", "delete", "111"), unit("a.x", "update", "111")})
	var got []string
	for _, c := range m.Changes {
		got = append(got, c.Address+"/"+c.Unit)
	}
	want := []string{"vpc:a.x/vpc", "vpc:dep.x/vpc", "app:a.x/app", "app:dep.x/app"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes %v, want %v", got, want)
	}
	if m.Summary.Actions.Create != 2 || m.Summary.Actions.Delete != 1 || m.Summary.Actions.Update != 1 || m.Summary.Total != 4 {
		t.Errorf("summary %+v", m.Summary)
	}
	if len(m.Findings) != 1 || m.Findings[0].Address != "vpc:a.x" || m.Findings[0].Unit != "vpc" {
		t.Errorf("findings %+v", m.Findings)
	}
	if len(m.Units) != 2 || m.Units[0].Label != "vpc" || m.Units[0].Actions.Delete != 1 || m.Units[1].Actions.Update != 1 {
		t.Errorf("units %+v", m.Units)
	}
	if m.Summary.ByModule["vpc:root"] != 2 || m.Summary.ByModule["app:root"] != 2 {
		t.Errorf("by module %v", m.Summary.ByModule)
	}
}

func TestMergeWavesAreJoinedPerNumber(t *testing.T) {
	m := Merge([]string{"u1", "u2"}, []*Analysis{unit("a.x", "update", "1"), unit("b.x", "update", "1")})
	// Each unit: dep.x in apply wave 1, its resource in apply wave 2.
	if len(m.Waves) != 2 || m.Waves[0].Number != 1 || len(m.Waves[0].Resources) != 2 {
		t.Fatalf("waves %+v", m.Waves)
	}
	var got []string
	for _, r := range m.Waves[1].Resources {
		got = append(got, r.Address)
	}
	if !reflect.DeepEqual(got, []string{"u1:a.x", "u2:b.x"}) {
		t.Errorf("wave 2 %v", got)
	}
	if m.Summary.ApplyWaves != 2 {
		t.Errorf("apply waves %d", m.Summary.ApplyWaves)
	}
	found := false
	for _, n := range m.WaveNotes {
		if len(n) > 20 && n[:20] == "Waves are computed i" {
			found = true
		}
	}
	if !found {
		t.Errorf("no cross-plan dependency note: %v", m.WaveNotes)
	}
}

func TestMergeApplyAccount(t *testing.T) {
	same := Merge([]string{"a", "b"}, []*Analysis{unit("a.x", "update", "111"), unit("a.x", "update", "111")})
	if same.ApplyAccount.ID != "111" || same.ApplyAccount.Source != SourceFlag {
		t.Errorf("same: %+v", same.ApplyAccount)
	}
	mixed := Merge([]string{"a", "b"}, []*Analysis{unit("a.x", "update", "111"), unit("a.x", "update", "222")})
	if mixed.ApplyAccount.Source != SourceMixed || !mixed.ApplyAccount.Resolved {
		t.Errorf("mixed: %+v", mixed.ApplyAccount)
	}
	if mixed.Units[0].ApplyAccount.ID != "111" || mixed.Units[1].ApplyAccount.ID != "222" {
		t.Errorf("units keep their own account: %+v", mixed.Units)
	}
}

func TestMergeModesAndNeverNil(t *testing.T) {
	a, b := unit("a.x", "update", "1"), unit("a.x", "update", "1")
	a.Summary.Mode, b.Summary.Mode = "normal", "destroy"
	m := Merge([]string{"a", "b"}, []*Analysis{a, b})
	if m.Summary.Mode != "mixed" {
		t.Errorf("mode %q", m.Summary.Mode)
	}
	e := Merge([]string{"a"}, []*Analysis{Analyze(&plan.Plan{}, nil, nil)})
	if e.Findings == nil || e.Waves == nil || e.Parents == nil || e.Cycles == nil || e.Notices == nil || e.WaveNotes == nil {
		t.Errorf("nil slice in %+v", e)
	}
}
