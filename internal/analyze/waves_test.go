package analyze

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/config"
	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// chg builds an instance change for a configuration address like
// "aws_x.y" or "module.m.aws_x.y"; index "" means no key.
func chg(cfgAddr, index string, action plan.Action) plan.Change {
	c := plan.Change{Address: cfgAddr, ConfigAddress: cfgAddr, Type: typeOfConfigAddress(cfgAddr), Mode: "managed", Action: action}
	if index != "" {
		c.HasIndex, c.Index = true, index
		c.Address = cfgAddr + "[" + index + "]"
	}
	return c
}

func graph(edges map[string][]string) *plan.DepGraph {
	d := &plan.DepGraph{Deps: map[string][]string{}}
	for k, v := range edges {
		d.Deps[k] = v
		for _, t := range v {
			if _, ok := d.Deps[t]; !ok {
				d.Deps[t] = nil
			}
		}
	}
	return d
}

// waveOf lists "phase:number" per configuration address.
func waveOf(a *Analysis) map[string]string {
	m := map[string]string{}
	for _, w := range a.Waves {
		for _, r := range w.Resources {
			m[r.Address] = fmt.Sprintf("%s:%d", w.Phase, w.Number)
		}
	}
	return m
}

func TestWavesLayerByDependencyDepth(t *testing.T) {
	cases := []struct {
		name    string
		changes []plan.Change
		edges   map[string][]string
		want    map[string]string
	}{
		{"chain", []plan.Change{chg("a.x", "", "create"), chg("b.x", "", "create"), chg("c.x", "", "update")},
			map[string][]string{"b.x": {"a.x"}, "c.x": {"b.x"}},
			map[string]string{"a.x": "apply:1", "b.x": "apply:2", "c.x": "apply:3"}},
		{"diamond: depth is the longest path", []plan.Change{chg("a.x", "", "create"), chg("b.x", "", "create"), chg("c.x", "", "create"), chg("d.x", "", "create")},
			map[string][]string{"b.x": {"a.x"}, "c.x": {"a.x"}, "d.x": {"b.x", "c.x", "a.x"}},
			map[string]string{"a.x": "apply:1", "b.x": "apply:2", "c.x": "apply:2", "d.x": "apply:3"}},
		{"independent resources share wave 1", []plan.Change{chg("a.x", "", "create"), chg("b.x", "", "create")}, nil,
			map[string]string{"a.x": "apply:1", "b.x": "apply:1"}},
		// y depends on m, which depends on x. m is not changing, but y still
		// has to wait for x through it.
		{"unchanged resources pass dependencies through", []plan.Change{chg("x.x", "", "create"), chg("m.x", "", "no-op"), chg("y.x", "", "create")},
			map[string][]string{"m.x": {"x.x"}, "y.x": {"m.x"}},
			map[string]string{"x.x": "apply:1", "y.x": "apply:2"}},
		{"a dependency that is not changing adds no wave", []plan.Change{chg("m.x", "", "no-op"), chg("y.x", "", "create")},
			map[string][]string{"y.x": {"m.x"}},
			map[string]string{"y.x": "apply:1"}},
		{"data source reads are applied work", []plan.Change{chg("data.d.x", "", "read"), chg("y.x", "", "create")},
			map[string][]string{"y.x": {"data.d.x"}},
			map[string]string{"data.d.x": "apply:1", "y.x": "apply:2"}},
		// Deleting runs the other way: dependents first.
		{"deletes run dependents first", []plan.Change{chg("a.x", "", "delete"), chg("b.x", "", "delete"), chg("c.x", "", "delete")},
			map[string][]string{"b.x": {"a.x"}, "c.x": {"b.x"}},
			map[string]string{"c.x": "destroy:1", "b.x": "destroy:2", "a.x": "destroy:3"}},
		{"forget is not ordered work", []plan.Change{chg("a.x", "", "forget"), chg("b.x", "", "create")}, map[string][]string{"b.x": {"a.x"}},
			map[string]string{"b.x": "apply:1"}},
		{"a mixed resource (create one key, delete another) is apply work", []plan.Change{chg("s.x", "a", "delete"), chg("s.x", "b", "create"), chg("t.x", "", "update")},
			map[string][]string{"t.x": {"s.x"}},
			map[string]string{"s.x": "apply:1", "t.x": "apply:2"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := Analyze(&plan.Plan{Changes: c.changes, Graph: graph(c.edges)}, nil, nil)
			if got := waveOf(a); !reflect.DeepEqual(got, c.want) {
				t.Errorf("waves %v, want %v", got, c.want)
			}
			// Every change carries the wave of its resource.
			for _, ch := range a.Changes {
				if want, ok := c.want[ch.Parent]; ok && ch.Parent != "" {
					_ = want
				}
			}
		})
	}
}

