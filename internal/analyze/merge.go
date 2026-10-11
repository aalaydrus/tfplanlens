package analyze

import (
	"fmt"
	"sort"
	"strings"
)

// Prefix puts a unit label in front of an address in a merged report.
func Prefix(unit, s string) string {
	if s == "" {
		return s
	}
	return unit + ":" + s
}

// Merge combines the analyses of several plans (the root modules of a
// monorepo, say) into one report. Each plan was analyzed on its
// own, so accounts, waves and dependencies are right per plan; Merge only
// prefixes addresses with the unit label ("vpc:aws_vpc.main") so identical
// addresses in different plans stay apart, and adds the counts together.
// labels must be unique and the same length as as.
func Merge(labels []string, as []*Analysis) *Analysis {
	m := &Analysis{
		SchemaVersion: as[0].SchemaVersion,
		Notices:       []string{},
		Findings:      []Finding{},
		Drift:         []DriftItem{},
		Groups:        []Group{},
		Changes:       []Change{},
		Parents:       []Parent{},
		Waves:         []Wave{},
		WaveNotes:     []string{},
		Cycles:        []Cycle{},
		Units:         []Unit{},
	}
	m.Summary = Summary{Applyable: true, ByType: map[string]int{}, ByModule: map[string]int{}}

	modes, versions, applyIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	waves := map[string]*Wave{}
	var order []string
	notes := map[string]bool{}

	for i, a := range as {
		l := labels[i]
		s := a.Summary
		m.Units = append(m.Units, Unit{Label: l, Mode: s.Mode, TerraformVersion: s.TerraformVersion, ApplyAccount: a.ApplyAccount, Actions: s.Actions})
		modes[s.Mode], versions[s.TerraformVersion], applyIDs[a.ApplyAccount.ID+"/"+a.ApplyAccount.Source] = true, true, true

		m.Summary.Applyable = m.Summary.Applyable && s.Applyable
		m.Summary.Errored = m.Summary.Errored || s.Errored
		m.Summary.Total += s.Total
		addCounts(&m.Summary.Actions, s.Actions)
		for k, v := range s.ByType {
			m.Summary.ByType[k] += v
		}
		for k, v := range s.ByModule {
			m.Summary.ByModule[Prefix(l, k)] += v
		}
		m.Summary.KnownAfterApply += s.KnownAfterApply
		m.Summary.DriftCount += s.DriftCount
		m.Summary.Warnings += s.Warnings
		m.Summary.Hidden += s.Hidden
		m.Summary.Redacted += s.Redacted
		m.Summary.ApplyWaves = max(m.Summary.ApplyWaves, s.ApplyWaves)
		m.Summary.DestroyWaves = max(m.Summary.DestroyWaves, s.DestroyWaves)

		for _, n := range a.Notices {
			m.Notices = append(m.Notices, l+": "+n)
		}
		for _, f := range a.Findings {
			f.Address, f.Unit = Prefix(l, f.Address), l
			m.Findings = append(m.Findings, f)
		}
		for _, d := range a.Drift {
			d.Address, d.Unit = Prefix(l, d.Address), l
			m.Drift = append(m.Drift, d)
		}
		for _, g := range a.Groups {
			g.Key = Prefix(l, g.Key)
			addrs := make([]string, len(g.Addresses))
			for j, ad := range g.Addresses {
				addrs[j] = Prefix(l, ad)
			}
			g.Addresses = addrs
			m.Groups = append(m.Groups, g)
		}
		for _, c := range a.Changes {
			c.Address, c.Module, c.Parent, c.Unit = Prefix(l, c.Address), Prefix(l, c.Module), Prefix(l, c.Parent), l
			if c.MovedFrom != "" {
				c.MovedFrom = Prefix(l, c.MovedFrom)
			}
			m.Changes = append(m.Changes, c)
		}
		for _, p := range a.Parents {
			p.Address = Prefix(l, p.Address)
			ins := make([]Instance, len(p.Instances))
			for j, in := range p.Instances {
				in.Address = Prefix(l, in.Address)
				ins[j] = in
			}
			p.Instances = ins
			m.Parents = append(m.Parents, p)
		}
		for _, w := range a.Waves {
			k := fmt.Sprintf("%s/%d", w.Phase, w.Number)
			mw := waves[k]
			if mw == nil {
				mw = &Wave{Phase: w.Phase, Number: w.Number, Resources: []WaveResource{}}
				waves[k] = mw
				order = append(order, k)
			}
			for _, r := range w.Resources {
				r.Address = Prefix(l, r.Address)
				mw.Resources = append(mw.Resources, r)
			}
		}
		for _, n := range a.WaveNotes {
			if !notes[n] {
				notes[n] = true
				m.WaveNotes = append(m.WaveNotes, n)
			}
		}
		for _, cy := range a.Cycles {
			rs := make([]string, len(cy.Resources))
			for j, r := range cy.Resources {
				rs[j] = Prefix(l, r)
			}
			m.Cycles = append(m.Cycles, Cycle{Resources: rs})
		}
	}

	m.Summary.Mode = joinKeys(modes, "mixed")
	m.Summary.TerraformVersion = strings.Join(nonEmptyKeys(versions), ", ")
	if len(applyIDs) == 1 {
		m.ApplyAccount = as[0].ApplyAccount
	} else {
		// Resolved, so --fail-on unresolved does not trip on the merge itself:
		// each unit's own account is in Units and in its own Notices.
		m.ApplyAccount = ApplyAccount{ID: "(differs per plan)", Source: SourceMixed, Resolved: true}
	}

	sort.SliceStable(order, func(i, j int) bool {
		a, b := waves[order[i]], waves[order[j]]
		if a.Phase != b.Phase {
			return a.Phase == PhaseApply
		}
		return a.Number < b.Number
	})
	for _, k := range order {
		m.Waves = append(m.Waves, *waves[k])
	}
	if len(m.Waves) > 0 {
		m.WaveNotes = append(m.WaveNotes, "Waves are computed inside each plan. Dependencies between plans (for example one root module reading another's outputs) are not in the plan JSON, so a wave can mix plans that really must run one after the other.")
	}
	return m
}

func addCounts(dst *ActionCounts, c ActionCounts) {
	dst.Create += c.Create
	dst.Update += c.Update
	dst.Replace += c.Replace
	dst.Delete += c.Delete
	dst.Read += c.Read
	dst.NoOp += c.NoOp
	dst.Forget += c.Forget
}

func nonEmptyKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// joinKeys is the single value of m, or mixed when there are several.
func joinKeys(m map[string]bool, mixed string) string {
	k := nonEmptyKeys(m)
	if len(k) == 1 {
		return k[0]
	}
	return mixed
}
