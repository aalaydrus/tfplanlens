package main

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The agent docs (field-notes/Author-Notes) exist to save an agent's time, so
// a stale path, test name or command in them is worse than no doc. These
// tests keep them true.
const (
	repoRoot  = "../.."
	agentDocs = repoRoot + "/field-notes/Author-Notes"
)

func agentDocFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(agentDocs, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
			files = append(files, p)
		}
		return err
	})
	if err != nil || len(files) == 0 {
		t.Fatalf("no agent docs under %s: %v", agentDocs, err)
	}
	return files
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The entry point is read on every task: keep it short.
func TestAgentDocsStartPageStaysSmall(t *testing.T) {
	s := read(t, agentDocs+"/AGENTS_START_HERE.md")
	if n := strings.Count(s, "\n"); n > 90 {
		t.Errorf("AGENTS_START_HERE.md is %d lines; move detail into refs/ or playbooks/ (limit 90)", n)
	}
}

func TestAgentDocsLinksAndPathsExist(t *testing.T) {
	link := regexp.MustCompile(`\]\(([^)\s#]+)(?:#[^)]*)?\)`)
	span := regexp.MustCompile("`([A-Za-z0-9_./-]+)`")
	roots := []string{"cmd/", "internal/", "testdata/", "scripts/", "simulation/", "docs/", "field-notes/", ".github/"}
	rootFiles := map[string]bool{"README.md": true, "SPEC.md": true, "go.mod": true, "tfplanlens.config.yaml": true, "AGENTS.md": true}
	for _, f := range agentDocFiles(t) {
		s := read(t, f)
		for _, m := range link.FindAllStringSubmatch(s, -1) {
			if strings.Contains(m[1], "://") {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(f), m[1])); err != nil {
				t.Errorf("%s: broken link %s", f, m[1])
			}
		}
		for _, m := range span.FindAllStringSubmatch(s, -1) {
			p := m[1]
			isRepoPath := rootFiles[p]
			for _, r := range roots {
				isRepoPath = isRepoPath || strings.HasPrefix(p, r)
			}
			if !isRepoPath || strings.Contains(p, "*") {
				continue
			}
			if _, err := os.Stat(filepath.Join(repoRoot, p)); err != nil {
				t.Errorf("%s: `%s` does not exist", f, p)
			}
		}
	}
}

func TestAgentDocsTestNamesExist(t *testing.T) {
	have := ""
	filepath.WalkDir(repoRoot, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, "_test.go") && !strings.Contains(p, "node_modules") {
			b, _ := os.ReadFile(p)
			have += string(b)
		}
		return nil
	})
	name := regexp.MustCompile("`(Test[A-Za-z0-9]+)(\\*?)`")
	for _, f := range agentDocFiles(t) {
		for _, m := range name.FindAllStringSubmatch(read(t, f), -1) {
			want := "func " + m[1]
			if m[2] == "" {
				want += "("
			}
			if !strings.Contains(have, want) {
				t.Errorf("%s: no test `%s%s`", f, m[1], m[2])
			}
		}
	}
}

// Every `tfplanlens ...` line in a code block must name a real command and
// real flags. Lines with [optional] parts (usage syntax) are skipped.
func TestAgentDocsCommandsParse(t *testing.T) {
	placeholder := regexp.MustCompile(`<[^>]*>`)
	checked := 0
	for _, f := range agentDocFiles(t) {
		inBlock := false
		for _, line := range strings.Split(read(t, f), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inBlock = !inBlock
				continue
			}
			line = strings.TrimSpace(line)
			if !inBlock || !strings.HasPrefix(line, "tfplanlens ") || strings.Contains(line, "[") {
				continue
			}
			for _, sep := range []string{" #", " ||", " |", " >"} {
				if i := strings.Index(line, sep); i >= 0 {
					line = line[:i]
				}
			}
			args := strings.Fields(placeholder.ReplaceAllString(line, "X"))[1:]
			root := newRootCmd(strings.NewReader(""), io.Discard, io.Discard)
			cmd, rest, err := root.Find(args)
			if err == nil && cmd != root {
				err = cmd.ParseFlags(rest)
			}
			if err != nil || cmd == root {
				t.Errorf("%s: `%s`: %v", f, line, err)
			}
			checked++
		}
	}
	if checked < 10 {
		t.Errorf("only %d commands checked: the docs lost their examples or the extractor broke", checked)
	}
}

func TestAgentSkillsHaveValidFrontmatter(t *testing.T) {
	dirs, _ := filepath.Glob(agentDocs + "/SKILLS/*/SKILL.md")
	if len(dirs) < 3 {
		t.Fatalf("found %d skills", len(dirs))
	}
	for _, f := range dirs {
		s := read(t, f)
		m := regexp.MustCompile(`(?s)\A---\nname: ([a-z0-9-]+)\ndescription: (.+?)\n---\n`).FindStringSubmatch(s)
		if m == nil {
			t.Errorf("%s: frontmatter must be exactly name and description between --- lines", f)
			continue
		}
		if dir := filepath.Base(filepath.Dir(f)); m[1] != dir {
			t.Errorf("%s: name %q does not match its folder %q", f, m[1], dir)
		}
		if len(m[2]) > 1024 || !strings.Contains(m[2], "tfplanlens") {
			t.Errorf("%s: description must mention tfplanlens and be under 1024 chars (%d)", f, len(m[2]))
		}
	}
}
