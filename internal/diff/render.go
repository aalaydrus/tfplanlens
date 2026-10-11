package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

// RenderJSON writes the result as indented JSON.
func RenderJSON(w io.Writer, r *Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func countsLine(c analyze.ActionCounts) string {
	return fmt.Sprintf("create %d, update %d, replace %d, delete %d", c.Create, c.Update, c.Replace, c.Delete)
}

func label(e Entry) string {
	switch e.Kind {
	case Added:
		return "+ " + e.NewAction
	case Removed:
		return "- was " + e.OldAction
	}
	return "~ " + e.OldAction + " -> " + e.NewAction
}

// RenderText writes a plain-text report.
func RenderText(w io.Writer, r *Result) error {
	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	p("== Plan diff ==")
	p("old: %s", countsLine(r.Old))
	p("new: %s", countsLine(r.New))
	for _, s := range r.Plan {
		p("%s", s)
	}
	if r.Empty() {
		p("")
		p("No differences: %d resource(s) change identically in both plans.", r.Unchanged)
		_, err := io.WriteString(w, b.String())
		return err
	}
	if len(r.NewDanger) > 0 {
		p("")
		p("== New danger ==")
		for _, a := range r.NewDanger {
			p("%s is now deleted or replaced", a)
		}
	}
	if len(r.Defused) > 0 {
		p("")
		p("== No longer deleted or replaced ==")
		for _, a := range r.Defused {
			p("%s", a)
		}
	}
	section := func(title string, es []Entry) {
		if len(es) == 0 {
			return
		}
		p("")
		p("== %s (%d) ==", title, len(es))
		for _, e := range es {
			p("%s  %s", label(e), e.Address)
			for _, d := range e.Differences {
				p("    %s", d)
			}
		}
	}
	section("Only in the new plan", r.Added)
	section("No longer in the plan", r.Removed)
	section("Changed", r.Changed)
	if len(r.DriftNew) > 0 || len(r.DriftGone) > 0 {
		p("")
		p("== Drift ==")
		for _, a := range r.DriftNew {
			p("new drift: %s", a)
		}
		for _, a := range r.DriftGone {
			p("drift gone: %s", a)
		}
	}
	p("")
	p("%d resource(s) change identically in both plans.", r.Unchanged)
	_, err := io.WriteString(w, b.String())
	return err
}

func code(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	pad := ""
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		pad = " "
	}
	return fence + pad + s + pad + fence
}

// RenderMarkdown writes a GitHub-flavored markdown report for a PR comment.
func RenderMarkdown(w io.Writer, r *Result) error {
	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	p("<!-- tfplanlens-diff -->")
	p("## tfplanlens plan diff")
	p("")
	p("| | |")
	p("|---|---|")
	p("| Old plan | %s |", countsLine(r.Old))
	p("| New plan | %s |", countsLine(r.New))
	p("| Unchanged | %d resource(s) |", r.Unchanged)
	for _, s := range r.Plan {
		p("| Plan | %s |", strings.ReplaceAll(s, "|", `\|`))
	}
	p("")
	if r.Empty() {
		p("No differences between the two plans.")
		_, err := io.WriteString(w, b.String())
		return err
	}
	if len(r.NewDanger) > 0 {
		p("**New danger**: the new plan deletes or replaces what the old one did not:")
		p("")
		for _, a := range r.NewDanger {
			p("- %s", code(a))
		}
		p("")
	}
	if len(r.Defused) > 0 {
		p("No longer deleted or replaced:")
		p("")
		for _, a := range r.Defused {
			p("- %s", code(a))
		}
		p("")
	}
	section := func(title string, es []Entry) {
		if len(es) == 0 {
			return
		}
		p("<details><summary><b>%s</b> &mdash; %d</summary>", title, len(es))
		p("")
		for _, e := range es {
			p("- **%s** %s", strings.ReplaceAll(label(e), "|", `\|`), code(e.Address))
			for _, d := range e.Differences {
				p("  - %s", code(d))
			}
		}
		p("")
		p("</details>")
		p("")
	}
	section("Only in the new plan", r.Added)
	section("No longer in the plan", r.Removed)
	section("Changed", r.Changed)
	if len(r.DriftNew) > 0 || len(r.DriftGone) > 0 {
		p("<details><summary><b>Drift</b> &mdash; %d new, %d gone</summary>", len(r.DriftNew), len(r.DriftGone))
		p("")
		for _, a := range r.DriftNew {
			p("- new drift: %s", code(a))
		}
		for _, a := range r.DriftGone {
			p("- drift gone: %s", code(a))
		}
		p("")
		p("</details>")
	}
	_, err := io.WriteString(w, b.String())
	return err
}