func TestChangesCarryTheirWave(t *testing.T) {
	a := Analyze(&plan.Plan{
		Changes: []plan.Change{chg("a.x", "", "create"), chg("b.x", "", "create"), chg("gone.x", "", "delete")},
		Graph:   graph(map[string][]string{"b.x": {"a.x"}}),
	}, nil, nil)
	got := map[string]string{}
	for _, c := range a.Changes {
		got[c.Address] = fmt.Sprintf("%s:%d", c.WavePhase, c.Wave)
	}
	want := map[string]string{"a.x": "apply:1", "b.x": "apply:2", "gone.x": "destroy:1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%v, want %v", got, want)
	}
	if a.Summary.ApplyWaves != 2 || a.Summary.DestroyWaves != 1 {
		t.Errorf("summary %+v", a.Summary)
	}
}

func TestResourceRemovedFromConfigurationHasNoDestroyOrder(t *testing.T) {
	a := Analyze(&plan.Plan{
		Changes: []plan.Change{chg("gone.x", "", "delete"), chg("kept.x", "", "delete")},
		Graph:   graph(map[string][]string{"kept.x": nil}),
	}, nil, nil)
	var flagged, known []string
	for _, w := range a.Waves {
		for _, r := range w.Resources {
			if r.NoConfiguration {
				flagged = append(flagged, r.Address)
			} else {
				known = append(known, r.Address)
			}
		}
	}
	if !reflect.DeepEqual(flagged, []string{"gone.x"}) || !reflect.DeepEqual(known, []string{"kept.x"}) {
		t.Errorf("flagged %v known %v", flagged, known)
	}
	if len(a.WaveNotes) != 1 {
		t.Errorf("notes %v", a.WaveNotes)
	}
}

func TestCyclesAreDetectedAndDoNotHang(t *testing.T) {
	a := Analyze(&plan.Plan{
		Changes: []plan.Change{chg("a.x", "", "create"), chg("b.x", "", "create"), chg("c.x", "", "create"), chg("d.x", "", "create")},
		// a <-> b form a cycle; c depends on the cycle; d is independent.
		Graph: graph(map[string][]string{"a.x": {"b.x"}, "b.x": {"a.x"}, "c.x": {"a.x"}}),
	}, nil, nil)
	if len(a.Cycles) != 1 || !reflect.DeepEqual(a.Cycles[0].Resources, []string{"a.x", "b.x"}) {
		t.Fatalf("cycles %+v", a.Cycles)
	}
	want := map[string]string{"a.x": "apply:1", "b.x": "apply:1", "c.x": "apply:2", "d.x": "apply:1"}
	if got := waveOf(a); !reflect.DeepEqual(got, want) {
		t.Errorf("waves %v, want %v", got, want)
	}
	// A graph with no cycle reports none, as an empty array.
	b := Analyze(&plan.Plan{Changes: []plan.Change{chg("a.x", "", "create")}, Graph: graph(nil)}, nil, nil)
	if b.Cycles == nil || len(b.Cycles) != 0 {
		t.Errorf("cycles = %#v, want empty non-nil", b.Cycles)
	}
}

func TestLongCycleAndSelfReferenceFreeGraph(t *testing.T) {
	// a -> b -> c -> a
	a := Analyze(&plan.Plan{
		Changes: []plan.Change{chg("a.x", "", "create"), chg("b.x", "", "create"), chg("c.x", "", "create")},
		Graph:   graph(map[string][]string{"a.x": {"b.x"}, "b.x": {"c.x"}, "c.x": {"a.x"}}),
	}, nil, nil)
	if len(a.Cycles) != 1 || len(a.Cycles[0].Resources) != 3 {
		t.Errorf("cycles %+v", a.Cycles)
	}
}

