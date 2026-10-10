package analyze

import (
	"fmt"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// Rule is a --fail-on rule: delete, replace or drift, optionally scoped to a
// resource type with "kind:type", or "unresolved" (an account the report could
// not determine).
type Rule struct {
	Kind string
	Type string
}

func (r Rule) String() string {
	if r.Type != "" {
		return r.Kind + ":" + r.Type
	}
	return r.Kind
}

// ParseRules parses a comma-separated --fail-on value.
func ParseRules(s string) ([]Rule, error) {
	var rules []Rule
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kind, typ, _ := strings.Cut(part, ":")
		switch kind {
		case "delete", "replace", "drift":
		case "unresolved":
			if typ != "" || strings.Contains(part, ":") {
				return nil, fmt.Errorf("--fail-on rule %q takes no type", part)
			}
		default:
			return nil, fmt.Errorf("unknown --fail-on rule %q (want delete, replace, drift or unresolved; the first three take an optional :type)", part)
		}
		if strings.Contains(part, ":") && typ == "" {
			return nil, fmt.Errorf("--fail-on rule %q has an empty type", part)
		}
		rules = append(rules, Rule{Kind: kind, Type: typ})
	}
	return rules, nil
}

// Match is a plan change that satisfied a rule.
type Match struct {
	Rule    Rule
	Address string
}

// CheckFailOn evaluates rules against the whole plan. It deliberately ignores
// ignore_types: hiding a type from the report never hides it from CI.
// "delete" matches pure deletes only; replacements are "replace".
// "unresolved" matches the apply account and every provider group whose target
// account could not be determined; it needs the analysis.
func CheckFailOn(p *plan.Plan, a *Analysis, rules []Rule) []Match {
	var out []Match
	for _, r := range rules {
		if r.Kind == "unresolved" {
			if !a.ApplyAccount.Resolved {
				out = append(out, Match{Rule: r, Address: "apply account"})
			}
			for _, g := range a.Groups {
				if !g.Resolved {
					out = append(out, Match{Rule: r, Address: "target account of " + g.Key})
				}
			}
			continue
		}
		list := p.Changes
		if r.Kind == "drift" {
			list = p.Drift
		}
		for _, c := range list {
			if r.Type != "" && c.Type != r.Type {
				continue
			}
			ok := false
			switch r.Kind {
			case "delete":
				ok = c.Action == plan.ActionDelete
			case "replace":
				ok = c.Action == plan.ActionReplace
			case "drift":
				ok = true
			}
			if ok {
				out = append(out, Match{Rule: r, Address: c.Address})
			}
		}
	}
	return out
}
