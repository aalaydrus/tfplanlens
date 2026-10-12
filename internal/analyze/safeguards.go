package analyze

import "github.com/aalaydrus/tfplanlens/internal/plan"

// Safeguard names and states. A safeguard is something already on a resource
// that decides what a delete does to it.
const (
	SafeguardDeletionProtection = "deletion_protection"
	SafeguardFinalSnapshot      = "final_snapshot"

	StateOn      = "on"
	StateOff     = "off"
	StateTaken   = "taken" // a final snapshot is taken before the delete
	StateNone    = "none"  // the delete skips the final snapshot
	StateUnknown = "unknown"
)

// Safeguard is one setting on a resource being deleted or replaced.
type Safeguard struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// protectionAttrs lists, per resource type, the attribute (in the "before"
// values) that holds each safeguard. Only types whose attribute names were
// checked are here; any other type shows nothing rather than a guess.
//
// aws_dynamodb_table: checked in a real plan (simulation/assets/input).
// aws_db_instance, aws_rds_cluster: names taken from the AWS provider's
// documentation, not yet seen in a real plan (see field-notes/WISHLIST.md).
var protectionAttrs = map[string]struct{ deletion, skipSnapshot string }{
	"aws_dynamodb_table": {deletion: "deletion_protection_enabled"},
	"aws_db_instance":    {deletion: "deletion_protection", skipSnapshot: "skip_final_snapshot"},
	"aws_rds_cluster":    {deletion: "deletion_protection", skipSnapshot: "skip_final_snapshot"},
}

// safeguardsFor reads the safeguards of a resource that is being deleted or
// replaced from its before values. It never returns nil (JSON arrays are []);
// a value that is missing, null or not a boolean is "unknown".
func safeguardsFor(c plan.Change) []Safeguard {
	out := []Safeguard{}
	attrs, ok := protectionAttrs[c.Type]
	if !ok {
		return out
	}
	before, _ := c.Before.(map[string]any)
	flag := func(name string) (bool, bool) {
		if name == "" {
			return false, false
		}
		v, ok := before[name].(bool)
		return v, ok
	}
	state := StateUnknown
	if v, ok := flag(attrs.deletion); ok {
		state = StateOff
		if v {
			state = StateOn
		}
	}
	out = append(out, Safeguard{SafeguardDeletionProtection, state})
	if attrs.skipSnapshot != "" {
		state = StateUnknown
		if v, ok := flag(attrs.skipSnapshot); ok {
			state = StateTaken
			if v {
				state = StateNone
			}
		}
		out = append(out, Safeguard{SafeguardFinalSnapshot, state})
	}
	return out
}

// SafeguardText describes a safeguard for a report line, or "" for one that
// needs no line.
func SafeguardText(s Safeguard) string {
	switch {
	case s.Name == SafeguardDeletionProtection && s.State == StateOn:
		return "deletion protection is ON: AWS is expected to refuse this delete, so the apply would fail"
	case s.Name == SafeguardDeletionProtection && s.State == StateOff:
		return "deletion protection is off"
	case s.Name == SafeguardDeletionProtection:
		return "deletion protection: unknown (not in the plan)"
	case s.Name == SafeguardFinalSnapshot && s.State == StateNone:
		return "no final snapshot (skip_final_snapshot is true)"
	case s.Name == SafeguardFinalSnapshot && s.State == StateTaken:
		return "a final snapshot is taken before the delete"
	case s.Name == SafeguardFinalSnapshot:
		return "final snapshot: unknown (not in the plan)"
	}
	return ""
}

// hasSafeguard reports whether the findings' safeguards include name in state.
func hasSafeguard(ss []Safeguard, name, state string) bool {
	for _, s := range ss {
		if s.Name == name && s.State == state {
			return true
		}
	}
	return false
}
