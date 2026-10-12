// Package markdown renders an analyze.Analysis as GitHub-flavored markdown
// for PR comments (SPEC 5.8): a summary table, the Danger section expanded,
// and everything else inside <details> blocks per account, then module.
package markdown

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

// MaxChars is the hard cap on output size. GitHub's comment limit is 65,536
// characters; the margin leaves room for a wrapper that adds its own text.
// It is measured in bytes, which is never less than characters.
const MaxChars = 60000

// Marker is a hidden comment that lets a CI action find and update its own
// PR comment.
const Marker = "<!-- tfplanlens-report -->"

// reserve is kept free for closing tags and the truncation note.
const reserve = 600

type writer struct {
	b         strings.Builder
	truncated bool
	omitted   int // listed items that did not fit
}

// fits reports whether s can be added without breaking the cap.
func (w *writer) fits(s string) bool {
	return !w.truncated && w.b.Len()+len(s) <= MaxChars-reserve
}

// add writes s if it fits and otherwise marks the output truncated.
func (w *writer) add(s string) bool {
	if !w.fits(s) {
		w.truncated = true
		return false
	}
	w.b.WriteString(s)
	return true
}

func (w *writer) line(format string, args ...any) bool {
	return w.add(fmt.Sprintf(format, args...) + "\n")
}

// Render writes the report.
func Render(out io.Writer, a *analyze.Analysis) error {
	w := &writer{}
	w.add(Marker + "\n")
	w.line("## tfplanlens plan report")
	w.line("")
	summary(w, a)
	danger(w, a)
	drift(w, a)
	byAccount(w, a)
	byModule(w, a)
	waves(w, a)
	parents(w, a)
	if w.truncated {
		fmt.Fprintf(&w.b, "\n> **Report truncated** at %d characters (GitHub comments are limited to 65,536); %d item(s) not shown. Use `--format html` or `--format json` for the full report.\n", MaxChars, w.omitted)
	}
	_, err := io.WriteString(out, w.b.String())
	return err
}

// code wraps s in an inline code span that cannot be broken out of.
func code(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
	fence := strings.Repeat("`", longestRun(s, '`')+1)
	pad := ""
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		pad = " "
	}
	return fence + pad + s + pad + fence
}

// cell makes s safe inside a table cell.
func cell(s string) string {
	return strings.NewReplacer("|", `\|`, "\r", " ", "\n", " ").Replace(s)
}

// codeCell is a code span inside a table cell.
func codeCell(s string) string { return cell(code(s)) }

func longestRun(s string, c byte) int {
	best, cur := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	return best
}

// fenced wraps lines in a code fence longer than any backtick run inside.
func fenced(lang string, lines []string) string {
	body := strings.Join(lines, "\n")
	fence := strings.Repeat("`", max(3, longestRun(body, '`')+1))
	return fence + lang + "\n" + body + "\n" + fence + "\n"
}

