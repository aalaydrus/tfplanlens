package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/plan"
)

func load(t *testing.T, name string) *analyze.Analysis {
	t.Helper()
	files, _ := filepath.Glob("../../testdata/plans/real/" + name + "-tf*.json")
	if len(files) == 0 {
		t.Skip("no real plan " + name)
	}
	f, err := os.Open(files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := plan.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	return analyze.Analyze(p, nil, &analyze.Options{ApplyAccount: "111111111111", ApplySource: "flag"})
}

func key(m *Model, k string) { m.Update(Key(k)) }

func labels(m *Model) string {
	var b strings.Builder
	for _, r := range m.rows {
		b.WriteString(r.n.label + "\n")
	}
	return b.String()
}

func TestInstancesAreGroupedUnderTheirParent(t *testing.T) {
	m := New(load(t, "complex"))
	out := labels(m)
	if !strings.Contains(out, "terraform_data.worker  [2 instances: 2 delete]") {
		t.Fatalf("no worker parent row:\n%s", out)
	}
	if strings.Contains(out, "worker[2]") || strings.Contains(out, "\n- 2") {
		t.Errorf("instances visible before expanding:\n%s", out)
	}
	// Move the cursor onto the worker parent and expand it.
	for i, r := range m.rows {
		if strings.HasPrefix(r.n.label, "terraform_data.worker") {
			m.cursor = i
		}
	}
	key(m, "enter")
	out = labels(m)
	for _, want := range []string{"-  2", "-  3"} {
		if !strings.Contains(out, want) {
			t.Errorf("expanded worker missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "=  0") {
		t.Errorf("plain no-op instance shown by default:\n%s", out)
	}
	key(m, "n")
	if out := labels(m); !strings.Contains(out, "[4 instances: 2 delete, 2 no-op]") || !strings.Contains(out, "=  0") {
		t.Errorf("no-ops not shown after n:\n%s", out)
	}
}

func TestExpandShowsAttributesAndWhy(t *testing.T) {
	m := New(load(t, "complex"))
	for i, r := range m.rows {
		if strings.Contains(r.n.label, "terraform_data.db") {
			m.cursor = i
		}
	}
	key(m, "enter")
	out := labels(m)
	if !strings.Contains(out, "why: Attribute change forces replacement") || !strings.Contains(out, "[forces replacement]") {
		t.Errorf("detail missing:\n%s", out)
	}
	key(m, "left")
	if strings.Contains(labels(m), "why:") {
		t.Errorf("collapse did not hide detail")
	}
}

func TestFilterSearchAndNoOps(t *testing.T) {
	m := New(load(t, "complex"))
	key(m, "f") // create
	if out := labels(m); !strings.Contains(out, "subnet") || strings.Contains(out, "terraform_data.db") {
		t.Errorf("create filter:\n%s", out)
	}
	for i := 0; i < len(actionFilters)-1; i++ {
		key(m, "f")
	}
	if actionFilters[m.filter] != "" {
		t.Fatal("filter did not cycle back to all")
	}
	key(m, "/")
	for _, r := range "lb" {
		key(m, string(r))
	}
	key(m, "enter")
	if out := labels(m); strings.Contains(out, "terraform_data.db") || !strings.Contains(out, "terraform_data.lb") {
		t.Errorf("search lb:\n%s", out)
	}
	key(m, "esc")
	if m.query != "" {
		t.Errorf("esc did not clear the search")
	}
	key(m, "n")
	if out := labels(m); !strings.Contains(out, "[4 instances: 1 create, 1 delete, 2 no-op]") {
		t.Errorf("showing no-ops did not add the subnet no-op instances:\n%s", out)
	}
}

func TestGroupByAccountAndModule(t *testing.T) {
	m := New(load(t, "complex"))
	key(m, "g")
	if !strings.Contains(labels(m), "111111111111 (") {
		t.Errorf("account grouping:\n%s", labels(m))
	}
	key(m, "g")
	if out := labels(m); !strings.Contains(out, "root (") {
		t.Errorf("module grouping:\n%s", out)
	}
}

func TestWavesView(t *testing.T) {
	m := New(load(t, "complex"))
	key(m, "w")
	out := labels(m)
	for _, want := range []string{"apply wave 1 (", "apply wave 2 (2)", "destroy wave 1 (1)", "terraform_data.lb"} {
		if !strings.Contains(out, want) {
			t.Errorf("waves view missing %q:\n%s", want, out)
		}
	}
	if v := m.View(); !strings.Contains(v, "possible parallelism") {
		t.Errorf("no parallelism caveat:\n%s", v)
	}
	key(m, "w")
	if strings.Contains(labels(m), "wave 1 (") {
		t.Errorf("did not switch back")
	}
}

func TestViewFitsTheTerminalAndScrolls(t *testing.T) {
	m := New(load(t, "complex"))
	m.Resize(60, 8)
	for i := 0; i < 50; i++ {
		key(m, "down")
	}
	lines := strings.Split(m.View(), "\n")
	if len(lines) > 8 {
		t.Errorf("view is %d lines in an 8 line terminal", len(lines))
	}
	if m.cursor != len(m.rows)-1 {
		t.Errorf("cursor %d, want last row %d", m.cursor, len(m.rows)-1)
	}
	if m.top == 0 {
		t.Errorf("list did not scroll")
	}
}

func TestQuit(t *testing.T) {
	m := New(load(t, "complex"))
	if !m.Update("q") || !m.Update("ctrl+c") {
		t.Error("q / ctrl+c did not quit")
	}
}

func TestParseKeys(t *testing.T) {
	for in, want := range map[string]string{
		"\x1b[A\x1b[B\x1b[C\x1b[D":   "up down right left",
		"\x1b[5~\x1b[6~\x1b[H\x1b[F": "pgup pgdown home end",
		"\x1b":                       "esc",
		"\r":                         "enter",
		"\x7f":                       "backspace",
		"\x03":                       "ctrl+c",
		"q/é":                        "q / é",
		"\x1b[99~x":                  "x",
	} {
		var got []string
		for _, k := range parseKeys([]byte(in)) {
			got = append(got, string(k))
		}
		if strings.Join(got, " ") != want {
			t.Errorf("%q: got %v, want %q", in, got, want)
		}
	}
}

func TestPlainViewMarksTheCursorWithoutColor(t *testing.T) {
	m := New(load(t, "complex"))
	v := m.View()
	if strings.Contains(v, "\x1b[") {
		t.Errorf("escape codes in uncolored view")
	}
	if !strings.Contains(v, "\n*") {
		t.Errorf("no cursor marker:\n%s", v)
	}
	m.Color = true
	if !strings.Contains(m.View(), "\x1b[7m") {
		t.Errorf("colored view has no reverse-video cursor")
	}
}
