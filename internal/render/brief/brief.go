// Package brief renders the short, ranked list of what needs attention in a
// plan: for a reviewer in a hurry or a code review agent that should not spend
// tokens on the rest. Plain text, no color.
package brief

import (
	"fmt"
	"io"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

// Render writes the first top items (0 for all) of the brief.
func Render(w io.Writer, a *analyze.Analysis, top int) error {
	b := analyze.BriefOf(a, top)
	var sb strings.Builder

	s := a.Summary
	sb.WriteString("tfplanlens brief | plan: " + s.Mode)
	if !s.Applyable {
		sb.WriteString(" | not applyable")
	}
	if s.Errored {
		sb.WriteString(" | ERRORED")
	}
	if counts := actionCounts(s.Actions); counts != "" {
		sb.WriteString(" | " + counts)
	}
	if a.ApplyAccount.Resolved {
		sb.WriteString(" | apply account " + a.ApplyAccount.ID)
	}
	fmt.Fprintf(&sb, " | showing %d of %d needing attention\n", len(b.Items), b.Total)
	for _, n := range a.Notices {
		sb.WriteString("NOTE: " + n + "\n")
	}

	if len(b.Items) == 0 {
		sb.WriteString("Nothing needs attention: no deletes, replaces, warnings, or updates that other resources depend on.\n")
	}
	for _, it := range b.Items {
		badge := ""
		if it.Stateful {
			badge = " [STATEFUL]"
		}
		fmt.Fprintf(&sb, "%d. %s %s%s\n", it.Rank, strings.ToUpper(it.Action), it.Address, badge)
		for _, why := range it.Why {
			sb.WriteString("   - " + why + "\n")
		}
	}
	if b.Omitted > 0 {
		fmt.Fprintf(&sb, "%d more not shown (raise --top, or --top 0 for all)\n", b.Omitted)
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

func actionCounts(c analyze.ActionCounts) string {
	var parts []string
	for _, p := range []struct {
		name string
		n    int
	}{{"create", c.Create}, {"update", c.Update}, {"replace", c.Replace}, {"delete", c.Delete}, {"forget", c.Forget}} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", p.name, p.n))
		}
	}
	return strings.Join(parts, ", ")
}
