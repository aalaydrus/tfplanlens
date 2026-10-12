// Package html renders an analyze.Analysis as one self-contained HTML file
// (SPEC 5.8): inline CSS, JS and SVG, no external requests, dark mode.
// All plan-derived text goes through html/template escaping and is inserted
// into the DOM by the page script via textContent or inert <template>s.
package html

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

//go:embed report.html
var reportTmpl string

//go:embed style.css
var styleCSS string

//go:embed app.js
var appJS string

var tmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"yesno": func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	},
	"upper":      strings.ToUpper,
	"dependents": analyze.DependentsText,
	"sym": func(a string) string {
		return map[string]string{"create": "+", "update": "~", "replace": "-/+", "delete": "-", "read": "<=", "no-op": "=", "forget": "x"}[a]
	},
	"raw": func(b []byte) string { return string(b) },
	"diffClass": func(l string) string {
		switch {
		case strings.HasPrefix(l, "@@"):
			return "hunk"
		case strings.HasPrefix(l, "+"):
			return "add"
		case strings.HasPrefix(l, "-"):
			return "del"
		}
		return "ctx"
	},
}).Parse(reportTmpl))

type tile struct {
	Class, Label string
	Count        int
}

type bar struct {
	Class, Label        string
	Count               int
	Y, TextY, W, CountX int
}

type chart struct {
	Width, Height, LabelW int
	Bars                  []bar
}

type row struct {
	Idx  int
	C    *analyze.Change
	Text string
}

type view struct {
	A       *analyze.Analysis
	CSS     template.CSS
	JS      template.JS
	Tiles   []tile
	Chart   chart
	Rows    []row
	Actions []string
}

func buildView(a *analyze.Analysis) *view {
	v := &view{A: a, CSS: template.CSS(styleCSS), JS: template.JS(appJS)}
	ac := a.Summary.Actions
	counts := []tile{
		{"create", "create", ac.Create}, {"update", "update", ac.Update}, {"replace", "replace", ac.Replace},
		{"delete", "delete", ac.Delete}, {"read", "read", ac.Read}, {"no-op", "no-op", ac.NoOp}, {"forget", "forget", ac.Forget},
	}
	for _, t := range counts {
		if t.Count > 0 || t.Class == "create" || t.Class == "update" || t.Class == "replace" || t.Class == "delete" {
			v.Tiles = append(v.Tiles, t)
		}
	}
	v.Tiles = append(v.Tiles,
		tile{"update", "drift", a.Summary.DriftCount},
		tile{"read", "known after apply", a.Summary.KnownAfterApply},
		tile{"replace", "warnings", a.Summary.Warnings},
	)

	// Horizontal bar chart in inline SVG.
	const labelW, maxW, rowH = 70, 230, 24
	top := 1
	for _, t := range counts {
		top = max(top, t.Count)
	}
	v.Chart = chart{Width: labelW + maxW + 50, LabelW: labelW}
	for i, t := range counts {
		w := t.Count * maxW / top
		if t.Count > 0 && w < 2 {
			w = 2
		}
		y := i*rowH + 4
		v.Chart.Bars = append(v.Chart.Bars, bar{Class: t.Class, Label: t.Label, Count: t.Count, Y: y, TextY: y + 13, W: w, CountX: labelW + w + 6})
	}
	v.Chart.Height = len(counts)*rowH + 8

	seen := map[string]bool{}
	for i := range a.Changes {
		c := &a.Changes[i]
		v.Rows = append(v.Rows, row{Idx: i, C: c, Text: strings.ToLower(fmt.Sprintf("%s %s %s %s %s", c.Address, c.Type, c.Account, c.Module, c.Action))})
		if !seen[c.Action] {
			seen[c.Action] = true
			v.Actions = append(v.Actions, c.Action)
		}
	}
	return v
}

// Render writes the report as a complete HTML document.
func Render(w io.Writer, a *analyze.Analysis) error {
	return tmpl.Execute(w, buildView(a))
}
