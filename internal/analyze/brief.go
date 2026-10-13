package analyze

import (
	"fmt"
	"sort"
)

// BriefItem is one change worth a reviewer's attention, with why.
type BriefItem struct {
	Rank     int
	Address  string
	Action   string
	Stateful bool
	// Why lists, most important first, the plain-English reasons this change
	// matters. Never empty.
	Why []string
}

// Brief is the short list of what needs attention in a plan, ranked.
type Brief struct {
	Items []BriefItem // at most the requested number; never nil
	// Total is how many changes qualified; Omitted how many of those Items
	// leaves out.
	Total   int
	Omitted int
}

// BriefOf ranks what needs attention and keeps the first top (0 keeps all).
// First every delete and replace, in the order of Findings; then other
// changes that carry a warning or that other resources depend on, those with a
// warning first and then those with more dependents. count and for_each
// instances of one resource are one entry. Plain creates, and updates nothing
// depends on, are not listed: the summary counts them.
func BriefOf(a *Analysis, top int) Brief {
	b := Brief{Items: []BriefItem{}}
	var all []BriefItem
	for _, f := range a.Findings {
		why := []string{f.ReasonText}
		if d := DependentsText(f.Dependents); d != "" {
			why = append(why, d)
		}
		for _, s := range f.Safeguards {
			if t := SafeguardText(s); t != "" {
				why = append(why, t)
			}
		}
		if f.IndexShift {
			why = append(why, "WARNING: "+f.Warning)
		}
		if f.DriftAlso {
			why = append(why, "HIGH RISK: resource also drifted outside Terraform")
		}
		all = append(all, BriefItem{Address: f.Address, Action: f.Action, Stateful: f.Stateful, Why: why})
	}

	type rest struct {
		item       BriefItem
		warned     bool
		dependents int
	}
	var cands []Change
	perParent := map[string]int{}
	for _, c := range a.Changes {
		if !c.Listed() || c.Action == "delete" || c.Action == "replace" {
			continue
		}
		if len(c.Warnings) == 0 && (c.Action != "update" || len(c.Dependents) == 0) {
			continue
		}
		cands = append(cands, c)
		if c.Parent != "" {
			perParent[c.Parent]++
		}
	}
	var others []rest
	folded := map[string]bool{}
	for _, c := range cands {
		addr := c.Address
		why := []string{"in-place " + c.Action}
		if c.Parent != "" && perParent[c.Parent] > 1 {
			// Several instances of one resource are one entry.
			if folded[c.Parent] {
				continue
			}
			folded[c.Parent] = true
			addr = c.Parent
			why = []string{fmt.Sprintf("%d instances changed in place", perParent[c.Parent])}
		}
		if d := DependentsText(c.Dependents); d != "" {
			why = append(why, d)
		}
		for _, w := range c.Warnings {
			why = append(why, "WARNING: "+w.Message)
		}
		others = append(others, rest{
			item:       BriefItem{Address: addr, Action: c.Action, Stateful: c.Stateful, Why: why},
			warned:     len(c.Warnings) > 0,
			dependents: len(c.Dependents),
		})
	}
	sort.SliceStable(others, func(i, j int) bool {
		if others[i].warned != others[j].warned {
			return others[i].warned
		}
		return others[i].dependents > others[j].dependents
	})
	for _, o := range others {
		all = append(all, o.item)
	}

	b.Total = len(all)
	if top > 0 && len(all) > top {
		b.Omitted = len(all) - top
		all = all[:top]
	}
	for i := range all {
		all[i].Rank = i + 1
		b.Items = append(b.Items, all[i])
	}
	return b
}
