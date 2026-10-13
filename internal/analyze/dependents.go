package analyze

import (
	"fmt"
	"strings"
)

// maxDependentsShown is how many dependents a report names; the rest are counted.
const maxDependentsShown = 2

// DependentsText describes what depends on a change for a report line, or ""
// when nothing in the plan does. "N+" because the plan cannot show other
// stacks or dependencies that pass through a local value.
func DependentsText(dependents []string) string {
	n := len(dependents)
	if n == 0 {
		return ""
	}
	shown := dependents
	more := ""
	if n > maxDependentsShown {
		shown = dependents[:maxDependentsShown]
		more = fmt.Sprintf(", +%d more", n-maxDependentsShown)
	}
	return fmt.Sprintf("used by %d+ resources in this plan: %s%s", n, strings.Join(shown, ", "), more)
}
