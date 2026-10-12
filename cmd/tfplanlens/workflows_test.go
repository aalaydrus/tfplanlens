package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// A tag such as actions/checkout@v4 can be moved to different code by whoever
// controls that repository; a commit SHA cannot. Workflows that run with write
// access must not depend on a movable tag. Dependabot keeps the pins current
// (.github/dependabot.yml). To pin a new action:
//
//	git ls-remote --tags https://github.com/OWNER/REPO 'refs/tags/vX*'
//
// and write `uses: OWNER/REPO@<40-char sha> # vX.Y.Z`.
func TestWorkflowActionsArePinnedToCommitSHAs(t *testing.T) {
	files, _ := filepath.Glob(repoRoot + "/.github/workflows/*.yml")
	if len(files) < 3 {
		t.Fatalf("found %d workflows", len(files))
	}
	uses := regexp.MustCompile(`(?m)^\s*(?:-\s*)?uses:\s*(\S+)`)
	pinned := regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._/-]+@[0-9a-f]{40}$`)
	n := 0
	for _, f := range files {
		s := read(t, f)
		var doc map[string]any
		if err := yaml.Unmarshal([]byte(s), &doc); err != nil {
			t.Errorf("%s is not valid YAML: %v", f, err)
		}
		for _, m := range uses.FindAllStringSubmatch(s, -1) {
			n++
			if strings.HasPrefix(m[1], "./") {
				continue // a local action
			}
			if !pinned.MatchString(m[1]) {
				t.Errorf("%s: %s is not pinned to a full commit SHA", filepath.Base(f), m[1])
			}
		}
	}
	if n < 10 {
		t.Errorf("only %d uses: lines found; did the extractor break?", n)
	}
}

func TestDependabotKeepsActionsAndModulesCurrent(t *testing.T) {
	var d struct {
		Updates []struct {
			Ecosystem string `yaml:"package-ecosystem"`
		}
	}
	if err := yaml.Unmarshal([]byte(read(t, repoRoot+"/.github/dependabot.yml")), &d); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, u := range d.Updates {
		got[u.Ecosystem] = true
	}
	if !got["github-actions"] || !got["gomod"] {
		t.Errorf("dependabot must cover github-actions and gomod, has %v", got)
	}
}

// Pushes made with a workflow's own token do not start other workflows, so a
// bot commit on a pull request has no CI results and would block required
// status checks. The smoke workflow therefore only reads the repository;
// scripts/smoke-refresh.sh refreshes the committed plans and reports.
func TestSmokeWorkflowNeverPushesToTheRepository(t *testing.T) {
	s := read(t, repoRoot+"/.github/workflows/smoke.yml")
	for _, bad := range []string{"git push", "git commit", "contents: write"} {
		if strings.Contains(s, bad) {
			t.Errorf("smoke.yml contains %q; it must not write to the repository (see scripts/smoke-refresh.sh)", bad)
		}
	}
	if !strings.Contains(s, "contents: read") {
		t.Error("smoke.yml should declare contents: read")
	}
	if !fileExists(repoRoot + "/scripts/smoke-refresh.sh") {
		t.Error("scripts/smoke-refresh.sh is missing")
	}
}
