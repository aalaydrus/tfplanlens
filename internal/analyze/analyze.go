// Package analyze turns a normalized plan into an Analysis: summary, danger
// findings, drift, and account grouping. Renderers consume only Analysis.
package analyze

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/config"
	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// Options are inputs that do not come from the plan or the config file.
type Options struct {
	// ApplyAccount is the account terraform apply runs from; ApplySource says
	// where it came from (SourceFlag, SourceEnv, SourceConfig). Empty means
	// "not given": it is then inferred from the plan's default provider.
	ApplyAccount string
	ApplySource  string
	// AccountOverrides maps a provider alias or config key to an account ID
	// (--account alias=id). They beat every other source.
	AccountOverrides map[string]string
}

// Analyze builds the Analysis for p. cfg and opts may be nil.
func Analyze(p *plan.Plan, cfg *config.Config, opts *Options) *Analysis {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if opts == nil {
		opts = &Options{}
	}
	stateful := toSet(builtinStateful, cfg.StatefulTypes)
	ignored := toSet(cfg.IgnoreTypes)

	a := &Analysis{
		SchemaVersion: SchemaVersion,
		Findings:      []Finding{},
		Drift:         []DriftItem{},
		Groups:        []Group{},
		Changes:       []Change{},
		Notices:       []string{},
		Parents:       []Parent{},
		Waves:         []Wave{},
		WaveNotes:     []string{},
		Cycles:        []Cycle{},
	}
	a.ApplyAccount = ApplyAccount{Source: SourceUnresolved}
	switch {
	case opts.ApplyAccount != "":
		a.ApplyAccount = ApplyAccount{ID: opts.ApplyAccount, Source: opts.ApplySource, Resolved: true}
	default:
		if id, via := inferApplyAccount(p, cfg, opts); id != "" {
			a.ApplyAccount = ApplyAccount{ID: id, Source: SourceInferred, Via: via, Resolved: true}
		}
	}
	// From here on the effective apply account is the one in the analysis.
	eff := *opts
	eff.ApplyAccount = a.ApplyAccount.ID
	opts = &eff
	if !a.ApplyAccount.Resolved {
		a.Notices = append(a.Notices, "Apply account unknown: set tfapply_account_id in tfplanlens.config.yaml (it cannot be read from the plan). Cross-account flags are off.")
	}
	switch p.Mode {
	case plan.ModeRefreshOnly:
		a.Notices = append(a.Notices, "Refresh-only plan: applying only updates state to match what exists; it creates, changes and destroys nothing.")
	case plan.ModeDestroy:
		a.Notices = append(a.Notices, "Destroy plan: applying destroys every resource listed under Danger.")
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
	waves, cycles, waveNotes, assigned := computeWaves(p)
	waves, renumber := hideIgnoredWaves(waves, ignored)
	a.Waves, a.Cycles = waves, cycles
	if len(cycles) == 0 {
		a.Cycles = []Cycle{}
	}
	if len(waves) > 0 && len(waveNotes) > 0 {
		a.WaveNotes = waveNotes
	}
	for _, w := range waves {
		if w.Phase == PhaseApply {
			a.Summary.ApplyWaves = w.Number
		} else {
			a.Summary.DestroyWaves = w.Number
		}
	}
	a.Parents = buildParents(visible)
	if a.Parents == nil {
		a.Parents = []Parent{}
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
	// Drift whose only difference is null vs an empty collection is the
	// provider reading back "{}" where state had null; Terraform's own plan
	// output does not report it either.
	realDrift := meaningfulDrift(p, cfg.LongValueKeys)
	driftAddr := map[string]bool{}
	for _, d := range realDrift {
		driftAddr[d.change.Address] = true
	}

	groups := map[string]*Group{}
	for _, c := range visible {
		acct := resolveAccount(p, cfg, opts, c.ProviderKey).crossFrom(opts.ApplyAccount)
		ch := Change{
			Address:       c.Address,
			Type:          c.Type,
			Module:        moduleName(c.ModuleAddress),
			Mode:          c.Mode,
			Action:        string(c.Action),
			Provider:      c.ProviderKey,
			Account:       acct.label,
			AccountSource: acct.source,
			CrossAccount:  acct.cross,
			Warnings:      []Warning{},
			Stateful:      stateful[c.Type],
			Reason:        c.Reason,
			ReplacePaths:  c.ReplacePaths,
			MovedFrom:     c.PreviousAddress,
			Import:        c.Importing,
			ImportID:      c.ImportID,
			StateOnly:     c.Action == plan.ActionForget,
			Attrs:         []Attr{},
		}
		if c.Action == plan.ActionDelete || c.Action == plan.ActionReplace {
			ch.ReasonText = ReasonText(c.Reason, c.Index, c.HasIndex, c.ReplacePaths)
			if c.Reason == "" && p.Mode == plan.ModeDestroy && c.Action == plan.ActionDelete {
				// Terraform reports no action_reason on a destroy plan: say why.
				ch.ReasonText = "Destroy plan: every resource is being destroyed"
			}
		}
		ch.KnownAfterApply = countUnknown(c.AfterUnknown)
		a.Summary.KnownAfterApply += ch.KnownAfterApply
		if c.Action == plan.ActionUpdate || c.Action == plan.ActionReplace {
			ch.Attrs = diffAttrs(c.Before, c.After, c.BeforeSensitive, c.AfterSensitive, c.AfterUnknown, c.ReplacePaths, cfg.LongValueKeys)
		}
		if active(c) && acct.resolved {
			if w := arnWarnings(c, acct); len(w) > 0 { // never nil: JSON arrays are [] not null
				ch.Warnings = w
				a.Summary.Warnings += len(w)
			}
		}
		if c.HasIndex {
			ch.Parent, ch.Key = c.ConfigAddress, c.Index
		}
		// Only instances that do something are in a wave: a no-op sibling
		// of a changing instance is not applied.
		if as, ok := assigned[c.ConfigAddress]; ok && c.Action != "no-op" && c.Action != "forget" {
			if n, ok := renumber[as.phase][as.wave]; ok {
				ch.Wave, ch.WavePhase = n, as.phase
			}
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
				Provider: c.ProviderKey, Account: acct.label, AccountSource: acct.source, Stateful: ch.Stateful,
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

	for _, rd := range realDrift {
		d := rd.change
		if ignored[d.Type] {
			continue
		}
		item := DriftItem{
			Address: d.Address, Type: d.Type, Action: string(d.Action),
			Attrs: rd.attrs, HighRisk: replaced[d.Address],
		}
		if d.Action == plan.ActionDelete {
			item.Note = "Deleted outside Terraform"
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
	if c.Reason == "delete_because_each_key" {
		// for_each keys do not shift; a removed key next to a new one is more
		// likely a rename that will destroy and recreate the resource.
		return true, "for_each key removed while sibling instances change (possible key rename: " + strings.Join(shown, ", ") + more + ")"
	}
	return true, "likely count index shift: other instances of this resource are also changing (" + strings.Join(shown, ", ") + more + ")"
}

type account struct {
	provider, alias, label, source string
	resolved, cross                bool
}

// resolveAccount finds the target account for a provider config key
// (SPEC 5.5), recording which source decided it.
func resolveAccount(p *plan.Plan, cfg *config.Config, opts *Options, key string) account {
	name, alias, _ := strings.Cut(key, ".")
	acct := account{provider: name, alias: alias, source: SourceUnresolved}
	pr, known := p.Providers[key]
	if known {
		acct.provider, acct.alias = pr.Name, pr.Alias
	}
	names := []string{key}
	if acct.alias != "" {
		names = []string{key, acct.alias}
	} else {
		names = []string{key, "default"}
	}
	set := func(id, source string) account {
		acct.label, acct.resolved, acct.source = id, true, source
		return acct
	}
	lookupMap := func(m map[string]string) string {
		for _, n := range names {
			if id := m[n]; id != "" {
				return id
			}
		}
		return ""
	}
	switch {
	case lookupMap(opts.AccountOverrides) != "": // a. --account flag
		return set(lookupMap(opts.AccountOverrides), SourceFlag)
	case lookupMap(cfg.Accounts) != "": // b. accounts: mapping
		return set(lookupMap(cfg.Accounts), SourceConfig)
	}
	if id := ParseARNAccount(pr.AssumeRoleARN); id != "" { // c. role_arn
		if pr.RoleARNVia == plan.ViaVariable {
			return set(id, SourceRoleARNVariable)
		}
		return set(id, SourceRoleARN)
	}
	if len(pr.AllowedAccountIDs) == 1 && validAccountID(pr.AllowedAccountIDs[0]) { // d.
		return set(pr.AllowedAccountIDs[0], SourceAllowedAccountID)
	}
	if len(pr.CallerIdentityIDs) == 1 && validAccountID(pr.CallerIdentityIDs[0]) { // e.
		return set(pr.CallerIdentityIDs[0], SourceCallerIdentity)
	}
	if !pr.AssumeRole && opts.ApplyAccount != "" { // f. runs as the apply account
		return set(opts.ApplyAccount, SourceApplyAccount)
	}
	shown := acct.alias
	if shown == "" {
		shown = "default"
	}
	acct.label = "unresolved (alias: " + shown + ")" // g.
	return acct
}

// crossFrom sets the cross-account flag once the apply account is known.
func (a account) crossFrom(apply string) account {
	a.cross = a.resolved && apply != "" && a.label != apply
	return a
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

// inferApplyAccount assumes terraform runs as the default (unaliased)
// provider. Only a default provider WITHOUT assume_role counts: one that
// assumes a role says nothing about where the runner lives. The account comes
// from the usual sources (accounts: mapping, a single allowed_account_ids, an
// aws_caller_identity in prior_state). The backend is not in the plan JSON, so
// it cannot be consulted.
func inferApplyAccount(p *plan.Plan, cfg *config.Config, opts *Options) (id, via string) {
	keys := map[string]bool{}
	for k, pr := range p.Providers {
		if pr.Alias == "" && !strings.Contains(k, ".") && !pr.AssumeRole {
			keys[k] = true
		}
	}
	for _, c := range p.Changes {
		if !strings.Contains(c.ProviderKey, ".") && !p.Providers[c.ProviderKey].AssumeRole {
			keys[c.ProviderKey] = true
		}
	}
	ordered := sortedKeys(keys)
	// Prefer "aws" when several default providers exist.
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i] == "aws" && ordered[j] != "aws" })
	// A copy without an apply account, so "runs as the apply account" cannot
	// resolve itself.
	probe := &Options{AccountOverrides: opts.AccountOverrides}
	for _, k := range ordered {
		if acct := resolveAccount(p, cfg, probe, k); acct.resolved {
			return acct.label, acct.source
		}
	}
	return "", ""
}

type driftEntry struct {
	change plan.Change
	attrs  []Attr
}

// meaningfulDrift returns the plan's drift entries that a reviewer should care
// about: deletes always, and updates with at least one difference that is not
// merely null versus an empty map or list.
func meaningfulDrift(p *plan.Plan, longKeys []string) []driftEntry {
	var out []driftEntry
	for _, d := range p.Drift {
		if d.Action == plan.ActionDelete {
			out = append(out, driftEntry{change: d, attrs: []Attr{}})
			continue
		}
		attrs := []Attr{}
		for _, at := range diffAttrs(d.Before, d.After, d.BeforeSensitive, d.AfterSensitive, d.AfterUnknown, nil, longKeys) {
			if at.Kind == "value" && nullish(at.Before) && nullish(at.After) {
				continue
			}
			attrs = append(attrs, at)
		}
		if len(attrs) > 0 {
			out = append(out, driftEntry{change: d, attrs: attrs})
		}
	}
	return out
}

// nullish: absent, null, {} or [].
func nullish(raw json.RawMessage) bool {
	switch strings.TrimSpace(string(raw)) {
	case "", "null", "{}", "[]":
		return true
	}
	return false
}

// hideIgnoredWaves drops resources of ignored types from the waves, removes
// waves that become empty and renumbers each phase consecutively. renumber maps
// a computed wave number to the displayed one.
func hideIgnoredWaves(waves []Wave, ignored map[string]bool) ([]Wave, map[string]map[int]int) {
	renumber := map[string]map[int]int{PhaseApply: {}, PhaseDestroy: {}}
	out := []Wave{}
	next := map[string]int{}
	for _, w := range waves {
		var rs []WaveResource
		for _, r := range w.Resources {
			if !ignored[typeOfConfigAddress(r.Address)] {
				rs = append(rs, r)
			}
		}
		if len(rs) == 0 {
			continue
		}
		next[w.Phase]++
		renumber[w.Phase][w.Number] = next[w.Phase]
		out = append(out, Wave{Phase: w.Phase, Number: next[w.Phase], Resources: rs})
	}
	return out, renumber
}

// typeOfConfigAddress is the resource type of a configuration address such as
// "module.a.data.aws_x.y" or "module.a.aws_x.y".
func typeOfConfigAddress(addr string) string {
	parts := strings.Split(addr, ".")
	for i := 0; i < len(parts); i++ {
		if parts[i] == "module" {
			i++ // skip the module name
			continue
		}
		if parts[i] == "data" && i+1 < len(parts) {
			return parts[i+1]
		}
		return parts[i]
	}
	return ""
}
