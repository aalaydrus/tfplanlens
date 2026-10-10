// Package analyze turns a normalized plan into an Analysis: summary, danger
// findings, drift, and account grouping. Renderers consume only Analysis.
package analyze

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/config"
	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// Analyze builds the Analysis for p. cfg may be nil.
func Analyze(p *plan.Plan, cfg *config.Config) *Analysis {
	if cfg == nil {
		cfg = &config.Config{}
	}
	stateful := toSet(builtinStateful, cfg.StatefulTypes)
	ignored := toSet(cfg.IgnoreTypes)

	a := &Analysis{
		SchemaVersion: SchemaVersion,
		Findings:      []Finding{},
		Drift:         []DriftItem{},
		Groups:        []Group{},
		Changes:       []Change{},
	}
	a.Summary = Summary{
		Mode:             string(p.Mode),
		Applyable:        p.Applyable,
		Errored:          p.Errored,
		TerraformVersion: p.TerraformVersion,
		ByType:           map[string]int{},
		ByModule:         map[string]int{},
	}

	var visible []plan.Change
	for _, c := range p.Changes {
		if ignored[c.Type] {
			a.Summary.Hidden++
			continue
		}
		visible = append(visible, c)
	}
	siblings := map[string][]plan.Change{}
	for _, c := range visible {
		k := c.ModuleAddress + "|" + c.Mode + "|" + c.Type + "." + c.Name
		siblings[k] = append(siblings[k], c)
	}
	replaced := map[string]bool{}
	for _, c := range p.Changes {
		if c.Action == plan.ActionReplace || c.Action == plan.ActionDelete {
			replaced[c.Address] = true
		}
	}
	driftAddr := map[string]bool{}
	for _, d := range p.Drift {
		driftAddr[d.Address] = true
	}

	groups := map[string]*Group{}
	for _, c := range visible {
		acct := resolveAccount(p, cfg, c.ProviderKey)
		ch := Change{
			Address:      c.Address,
			Type:         c.Type,
			Module:       moduleName(c.ModuleAddress),
			Mode:         c.Mode,
			Action:       string(c.Action),
			Provider:     c.ProviderKey,
			Account:      acct.label,
			CrossAccount: acct.cross,
			Stateful:     stateful[c.Type],
			Reason:       c.Reason,
			ReplacePaths: c.ReplacePaths,
			MovedFrom:    c.PreviousAddress,
			Import:       c.Importing,
			ImportID:     c.ImportID,
			StateOnly:    c.Action == plan.ActionForget,
			Attrs:        []Attr{},
		}
		if c.Action == plan.ActionDelete || c.Action == plan.ActionReplace {
			ch.ReasonText = ReasonText(c.Reason, c.Index, c.HasIndex, c.ReplacePaths)
		}
		ch.KnownAfterApply = countUnknown(c.AfterUnknown)
		a.Summary.KnownAfterApply += ch.KnownAfterApply
		if c.Action == plan.ActionUpdate || c.Action == plan.ActionReplace {
			ch.Attrs = diffAttrs(c.Before, c.After, c.BeforeSensitive, c.AfterSensitive, c.AfterUnknown, c.ReplacePaths, cfg.LongValueKeys)
		}
		a.Changes = append(a.Changes, ch)

		count(&a.Summary.Actions, c.Action)
		a.Summary.Total++
		if active(c) {
			a.Summary.ByType[c.Type]++
			a.Summary.ByModule[ch.Module]++
		}

		g := groups[c.ProviderKey]
		if g == nil {
			g = &Group{
				Key: c.ProviderKey, Provider: acct.provider, Alias: acct.alias,
				Account: acct.label, Resolved: acct.resolved, Source: acct.source,
				Default: acct.alias == "", Cross: acct.cross, Addresses: []string{},
			}
			groups[c.ProviderKey] = g
		}
		count(&g.Counts, c.Action)
		g.Addresses = append(g.Addresses, c.Address)

		if c.Action == plan.ActionDelete || c.Action == plan.ActionReplace {
			f := Finding{
				Address: c.Address, Type: c.Type, Action: string(c.Action),
				Provider: c.ProviderKey, Account: acct.label, Stateful: ch.Stateful,
				Reason: c.Reason, ReasonText: ch.ReasonText, ReplacePaths: c.ReplacePaths,
				DriftAlso: driftAddr[c.Address],
			}
			f.IndexShift, f.Warning = indexShift(c, siblings)
			a.Findings = append(a.Findings, f)
		}
	}
	for _, k := range sortedKeys(groups) {
		a.Groups = append(a.Groups, *groups[k])
	}

	for _, d := range p.Drift {
		if ignored[d.Type] {
			continue
		}
		item := DriftItem{
			Address: d.Address, Type: d.Type, Action: string(d.Action),
			Attrs: []Attr{}, HighRisk: replaced[d.Address],
		}
		if d.Action == plan.ActionDelete {
			item.Note = "Deleted outside Terraform"
		} else {
			item.Attrs = diffAttrs(d.Before, d.After, d.BeforeSensitive, d.AfterSensitive, d.AfterUnknown, nil, cfg.LongValueKeys)
		}
		if item.HighRisk {
			item.Note = strings.TrimSpace(item.Note + " Also being replaced or deleted by this plan.")
		}
		a.Drift = append(a.Drift, item)
	}
	a.Summary.DriftCount = len(a.Drift)
	return a
}

