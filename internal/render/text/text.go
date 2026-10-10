// Package text renders an analyze.Analysis for terminals (SPEC 5.8).
package text

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

// MaxList is how many items a long list shows before "+N more".
const MaxList = 10

// Options control rendering.
type Options struct {
	Color   bool
	MaxLine int // 0 means no truncation
}

type renderer struct {
	b   strings.Builder
	opt Options
}

// Render writes the text report: Summary, Danger, Drift, Changes by account,
// Changes by module.
func Render(w io.Writer, a *analyze.Analysis, opt Options) error {
	r := &renderer{opt: opt}
	r.summary(a)
	r.danger(a)
	r.drift(a)
	r.byAccount(a)
	r.byModule(a)
	_, err := io.WriteString(w, r.b.String())
	return err
}

const (
	red    = "\x1b[31m"
	yellow = "\x1b[33m"
	bold   = "\x1b[1m"
)

func (r *renderer) c(color, s string) string {
	if !r.opt.Color {
		return s
	}
	return color + s + "\x1b[0m"
}

// line writes one line, truncated to MaxLine runes.
func (r *renderer) line(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	if r.opt.MaxLine > 0 {
		if rs := []rune(s); len(rs) > r.opt.MaxLine && !strings.Contains(s, "\x1b") {
			s = string(rs[:max(r.opt.MaxLine-1, 1)]) + "…"
		}
	}
	r.b.WriteString(s)
	r.b.WriteByte('\n')
}

func (r *renderer) heading(s string) {
	if r.b.Len() > 0 {
		r.b.WriteByte('\n')
	}
	r.line("%s", r.c(bold, "== "+s+" =="))
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func (r *renderer) summary(a *analyze.Analysis) {
	s := a.Summary
	r.heading("Summary")
	r.line("Plan: %s | applyable: %s | errored: %s", s.Mode, yesno(s.Applyable), yesno(s.Errored))
	ac := s.Actions
	r.line("Actions: create %d, update %d, replace %d, delete %d, read %d, no-op %d, forget %d",
		ac.Create, ac.Update, ac.Replace, ac.Delete, ac.Read, ac.NoOp, ac.Forget)
	r.line("Known after apply: %d attribute(s)", s.KnownAfterApply)
	if s.Hidden > 0 {
		r.line("Hidden by ignore_types: %d change(s)", s.Hidden)
	}
	r.counts("By type", s.ByType)
	r.counts("By module", s.ByModule)
}

func (r *renderer) counts(label string, m map[string]int) {
	if len(m) == 0 {
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] > m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", k, m[k]))
	}
	more := ""
	if len(parts) > MaxList {
		more = fmt.Sprintf(", +%d more", len(parts)-MaxList)
		parts = parts[:MaxList]
	}
	r.line("%s: %s%s", label, strings.Join(parts, ", "), more)
}

func (r *renderer) danger(a *analyze.Analysis) {
	r.heading("Danger")
	if len(a.Findings) == 0 {
		r.line("No deletes or replacements.")
		return
	}
	// Never collapsed: every delete and replace must be visible.
	for _, f := range a.Findings {
		badge := ""
		if f.Stateful {
			badge = " " + r.c(red+bold, "[STATEFUL]")
		}
		r.line("%s %s%s", r.c(red+bold, "! "+strings.ToUpper(f.Action)), f.Address, badge)
		r.line("    type: %s | account: %s", f.Type, f.Account)
		r.line("    why: %s", f.ReasonText)
		if f.IndexShift {
			r.line("    %s", r.c(red+bold, "WARNING: "+f.Warning))
		}
		if f.DriftAlso {
			r.line("    %s", r.c(red+bold, "HIGH RISK: resource also drifted outside Terraform"))
		}
	}
}

func (r *renderer) drift(a *analyze.Analysis) {
	r.heading("Drift")
	if len(a.Drift) == 0 {
		r.line("No changes made outside Terraform.")
		return
	}
	for i, d := range a.Drift {
		if i == MaxList {
			r.line("+%d more", len(a.Drift)-MaxList)
			break
		}
		risk := ""
		if d.HighRisk {
			risk = " " + r.c(red+bold, "[HIGH RISK]")
		}
		r.line("~ %s%s", d.Address, risk)
		if d.Note != "" {
			r.line("    %s", d.Note)
		}
		r.attrs(d.Attrs, "    ")
	}
}

