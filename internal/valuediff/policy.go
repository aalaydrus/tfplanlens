package valuediff

import (
	"encoding/json"
	"sort"
	"strings"
)

// PolicyStatements renders an IAM policy document as one line per statement,
// `Effect Action(s) -> Resource(s)`. It returns nil if doc is not a policy.
// Trust policies have no Resource; they show the principal instead.
func PolicyStatements(doc string) []string {
	var p struct {
		Statement json.RawMessage `json:"Statement"`
	}
	if json.Unmarshal([]byte(doc), &p) != nil || len(p.Statement) == 0 {
		return nil
	}
	var stmts []map[string]json.RawMessage
	if json.Unmarshal(p.Statement, &stmts) != nil {
		var one map[string]json.RawMessage
		if json.Unmarshal(p.Statement, &one) != nil {
			return nil
		}
		stmts = []map[string]json.RawMessage{one}
	}
	out := make([]string, 0, len(stmts))
	for _, s := range stmts {
		out = append(out, statementLine(s))
	}
	return out
}

func statementLine(s map[string]json.RawMessage) string {
	effect := firstString(s["Effect"])
	actions := listField(s, "Action", "NotAction")
	target := listField(s, "Resource", "NotResource")
	if len(target) == 0 {
		if pr := principals(s["Principal"]); len(pr) > 0 {
			target = []string{"principal " + strings.Join(pr, ",")}
		} else {
			target = []string{"(none)"}
		}
	}
	return strings.TrimSpace(effect + " " + strings.Join(actions, ",") + " -> " + strings.Join(target, ","))
}

func listField(s map[string]json.RawMessage, key, notKey string) []string {
	prefix := ""
	raw, ok := s[key]
	if !ok {
		if raw, ok = s[notKey]; !ok {
			return nil
		}
		prefix = notKey + ":"
	}
	vals := stringList(raw)
	sort.Strings(vals)
	if prefix != "" && len(vals) > 0 {
		vals[0] = prefix + vals[0]
	}
	return vals
}

func principals(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	if s := firstString(raw); s != "" {
		return []string{s}
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	var out []string
	for k, v := range m {
		vals := stringList(v)
		sort.Strings(vals)
		out = append(out, k+"="+strings.Join(vals, "|"))
	}
	sort.Strings(out)
	return out
}

func firstString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return ""
}

func stringList(raw json.RawMessage) []string {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		return many
	}
	return nil
}

// statementDiff lists statements removed (-) and added (+); unchanged
// statements are omitted.
func statementDiff(before, after []string) []string {
	count := func(xs []string) map[string]int {
		m := map[string]int{}
		for _, x := range xs {
			m[x]++
		}
		return m
	}
	bc, ac := count(before), count(after)
	out := []string{}
	for _, s := range before {
		if ac[s] > 0 {
			ac[s]--
			continue
		}
		out = append(out, "-"+s)
	}
	for _, s := range after {
		if bc[s] > 0 {
			bc[s]--
			continue
		}
		out = append(out, "+"+s)
	}
	if len(out) == 0 {
		out = append(out, " (statements unchanged)")
	}
	return out
}
