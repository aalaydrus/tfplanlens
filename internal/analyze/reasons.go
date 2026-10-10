package analyze

import (
	"fmt"
	"strings"
)

// ReasonText turns an action_reason into plain English (SPEC 5.2). Unknown
// reasons are shown raw; a missing reason is reported as such.
func ReasonText(reason, index string, hasIndex bool, replacePaths []string) string {
	switch reason {
	case "":
		return "(no reason reported)"
	case "delete_because_no_resource_config":
		return "Removed from config"
	case "delete_because_wrong_repetition":
		return "Resource switched between count/for_each/single"
	case "delete_because_count_index":
		if hasIndex {
			return fmt.Sprintf("count shrank; index %s no longer exists", index)
		}
		return "count shrank; index no longer exists"
	case "delete_because_each_key":
		if hasIndex {
			return fmt.Sprintf("for_each key removed: %s", index)
		}
		return "for_each key removed"
	case "delete_because_no_module":
		return "Whole module removed or its call changed"
	case "delete_because_no_move_target":
		return "Moved to an address that no longer exists"
	case "replace_because_cannot_update":
		if len(replacePaths) > 0 {
			return "Attribute change forces replacement: " + strings.Join(replacePaths, ", ")
		}
		return "Attribute change forces replacement"
	case "replace_because_tainted":
		return "Resource is tainted"
	case "replace_by_request":
		return "Replacement requested with -replace"
	case "replace_by_triggers":
		return "replace_triggered_by dependency changed"
	}
	return reason
}
