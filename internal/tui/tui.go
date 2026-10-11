// Package tui is the interactive terminal view of an analyze.Analysis
// (SPEC 10): an expandable tree of changes with search, an action filter,
// grouping, and a waves view. It consumes only the Analysis.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

// Group-by modes, cycled with "g".
var groupModes = []string{"none", "account", "module", "unit"}

// Action filters, cycled with "f".
var actionFilters = []string{"", "create", "update", "replace", "delete", "read", "forget"}

const (
	viewChanges = iota
	viewWaves
)

// node is one line of the tree. Nodes with children can be expanded.
type node struct {
	key      string // stable identity for the expanded set
	label    string
	action   string // colors the label
	children []*node
}

// Model is the Bubble Tea model.
type Model struct {
	a *analyze.Analysis

	view      int
	group     int
	filter    int
	showQuiet bool
	query     string
	searching bool

	expanded map[string]bool
	cursor   int
	top      int
	width    int
	height   int

	// Color enables ANSI colors.
	Color bool

	tree []*node
	rows []row // flattened visible tree
}

type row struct {
	n     *node
	depth int
}

// New returns a model showing a.
func New(a *analyze.Analysis) *Model {
	m := &Model{a: a, expanded: map[string]bool{}, width: 100, height: 30}
	// Waves are short: show them open.
	for _, w := range a.Waves {
		m.expanded[waveKey(w)] = true
	}
	m.rebuild()
	return m
}

func waveKey(w analyze.Wave) string { return fmt.Sprintf("wave/%s/%d", w.Phase, w.Number) }

// Key is one key press: a name for special keys (up, down, left, right,
// pgup, pgdown, home, end, enter, esc, backspace, ctrl+c) or the typed text.
type Key string

// Resize sets the terminal size.
func (m *Model) Resize(w, h int) {
	m.width, m.height = w, h
	m.clamp()
}

// Update applies a key press and reports whether the program should quit.
func (m *Model) Update(k Key) (quit bool) {
	if m.searching {
		m.searchKey(k)
		return false
	}
	switch k {
	case "q", "ctrl+c":
		return true
	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "pgup":
		m.cursor -= m.listHeight()
	case "pgdown":
		m.cursor += m.listHeight()
	case "home":
		m.cursor = 0
	case "end":
		m.cursor = len(m.rows) - 1
	case "enter", " ", "right", "l":
		m.toggle(true)
	case "left", "h":
		m.toggle(false)
	case "e":
		m.setAll(true)
	case "c":
		m.setAll(false)
	case "/":
		m.searching = true
	case "esc":
		m.query = ""
		m.rebuild()
	case "f":
		m.filter = (m.filter + 1) % len(actionFilters)
		m.rebuild()
	case "g":
		m.group = (m.group + 1) % len(groupModes)
		m.rebuild()
	case "n":
		m.showQuiet = !m.showQuiet
		m.rebuild()
	case "w":
		if m.view == viewChanges {
			m.view = viewWaves
		} else {
			m.view = viewChanges
		}
		m.cursor, m.top = 0, 0
		m.rebuild()
	}
	m.clamp()
	return false
}

func (m *Model) searchKey(k Key) {
	switch k {
	case "enter", "esc":
		m.searching = false
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
	case "ctrl+c":
		m.searching = false
	default:
		if len([]rune(string(k))) == 1 && k >= " " {
			m.query += string(k)
		}
	}
	m.rebuild()
}

// toggle expands (open) or collapses the node under the cursor. Collapsing a
// leaf or an already-collapsed node jumps to its parent.
func (m *Model) toggle(open bool) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	r := m.rows[m.cursor]
	if len(r.n.children) > 0 {
		if open {
			m.expanded[r.n.key] = !m.expanded[r.n.key]
		} else if m.expanded[r.n.key] {
			m.expanded[r.n.key] = false
		} else {
			m.toParent()
		}
	} else if !open {
		m.toParent()
	}
	m.flatten()
}

func (m *Model) toParent() {
	d := m.rows[m.cursor].depth
	for i := m.cursor - 1; i >= 0; i-- {
		if m.rows[i].depth < d {
			m.cursor = i
			return
		}
	}
}

func (m *Model) setAll(open bool) {
	var walk func(ns []*node)
	walk = func(ns []*node) {
		for _, n := range ns {
			if len(n.children) > 0 {
				m.expanded[n.key] = open
				walk(n.children)
			}
		}
	}
	walk(m.roots())
	m.flatten()
	m.cursor = 0
}

func (m *Model) listHeight() int { return max(m.height-4, 3) }

func (m *Model) clamp() {
	m.cursor = min(max(m.cursor, 0), max(len(m.rows)-1, 0))
	h := m.listHeight()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+h {
		m.top = m.cursor - h + 1
	}
	m.top = max(m.top, 0)
}

func (m *Model) roots() []*node { return m.tree }

func (m *Model) rebuild() {
	if m.view == viewWaves {
		m.tree = m.wavesTree()
	} else {
		m.tree = m.changesTree()
	}
	m.flatten()
	m.clamp()
}

