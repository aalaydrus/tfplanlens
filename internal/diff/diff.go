// Package diff compares two analyses of the same configuration ("what
// changed since the last time I reviewed this plan?", SPEC 10). Resources
// are matched by address (a resource the new plan moved from an address the
// old plan listed counts as the same one); a resource that is absent or a plain no-op counts
// as "not changing".
package diff

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

// SchemaVersion of the JSON output.
const SchemaVersion = 1

// Kinds of entry.
const (
	Added   = "added"   // changes in the new plan only
	Removed = "removed" // changed in the old plan, no longer changes
	Changed = "changed" // changes in both, differently
)

// Result is the difference between an old and a new plan.
type Result struct {
	SchemaVersion int `json:"schema_version"`
	// Plan lists plan-wide differences (mode, Terraform version, apply account).
	Plan []string `json:"plan"`
	// Old and New are the action counts of each plan.
	Old analyze.ActionCounts `json:"old"`
	New analyze.ActionCounts `json:"new"`

	Added   []Entry `json:"added"`
	Removed []Entry `json:"removed"`
	Changed []Entry `json:"changed"`
	// Unchanged counts resources that change identically in both plans.
	Unchanged int `json:"unchanged"`

	// NewDanger lists addresses that are deleted or replaced by the new plan
	// and were not by the old one. Defused lists the reverse.
	NewDanger []string `json:"new_danger"`
	Defused   []string `json:"defused"`

	// DriftNew and DriftGone list addresses that drifted only in the new or
	// only in the old plan.
	DriftNew  []string `json:"drift_new"`
	DriftGone []string `json:"drift_gone"`
}

// Entry is one resource that differs.
type Entry struct {
	Address string `json:"address"`
	Kind    string `json:"kind"`
	// OldAction and NewAction are empty when the resource is not changing.
	OldAction string `json:"old_action,omitempty"`
	NewAction string `json:"new_action,omitempty"`
	// Differences says exactly what differs for a changed resource.
	Differences []string `json:"differences"`
	// Danger is set when the new plan deletes or replaces it and the old one
	// did not.
	Danger bool `json:"danger,omitempty"`
	// Defused is set when the old plan deleted or replaced it and the new
	// one does not.
	Defused bool `json:"defused,omitempty"`
}

// Dangerous reports whether the new plan has any newly destructive change.
func (r *Result) Dangerous() bool { return len(r.NewDanger) > 0 }

// Empty reports whether the two plans have no differences at all.
func (r *Result) Empty() bool {
	return len(r.Plan) == 0 && len(r.Added) == 0 && len(r.Removed) == 0 && len(r.Changed) == 0 &&
		len(r.DriftNew) == 0 && len(r.DriftGone) == 0
}

func destructive(a string) bool { return a == "delete" || a == "replace" }