func (r *renderer) byAccount(a *analyze.Analysis) {
	r.heading("Changes by account")
	if len(a.Groups) == 0 {
		r.line("No changes.")
		return
	}
	byAddr := map[string]*analyze.Change{}
	for i := range a.Changes {
		byAddr[a.Changes[i].Address] = &a.Changes[i]
	}
	for gi, g := range a.Groups {
		if gi > 0 {
			r.b.WriteByte('\n')
		}
		cross := ""
		if g.Cross {
			cross = " " + r.c(yellow+bold, "[DIFFERENT ACCOUNT than default provider]")
		}
		r.line("%s: %s%s", r.c(bold, g.Key), g.Account, cross)
		r.line("    %s", countsLine(g.Counts))
		n := 0
		for _, addr := range g.Addresses {
			c := byAddr[addr]
			if !listed(c) {
				continue
			}
			if n == MaxList {
				r.line("    +%d more", countListed(g, byAddr)-MaxList)
				break
			}
			n++
			r.change(c)
		}
	}
}

func countListed(g analyze.Group, by map[string]*analyze.Change) int {
	n := 0
	for _, a := range g.Addresses {
		if listed(by[a]) {
			n++
		}
	}
	return n
}

func listed(c *analyze.Change) bool {
	return c.Action != "no-op" || c.MovedFrom != "" || c.Import
}

func countsLine(c analyze.ActionCounts) string {
	var p []string
	add := func(n int, s string) {
		if n > 0 {
			p = append(p, fmt.Sprintf("%s %d", s, n))
		}
	}
	add(c.Create, "create")
	add(c.Update, "update")
	add(c.Replace, "replace")
	add(c.Delete, "delete")
	add(c.Read, "read")
	add(c.Forget, "forget")
	add(c.NoOp, "no-op")
	if len(p) == 0 {
		return "(nothing)"
	}
	return strings.Join(p, ", ")
}

var symbols = map[string]string{
	"create": "+", "update": "~", "replace": "-/+", "delete": "-",
	"read": "<=", "no-op": "=", "forget": "x",
}

func (r *renderer) change(c *analyze.Change) {
	label := c.Action
	switch {
	case c.StateOnly:
		label = "removed from state only"
	case c.Import && c.Action == "no-op":
		label = "import"
	case c.Import:
		label = "import + " + c.Action
	}
	if c.MovedFrom != "" {
		if c.Action == "no-op" {
			label = "moved from " + c.MovedFrom
		} else {
			label += ", moved from " + c.MovedFrom
		}
	}
	badge := ""
	if c.Stateful {
		badge = " [STATEFUL]"
	}
	r.line("    %s %s%s  (%s)", symbols[c.Action], c.Address, badge, label)
	r.attrs(c.Attrs, "        ")
}

func (r *renderer) attrs(attrs []analyze.Attr, indent string) {
	for i, at := range attrs {
		if i == MaxList {
			r.line("%s+%d more attribute(s)", indent, len(attrs)-MaxList)
			return
		}
		force := ""
		if at.ForcesReplace {
			force = "  [forces replacement]"
		}
		switch at.Kind {
		case "sensitive":
			r.line("%s%s: (sensitive value changed)%s", indent, at.Path, force)
		case "unknown":
			if len(at.Before) > 0 {
				r.line("%s%s: %s -> (known after apply)%s", indent, at.Path, string(at.Before), force)
			} else {
				r.line("%s%s: (known after apply)%s", indent, at.Path, force)
			}
		case "long":
			r.line("%s%s%s", indent, at.Diff.Header, force)
			for _, l := range at.Diff.Lines {
				r.line("%s  %s", indent, l)
			}
			if at.Diff.Truncated > 0 {
				r.line("%s  … %d more diff line(s) hidden", indent, at.Diff.Truncated)
			}
		default:
			r.line("%s%s: %s -> %s%s", indent, at.Path, orAbsent(at.Before), orAbsent(at.After), force)
		}
	}
}

func orAbsent(s json.RawMessage) string {
	if len(s) == 0 {
		return "(absent)"
	}
	return string(s)
}

func (r *renderer) byModule(a *analyze.Analysis) {
	r.heading("Changes by module")
	type mod struct {
		name string
		c    analyze.ActionCounts
	}
	idx := map[string]*mod{}
	var mods []*mod
	for _, ch := range a.Changes {
		if ch.Action == "no-op" && ch.MovedFrom == "" && !ch.Import {
			continue
		}
		m := idx[ch.Module]
		if m == nil {
			m = &mod{name: ch.Module}
			idx[ch.Module] = m
			mods = append(mods, m)
		}
		switch ch.Action {
		case "create":
			m.c.Create++
		case "update":
			m.c.Update++
		case "replace":
			m.c.Replace++
		case "delete":
			m.c.Delete++
		case "read":
			m.c.Read++
		case "forget":
			m.c.Forget++
		default:
			m.c.NoOp++
		}
	}
	if len(mods) == 0 {
		r.line("No changes.")
		return
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].name < mods[j].name })
	for i, m := range mods {
		if i == MaxList {
			r.line("+%d more", len(mods)-MaxList)
			break
		}
		r.line("%s: %s", m.name, countsLine(m.c))
	}
}