func (m *Model) flatten() {
	m.rows = m.rows[:0]
	var walk func(ns []*node, depth int)
	walk = func(ns []*node, depth int) {
		for _, n := range ns {
			m.rows = append(m.rows, row{n, depth})
			if m.expanded[n.key] {
				walk(n.children, depth+1)
			}
		}
	}
	walk(m.roots(), 0)
	m.clamp()
}

func (m *Model) matches(c *analyze.Change) bool {
	if f := actionFilters[m.filter]; f != "" && c.Action != f {
		return false
	}
	if !m.showQuiet && !c.Listed() {
		return false
	}
	if q := strings.ToLower(m.query); q != "" {
		hay := strings.ToLower(c.Address + " " + c.Type + " " + c.Account + " " + c.Module + " " + c.Action)
		if !strings.Contains(hay, q) {
			return false
		}
	}
	return true
}

func (m *Model) groupKey(c *analyze.Change) string {
	switch groupModes[m.group] {
	case "account":
		return c.Account
	case "module":
		return c.Module
	case "unit":
		return c.Unit
	}
	return ""
}

func (m *Model) changesTree() []*node {
	var visible []*analyze.Change
	for i := range m.a.Changes {
		if c := &m.a.Changes[i]; m.matches(c) {
			visible = append(visible, c)
		}
	}
	// Group count/for_each instances under their parent.
	byParent := map[string][]*analyze.Change{}
	for _, c := range visible {
		if c.Parent != "" {
			byParent[c.Parent] = append(byParent[c.Parent], c)
		}
	}
	var items []*node
	itemGroup := map[*node]string{}
	seenParent := map[string]bool{}
	for _, c := range visible {
		var n *node
		if c.Parent != "" && len(byParent[c.Parent]) > 1 {
			if seenParent[c.Parent] {
				continue
			}
			seenParent[c.Parent] = true
			n = parentNode(c.Parent, byParent[c.Parent])
		} else {
			n = changeNode(c)
		}
		items = append(items, n)
		itemGroup[n] = m.groupKey(c)
	}
	if groupModes[m.group] == "none" {
		return items
	}
	var keys []string
	idx := map[string]*node{}
	for _, n := range items {
		g := itemGroup[n]
		gn := idx[g]
		if gn == nil {
			gn = &node{key: "group/" + g, label: g}
			idx[g] = gn
			keys = append(keys, g)
		}
		gn.children = append(gn.children, n)
	}
	sort.Strings(keys)
	out := make([]*node, 0, len(keys))
	for _, k := range keys {
		g := idx[k]
		g.label = fmt.Sprintf("%s (%d)", k, len(g.children))
		out = append(out, g)
	}
	return out
}

func parentNode(parent string, cs []*analyze.Change) *node {
	var counts analyze.ActionCounts
	for _, c := range cs {
		switch c.Action {
		case "create":
			counts.Create++
		case "update":
			counts.Update++
		case "replace":
			counts.Replace++
		case "delete":
			counts.Delete++
		case "read":
			counts.Read++
		case "forget":
			counts.Forget++
		default:
			counts.NoOp++
		}
	}
	n := &node{key: "parent/" + parent, label: fmt.Sprintf("%s  [%d instances: %s]", parent, len(cs), countsText(counts))}
	n.action = "update"
	for _, c := range cs {
		child := changeNode(c)
		child.label = fmt.Sprintf("%s  %s", symbol(c.Action), c.Key)
		if c.Wave > 0 {
			child.label += fmt.Sprintf("  (%s wave %d)", c.WavePhase, c.Wave)
		}
		n.children = append(n.children, child)
	}
	return n
}

func countsText(c analyze.ActionCounts) string {
	var parts []string
	for _, p := range []struct {
		n    int
		name string
	}{{c.Create, "create"}, {c.Update, "update"}, {c.Replace, "replace"}, {c.Delete, "delete"}, {c.Read, "read"}, {c.Forget, "forget"}, {c.NoOp, "no-op"}} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.name))
		}
	}
	return strings.Join(parts, ", ")
}

func symbol(a string) string {
	if s, ok := map[string]string{"create": "+", "update": "~", "replace": "-/+", "delete": "-", "read": "<=", "no-op": "=", "forget": "x"}[a]; ok {
		return s
	}
	return "?"
}

func changeNode(c *analyze.Change) *node {
	label := fmt.Sprintf("%s  %s", symbol(c.Action), c.Address)
	if c.Stateful {
		label += "  [STATEFUL]"
	}
	if c.Wave > 0 {
		label += fmt.Sprintf("  (%s wave %d)", c.WavePhase, c.Wave)
	}
	n := &node{key: "change/" + c.Address, label: label, action: c.Action}
	detail := func(s string) { n.children = append(n.children, &node{key: n.key + "/" + s, label: s}) }
	detail(fmt.Sprintf("%s; %s", c.Label(), c.Type))
	detail(fmt.Sprintf("target account %s (source: %s)", c.Account, c.AccountSource))
	if c.ReasonText != "" {
		detail("why: " + c.ReasonText)
	}
	for _, w := range c.Warnings {
		detail("WARNING: " + w.Message)
	}
	for _, at := range c.Attrs {
		detail(attrLine(at))
	}
	return n
}