// active reports whether a change does something worth listing.
func active(c plan.Change) bool {
	return c.Action != plan.ActionNoOp || c.PreviousAddress != "" || c.Importing
}

func moduleName(addr string) string {
	if addr == "" {
		return "root"
	}
	return addr
}

func count(ac *ActionCounts, a plan.Action) {
	switch a {
	case plan.ActionCreate:
		ac.Create++
	case plan.ActionUpdate:
		ac.Update++
	case plan.ActionReplace:
		ac.Replace++
	case plan.ActionDelete:
		ac.Delete++
	case plan.ActionRead:
		ac.Read++
	case plan.ActionForget:
		ac.Forget++
	default:
		ac.NoOp++
	}
}

// indexShift flags a delete of one instance of a count/for_each resource
// while sibling instances are being created, updated or replaced: the classic
// "list changed and indexes moved" accident.
func indexShift(c plan.Change, siblings map[string][]plan.Change) (bool, string) {
	if c.Action != plan.ActionDelete {
		return false, ""
	}
	if c.Reason == "delete_because_wrong_repetition" {
		return true, "count/for_each was added or removed on this resource; instances are being destroyed rather than moved"
	}
	if !c.HasIndex {
		return false, ""
	}
	k := c.ModuleAddress + "|" + c.Mode + "|" + c.Type + "." + c.Name
	var changing []string
	for _, s := range siblings[k] {
		if s.Address == c.Address {
			continue
		}
		switch s.Action {
		case plan.ActionCreate, plan.ActionUpdate, plan.ActionReplace:
			changing = append(changing, s.Address)
		}
	}
	if len(changing) == 0 {
		return false, ""
	}
	shown := changing
	if len(shown) > 3 {
		shown = shown[:3]
	}
	more := ""
	if len(changing) > len(shown) {
		more = fmt.Sprintf(", +%d more", len(changing)-len(shown))
	}
	return true, "likely count/for_each index shift: sibling instances also changing (" + strings.Join(shown, ", ") + more + ")"
}

type account struct {
	provider, alias, label, source string
	resolved, cross                bool
}

// resolveAccount applies SPEC 5.5: literal assume_role ARN, then the config
// accounts mapping, then an "unresolved" label.
func resolveAccount(p *plan.Plan, cfg *config.Config, key string) account {
	name, alias, _ := strings.Cut(key, ".")
	acct := lookup(p, cfg, key, name, alias)
	if alias != "" {
		if def := lookup(p, cfg, name, name, ""); def.label != acct.label {
			acct.cross = true
		}
	}
	return acct
}

func lookup(p *plan.Plan, cfg *config.Config, key, name, alias string) account {
	acct := account{provider: name, alias: alias, source: "none"}
	if pr, ok := p.Providers[key]; ok {
		acct.provider, acct.alias = pr.Name, pr.Alias
		if id := ParseARNAccount(pr.AssumeRoleARN); id != "" {
			acct.label, acct.resolved, acct.source = id, true, "role_arn"
			return acct
		}
	}
	for _, k := range []string{acct.alias, key} {
		if id := cfg.Accounts[k]; k != "" && id != "" {
			acct.label, acct.resolved, acct.source = id, true, "config"
			return acct
		}
	}
	shown := acct.alias
	if shown == "" {
		shown = "default"
	}
	acct.label = "unresolved (alias: " + shown + ")"
	return acct
}

func toSet(lists ...[]string) map[string]bool {
	m := map[string]bool{}
	for _, l := range lists {
		for _, s := range l {
			m[s] = true
		}
	}
	return m
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