func summary(w *writer, a *analyze.Analysis) {
	s := a.Summary
	w.line("| | |")
	w.line("|---|---|")
	w.line("| Plan | %s, applyable: %s, errored: %s |", s.Mode, yesno(s.Applyable), yesno(s.Errored))
	if aa := a.ApplyAccount; aa.Resolved {
		w.line("| Executed from (apply account) | %s (%s) |", code(aa.ID), aa.Describe())
	} else {
		w.line("| Executed from (apply account) | **unresolved** |")
	}
	for _, r := range []struct {
		name string
		n    int
	}{
		{"create", s.Actions.Create}, {"update", s.Actions.Update}, {"replace", s.Actions.Replace},
		{"delete", s.Actions.Delete}, {"read", s.Actions.Read}, {"no-op", s.Actions.NoOp}, {"forget", s.Actions.Forget},
	} {
		if r.n > 0 || r.name == "create" || r.name == "update" || r.name == "replace" || r.name == "delete" {
			w.line("| %s | %d |", r.name, r.n)
		}
	}
	w.line("| Drift | %d |", s.DriftCount)
	w.line("| Known after apply | %d attribute(s) |", s.KnownAfterApply)
	if s.Warnings > 0 {
		w.line("| Warnings | %d |", s.Warnings)
	}
	if s.Hidden > 0 {
		w.line("| Hidden by ignore_types | %d |", s.Hidden)
	}
	w.line("")
	units(w, a)
	for _, n := range a.Notices {
		w.line("> **Notice:** %s", cell(n))
		w.line("")
	}
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func danger(w *writer, a *analyze.Analysis) {
	w.line("### Danger")
	w.line("")
	if len(a.Findings) == 0 {
		w.line("No deletes or replacements.")
		w.line("")
		return
	}
	w.line("| Action | Resource | Why | Target account |")
	w.line("|---|---|---|---|")
	for i, f := range a.Findings {
		res := codeCell(f.Address)
		if f.Stateful {
			res += " **STATEFUL**"
		}
		why := cell(code(f.ReasonText))
		if f.IndexShift {
			why += "<br>**WARNING:** " + cell(code(f.Warning))
		}
		if d := analyze.DependentsText(f.Dependents); d != "" {
			why += "<br>" + cell(code(d))
		}
		if f.DriftAlso {
			why += "<br>**HIGH RISK:** also drifted outside Terraform"
		}
		if !w.line("| **%s** | %s | %s | %s (source: %s) |", strings.ToUpper(f.Action), res, why, codeCell(f.Account), f.AccountSource) {
			w.omitted += len(a.Findings) - i
			return
		}
	}
	w.line("")
}

func drift(w *writer, a *analyze.Analysis) {
	if len(a.Drift) == 0 {
		return
	}
	high := 0
	for _, d := range a.Drift {
		if d.HighRisk {
			high++
		}
	}
	open := ""
	if high > 0 {
		open = " open"
	}
	if !w.line("<details%s><summary><b>Drift</b>: %d changed outside Terraform (%d high risk)</summary>", open, len(a.Drift), high) {
		return
	}
	w.line("")
	for i, d := range a.Drift {
		head := "- " + code(d.Address)
		if d.HighRisk {
			head += " **HIGH RISK**"
		}
		if d.Note != "" {
			head += ": " + cell(d.Note)
		}
		var lines []string
		for _, at := range d.Attrs {
			lines = append(lines, attrLines(at, "  ")...)
		}
		block := head + "\n" + strings.Join(lines, "\n")
		if len(lines) > 0 {
			block += "\n"
		}
		if !w.add(block) {
			w.omitted += len(a.Drift) - i
			break
		}
	}
	w.closeDetails()
}

// closeDetails ends a <details> block; the reserve guarantees room for it.
func (w *writer) closeDetails() {
	w.b.WriteString("\n</details>\n\n")
}

// attrLines renders one changed attribute as list lines.
func attrLines(at analyze.Attr, indent string) []string {
	force := ""
	if at.ForcesReplace {
		force = " (forces replacement)"
	}
	switch at.Kind {
	case "sensitive":
		return []string{indent + "- " + code(at.Path) + ": (sensitive value changed)" + force}
	case "long":
		out := []string{indent + "- " + code(at.Diff.Header) + force}
		diff := append([]string{}, at.Diff.Lines...)
		if at.Diff.Truncated > 0 {
			diff = append(diff, fmt.Sprintf("… %d more diff line(s) hidden", at.Diff.Truncated))
		}
		for _, l := range strings.Split(strings.TrimRight(fenced("diff", diff), "\n"), "\n") {
			out = append(out, indent+"  "+l)
		}
		return out
	}
	return []string{indent + "- " + code(at.Path) + ": " + code(orAbsent(at.Before)) + " → " + code(orAbsent(at.After)) + force}
}

func orAbsent(b []byte) string {
	if len(b) == 0 {
		return "(absent)"
	}
	return string(b)
}

func byAccount(w *writer, a *analyze.Analysis) {
	if len(a.Groups) == 0 {
		return
	}
	w.line("### Changes by account")
	w.line("")
	byAddr := map[string]*analyze.Change{}
	for i := range a.Changes {
		byAddr[a.Changes[i].Address] = &a.Changes[i]
	}
	for _, g := range a.Groups {
		cross := ""
		if g.Cross {
			cross = " &mdash; <b>differs from apply account</b>"
		}
		head := fmt.Sprintf("<details><summary><b>%s</b> &mdash; target account %s (source: %s)%s &mdash; %s</summary>",
			htmlEscape(g.Key), htmlEscape(g.Account), htmlEscape(g.Source), cross, htmlEscape(countsLine(g.Counts)))
		if !w.line("%s", head) {
			return
		}
		w.line("")
		for i, addr := range g.Addresses {
			c := byAddr[addr]
			if !c.Listed() {
				continue
			}
			if !w.add(changeBlock(c)) {
				w.omitted += len(g.Addresses) - i
				break
			}
		}
		w.closeDetails()
		if w.truncated {
			return
		}
	}
}

func changeBlock(c *analyze.Change) string {
	var b strings.Builder
	badge := ""
	if c.Stateful {
		badge = " **STATEFUL**"
	}
	fmt.Fprintf(&b, "- %s%s (%s)\n", code(c.Address), badge, cell(c.Label()))
	for _, wn := range c.Warnings {
		fmt.Fprintf(&b, "  - **WARNING:** %s\n", code(wn.Message))
	}
	for _, at := range c.Attrs {
		for _, l := range attrLines(at, "  ") {
			b.WriteString(l + "\n")
		}
	}
	return b.String()
}

func byModule(w *writer, a *analyze.Analysis) {
	idx := map[string][]*analyze.Change{}
	for i := range a.Changes {
		c := &a.Changes[i]
		if c.Listed() {
			idx[c.Module] = append(idx[c.Module], c)
		}
	}
	if len(idx) == 0 {
		return
	}
	mods := make([]string, 0, len(idx))
	for m := range idx {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	w.line("### Changes by module")
	w.line("")
	for _, m := range mods {
		if !w.line("<details><summary><b>%s</b> &mdash; %d change(s)</summary>", htmlEscape(m), len(idx[m])) {
			return
		}
		w.line("")
		w.line("| Action | Resource | Target account |")
		w.line("|---|---|---|")
		for i, c := range idx[m] {
			if !w.line("| %s | %s | %s (%s) |", cell(c.Label()), codeCell(c.Address), codeCell(c.Account), c.AccountSource) {
				w.omitted += len(idx[m]) - i
				break
			}
		}
		w.closeDetails()
		if w.truncated {
			return
		}
	}
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
		return "no changes"
	}
	return strings.Join(p, ", ")
}

var htmlReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func htmlEscape(s string) string { return htmlReplacer.Replace(s) }

func waves(w *writer, a *analyze.Analysis) {
	if len(a.Cycles) > 0 {
		w.line("### Dependency cycles")
		w.line("")
		for _, cy := range a.Cycles {
			parts := make([]string, len(cy.Resources))
			for i, r := range cy.Resources {
				parts[i] = code(r)
			}
			if !w.line("- **cycle:** %s", strings.Join(parts, " &rarr; ")) {
				return
			}
		}
		w.line("")
	}
	if len(a.Waves) == 0 {
		return
	}
	w.line("### Waves")
	w.line("")
	w.line("Terraform applies in parallel: this shows possible parallelism, not a strict order.")
	w.line("")
	for _, wv := range a.Waves {
		if !w.line("<details><summary><b>%s wave %d</b> &mdash; %d resource(s)</summary>", wv.Phase, wv.Number, len(wv.Resources)) {
			return
		}
		w.line("")
		for i, r := range wv.Resources {
			extra := ""
			if r.Instances > 1 {
				extra = fmt.Sprintf(" (%d instances)", r.Instances)
			}
			if r.NoConfiguration {
				extra += " (no configuration: order unknown)"
			}
			if !w.line("- %s %s%s", cell(r.Action), code(r.Address), extra) {
				w.omitted += len(wv.Resources) - i
				break
			}
		}
		w.closeDetails()
		if w.truncated {
			return
		}
	}
	for _, n := range a.WaveNotes {
		w.line("")
		w.line("> %s", cell(n))
	}
}

func parents(w *writer, a *analyze.Analysis) {
	if len(a.Parents) == 0 {
		return
	}
	w.line("### Resources with multiple instances")
	w.line("")
	for _, p := range a.Parents {
		if !w.line("<details><summary>%s (%s) &mdash; %d instance(s)</summary>", htmlEscape(p.Address), p.Kind, len(p.Instances)) {
			return
		}
		w.line("")
		for i, in := range p.Instances {
			if !w.line("- %s %s", cell(in.Action), codeCell(in.Key)) {
				w.omitted += len(p.Instances) - i
				break
			}
		}
		w.closeDetails()
		if w.truncated {
			return
		}
	}
}

// units lists the plans of a merged report.
func units(w *writer, a *analyze.Analysis) {
	if len(a.Units) == 0 {
		return
	}
	w.line("| Plan | Mode | Executed from | Changes |")
	w.line("|---|---|---|---|")
	for _, u := range a.Units {
		acct := code(u.ApplyAccount.ID)
		if !u.ApplyAccount.Resolved {
			acct = "**unresolved**"
		}
		if !w.line("| %s | %s | %s | %s |", codeCell(u.Label), cell(u.Mode), acct, countsLine(u.Actions)) {
			return
		}
	}
	w.line("")
}
