package analyze

import (
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// Parent groups the instances of one count/for_each resource.
type Parent struct {
	Address   string       `json:"address"` // configuration address, no instance keys
	Kind      string       `json:"kind"`    // count | for_each
	Counts    ActionCounts `json:"counts"`
	Instances []Instance   `json:"instances"`
}

// Instance is one key of a Parent.
type Instance struct {
	Key     string `json:"key"`
	Address string `json:"address"`
	Action  string `json:"action"`
}

// buildParents groups instances of the same resource, in plan order. A
// resource with a single instance (count = 1) is not grouped.
func buildParents(visible []plan.Change) []Parent {
	idx := map[string]int{}
	var parents []Parent
	for _, c := range visible {
		if !c.HasIndex {
			continue
		}
		i, ok := idx[c.ConfigAddress]
		if !ok {
			kind := "for_each"
			if isCountKey(c.Index) {
				kind = "count"
			}
			parents = append(parents, Parent{Address: c.ConfigAddress, Kind: kind, Instances: []Instance{}})
			i = len(parents) - 1
			idx[c.ConfigAddress] = i
		}
		parents[i].Instances = append(parents[i].Instances, Instance{Key: c.Index, Address: c.Address, Action: string(c.Action)})
		count(&parents[i].Counts, c.Action)
	}
	out := parents[:0]
	for _, p := range parents {
		if len(p.Instances) > 1 {
			out = append(out, p)
		}
	}
	return out
}

func isCountKey(k string) bool {
	return k != "" && strings.Trim(k, "0123456789") == ""
}