func attrLine(at analyze.Attr) string {
	mark := ""
	if at.ForcesReplace {
		mark = "  [forces replacement]"
	}
	switch at.Kind {
	case "sensitive":
		return at.Path + ": (sensitive value changed)" + mark
	case "long":
		if at.Diff != nil {
			return fmt.Sprintf("%s: %s (+%d -%d of %d lines)%s", at.Path, at.Diff.Header, at.Diff.Added, at.Diff.Removed, at.Diff.Total, mark)
		}
	}
	return fmt.Sprintf("%s: %s -> %s%s", at.Path, short(at.Before), short(at.After), mark)
}

func short(b []byte) string {
	if len(b) == 0 {
		return "(absent)"
	}
	s := string(b)
	if len(s) > 60 {
		s = s[:57] + "..."
	}
	return s
}

func (m *Model) wavesTree() []*node {
	var out []*node
	for _, cy := range m.a.Cycles {
		out = append(out, &node{label: "CYCLE: " + strings.Join(cy.Resources, " -> "), action: "delete", key: "cycle/" + strings.Join(cy.Resources, ",")})
	}
	q := strings.ToLower(m.query)
	for _, w := range m.a.Waves {
		wn := &node{key: waveKey(w)}
		for _, r := range w.Resources {
			if q != "" && !strings.Contains(strings.ToLower(r.Address), q) {
				continue
			}
			if f := actionFilters[m.filter]; f != "" && r.Action != f {
				continue
			}
			label := fmt.Sprintf("%s  %s", symbol(r.Action), r.Address)
			if r.Instances > 1 {
				label += fmt.Sprintf("  (%d instances)", r.Instances)
			}
			if r.NoConfiguration {
				label += "  [no configuration: order unknown]"
			}
			wn.children = append(wn.children, &node{key: wn.key + "/" + r.Address, label: label, action: r.Action})
		}
		if len(wn.children) == 0 {
			continue
		}
		wn.label = fmt.Sprintf("%s wave %d (%d)", w.Phase, w.Number, len(wn.children))
		out = append(out, wn)
	}
	return out
}

// style wraps text in an SGR sequence when color is on.
func (m *Model) style(code, s string) string {
	if !m.Color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

var actionColor = map[string]string{
	"create": "32", "update": "33", "replace": "38;5;208", "delete": "31", "read": "34", "forget": "35",
}

// View renders the screen as lines separated by \\n.
func (m *Model) View() string {
	var b strings.Builder
	s := m.a.Summary
	b.WriteString(m.style("1", "tfplanlens") + fmt.Sprintf("  %s plan: %d create, %d update, %d replace, %d delete", s.Mode, s.Actions.Create, s.Actions.Update, s.Actions.Replace, s.Actions.Delete))
	if len(m.a.Cycles) > 0 {
		b.WriteString("  " + m.style("31", "CYCLES"))
	}
	b.WriteString("\n")
	view := "changes"
	if m.view == viewWaves {
		view = "waves"
	}
	status := fmt.Sprintf("view: %s | group: %s | action: %s | no-ops: %v", view, groupModes[m.group], orAll(actionFilters[m.filter]), m.showQuiet)
	if m.query != "" || m.searching {
		status += " | search: " + m.query
		if m.searching {
			status += "_"
		}
	}
	b.WriteString(m.dim(status) + "\n")

	h := m.listHeight()
	end := min(m.top+h, len(m.rows))
	if len(m.rows) == 0 {
		b.WriteString("  (nothing to show)\n")
	}
	for i := m.top; i < end; i++ {
		r := m.rows[i]
		mark := "  "
		if len(r.n.children) > 0 {
			mark = "> "
			if m.expanded[r.n.key] {
				mark = "v "
			}
		}
		line := strings.Repeat("  ", r.depth) + mark + r.n.label
		if m.width > 4 && len([]rune(line)) > m.width-1 {
			line = string([]rune(line)[:m.width-2]) + "…"
		}
		switch {
		case i == m.cursor && m.Color:
			line = m.style("7", " "+line)
		case i == m.cursor:
			line = "*" + line
		case actionColor[r.n.action] != "":
			line = m.style(actionColor[r.n.action], " "+line)
		default:
			line = " " + line
		}
		b.WriteString(line + "\n")
	}
	if m.view == viewWaves {
		b.WriteString(m.dim("Terraform applies in parallel: possible parallelism, not a strict order.") + "\n")
	}
	b.WriteString(m.dim("↑↓ move  →/enter expand  ← collapse  e/c all  / search  f action  g group  n no-ops  w waves  q quit"))
	return b.String()
}

func (m *Model) dim(s string) string { return m.style("2", s) }

func orAll(s string) string {
	if s == "" {
		return "all"
	}
	return s
}