// Compare diffs two analyses. The old plan is the one reviewed before.
func Compare(oldA, newA *analyze.Analysis) *Result {
	r := &Result{
		SchemaVersion: SchemaVersion,
		Plan:          []string{},
		Old:           oldA.Summary.Actions,
		New:           newA.Summary.Actions,
		Added:         []Entry{},
		Removed:       []Entry{},
		Changed:       []Entry{},
		NewDanger:     []string{},
		Defused:       []string{},
		DriftNew:      []string{},
		DriftGone:     []string{},
	}
	if o, n := oldA.Summary.Mode, newA.Summary.Mode; o != n {
		r.Plan = append(r.Plan, fmt.Sprintf("plan mode: %s -> %s", o, n))
	}
	if o, n := oldA.Summary.TerraformVersion, newA.Summary.TerraformVersion; o != n {
		r.Plan = append(r.Plan, fmt.Sprintf("terraform version: %s -> %s", orNone(o), orNone(n)))
	}
	if o, n := oldA.ApplyAccount.ID, newA.ApplyAccount.ID; o != n {
		r.Plan = append(r.Plan, fmt.Sprintf("apply account: %s -> %s", orNone(o), orNone(n)))
	}

	oldBy, newBy := index(oldA), index(newA)
	// A resource renamed with a moved block is the same resource: the old
	// plan listed it under its old address, the new one under the new address
	// with moved_from. Compare them as one instead of removed plus added.
	for addr, n := range newBy {
		if x := n.MovedFrom; x != "" && oldBy[x] != nil && newBy[x] == nil && oldBy[addr] == nil {
			oldBy[addr] = oldBy[x]
			delete(oldBy, x)
		}
	}
	addrs := map[string]bool{}
	for a := range oldBy {
		addrs[a] = true
	}
	for a := range newBy {
		addrs[a] = true
	}
	sorted := make([]string, 0, len(addrs))
	for a := range addrs {
		sorted = append(sorted, a)
	}
	sort.Strings(sorted)

	for _, addr := range sorted {
		o, n := oldBy[addr], newBy[addr]
		switch {
		case o == nil && n == nil:
		case o == nil:
			e := Entry{Address: addr, Kind: Added, NewAction: n.Action, Differences: []string{}, Danger: destructive(n.Action)}
			r.Added = append(r.Added, e)
		case n == nil:
			e := Entry{Address: addr, Kind: Removed, OldAction: o.Action, Differences: []string{}, Defused: destructive(o.Action)}
			r.Removed = append(r.Removed, e)
		default:
			d := differences(o, n)
			if len(d) == 0 {
				r.Unchanged++
				continue
			}
			r.Changed = append(r.Changed, Entry{
				Address: addr, Kind: Changed, OldAction: o.Action, NewAction: n.Action, Differences: d,
				Danger:  destructive(n.Action) && !destructive(o.Action),
				Defused: destructive(o.Action) && !destructive(n.Action),
			})
		}
	}
	for _, list := range [][]Entry{r.Added, r.Changed} {
		for _, e := range list {
			if e.Danger {
				r.NewDanger = append(r.NewDanger, e.Address)
			}
		}
	}
	for _, list := range [][]Entry{r.Removed, r.Changed} {
		for _, e := range list {
			if e.Defused {
				r.Defused = append(r.Defused, e.Address)
			}
		}
	}
	sort.Strings(r.NewDanger)
	sort.Strings(r.Defused)

	oldDrift, newDrift := driftSet(oldA), driftSet(newA)
	for a := range newDrift {
		if !oldDrift[a] {
			r.DriftNew = append(r.DriftNew, a)
		}
	}
	for a := range oldDrift {
		if !newDrift[a] {
			r.DriftGone = append(r.DriftGone, a)
		}
	}
	sort.Strings(r.DriftNew)
	sort.Strings(r.DriftGone)
	return r
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// index maps address to the change, leaving out resources that are not
// changing (plain no-ops).
func index(a *analyze.Analysis) map[string]*analyze.Change {
	m := map[string]*analyze.Change{}
	for i := range a.Changes {
		c := &a.Changes[i]
		if c.Listed() {
			m[c.Address] = c
		}
	}
	return m
}

func driftSet(a *analyze.Analysis) map[string]bool {
	m := map[string]bool{}
	for _, d := range a.Drift {
		m[d.Address] = true
	}
	return m
}

// differences lists what differs between two changes to the same resource.
func differences(o, n *analyze.Change) []string {
	d := []string{}
	if o.Action != n.Action {
		d = append(d, fmt.Sprintf("action: %s -> %s", o.Action, n.Action))
	}
	if o.Reason != n.Reason {
		d = append(d, fmt.Sprintf("reason: %s -> %s", orNone(o.Reason), orNone(n.Reason)))
	}
	if o.Account != n.Account {
		d = append(d, fmt.Sprintf("target account: %s -> %s", o.Account, n.Account))
	}
	if o.MovedFrom != n.MovedFrom {
		d = append(d, fmt.Sprintf("moved from: %s -> %s", orNone(o.MovedFrom), orNone(n.MovedFrom)))
	}
	if o.Import != n.Import {
		d = append(d, fmt.Sprintf("import: %v -> %v", o.Import, n.Import))
	}
	oa, na := attrs(o), attrs(n)
	paths := map[string]bool{}
	for p := range oa {
		paths[p] = true
	}
	for p := range na {
		paths[p] = true
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)
	for _, p := range sorted {
		x, y := oa[p], na[p]
		switch {
		case x == nil:
			d = append(d, "attribute now changes: "+p)
		case y == nil:
			d = append(d, "attribute no longer changes: "+p)
		case x.Kind != y.Kind:
			d = append(d, fmt.Sprintf("attribute %s: %s -> %s", p, x.Kind, y.Kind))
		case x.ForcesReplace != y.ForcesReplace:
			d = append(d, fmt.Sprintf("attribute %s: forces replacement %v -> %v", p, x.ForcesReplace, y.ForcesReplace))
		case x.Kind == "sensitive":
			// Values are never known, so two sensitive changes are equal.
		case !bytes.Equal(x.Before, y.Before) || !bytes.Equal(x.After, y.After) || diffLen(x) != diffLen(y):
			d = append(d, fmt.Sprintf("attribute %s: was %s, now %s", p, show(x), show(y)))
		}
	}
	return d
}

func diffLen(a *analyze.Attr) int {
	if a.Diff == nil {
		return 0
	}
	return a.Diff.Added*100000 + a.Diff.Removed
}

func attrs(c *analyze.Change) map[string]*analyze.Attr {
	m := map[string]*analyze.Attr{}
	for i := range c.Attrs {
		m[c.Attrs[i].Path] = &c.Attrs[i]
	}
	return m
}

// show is the "before -> after" of an attribute, kept short.
func show(a *analyze.Attr) string {
	if a.Kind == "long" && a.Diff != nil {
		return fmt.Sprintf("+%d/-%d lines", a.Diff.Added, a.Diff.Removed)
	}
	return trunc(string(a.Before)) + " => " + trunc(string(a.After))
}

func trunc(s string) string {
	if s == "" {
		return "(absent)"
	}
	if len(s) > 40 {
		return s[:37] + "..."
	}
	return s
}
