package analyze

import (
	"fmt"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// Rule is a --fail-on rule: delete, replace or drift, optionally scoped to a
// resource type with "kind:type".
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
		default:
			return nil, fmt.Errorf("unknown --fail-on rule %q (want delete, replace or drift, optionally kind:type)", part)
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
func CheckFailOn(p *plan.Plan, rules []Rule) []Match {
	var out []Match
	for _, r := range rules {
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