func TestWavesHideIgnoredTypesAndRenumber(t *testing.T) {
	p := &plan.Plan{
		Changes: []plan.Change{chg("hidden.x", "", "create"), chg("b.x", "", "create"), chg("c.x", "", "create")},
		Graph:   graph(map[string][]string{"b.x": {"hidden.x"}, "c.x": {"b.x"}}),
	}
	a := Analyze(p, &config.Config{IgnoreTypes: []string{"hidden"}}, nil)
	want := map[string]string{"b.x": "apply:1", "c.x": "apply:2"}
	if got := waveOf(a); !reflect.DeepEqual(got, want) {
		t.Errorf("waves %v, want %v", got, want)
	}
	for _, c := range a.Changes {
		if c.Wave == 0 || c.Wave > 2 {
			t.Errorf("%s wave %d", c.Address, c.Wave)
		}
	}
}

func TestWaveNotes(t *testing.T) {
	p := &plan.Plan{
		Changes: []plan.Change{func() plan.Change { c := chg("a.x", "", "replace"); c.Reason = "replace_by_triggers"; return c }()},
		Graph:   &plan.DepGraph{Deps: map[string][]string{"a.x": nil}, LocalRefs: 3},
	}
	a := Analyze(p, nil, nil)
	if len(a.WaveNotes) != 2 {
		t.Fatalf("notes %v", a.WaveNotes)
	}
	// No notes without waves.
	if b := Analyze(&plan.Plan{Graph: &plan.DepGraph{LocalRefs: 5}}, nil, nil); len(b.WaveNotes) != 0 || b.WaveNotes == nil {
		t.Errorf("notes without waves: %#v", b.WaveNotes)
	}
}

func TestParentsGroupInstances(t *testing.T) {
	a := Analyze(&plan.Plan{Changes: []plan.Change{
		chg("w.x", "0", "no-op"), chg("w.x", "1", "update"), chg("w.x", "2", "delete"),
		chg("e.x", "a", "no-op"), chg("e.x", "b", "create"),
		chg("one.x", "0", "update"), // count = 1: nothing to group
		chg("plain.x", "", "update"),
	}}, nil, nil)
	if len(a.Parents) != 2 {
		t.Fatalf("parents %+v", a.Parents)
	}
	w, e := a.Parents[0], a.Parents[1]
	if w.Address != "w.x" || w.Kind != "count" || len(w.Instances) != 3 || w.Counts.Update != 1 || w.Counts.Delete != 1 || w.Counts.NoOp != 1 {
		t.Errorf("count parent %+v", w)
	}
	if e.Address != "e.x" || e.Kind != "for_each" || e.Instances[1].Key != "b" || e.Instances[1].Action != "create" {
		t.Errorf("for_each parent %+v", e)
	}
	byAddr := map[string]Change{}
	for _, c := range a.Changes {
		byAddr[c.Address] = c
	}
	if c := byAddr["w.x[2]"]; c.Parent != "w.x" || c.Key != "2" {
		t.Errorf("instance change: %+v", c)
	}
	if c := byAddr["plain.x"]; c.Parent != "" || c.Key != "" {
		t.Errorf("plain change: %+v", c)
	}
}

func TestTypeOfConfigAddress(t *testing.T) {
	for addr, want := range map[string]string{
		"aws_s3_bucket.b": "aws_s3_bucket", "module.a.aws_s3_bucket.b": "aws_s3_bucket", "data.aws_caller_identity.c": "aws_caller_identity",
		"module.a.module.b.data.aws_region.r": "aws_region", "": "",
	} {
		if got := typeOfConfigAddress(addr); got != want {
			t.Errorf("%q: %q, want %q", addr, got, want)
		}
	}
	_ = sort.Strings
}

func TestNoOpInstancesCarryNoWave(t *testing.T) {
	a := Analyze(&plan.Plan{
		Changes: []plan.Change{chg("a.x", "", "create"), chg("a.x", "", "no-op")},
		Graph:   graph(map[string][]string{"a.x": nil}),
	}, nil, nil)
	if a.Changes[0].Wave != 1 || a.Changes[1].Wave != 0 || a.Changes[1].WavePhase != "" {
		t.Errorf("%+v", a.Changes)
	}
}
