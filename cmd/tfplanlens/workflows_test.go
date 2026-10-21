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

// A bump of golang.org/x/net and x/term raised go.mod from "go 1.23.0" to
// "go 1.26.0". Dependabot must not propose minor or major x/* updates, so the
// minimum Go version only changes when a person decides it.
func TestDependabotDoesNotRaiseTheMinimumGoVersion(t *testing.T) {
	var d struct {
		Updates []struct {
			Ecosystem string `yaml:"package-ecosystem"`
			Ignore    []struct {
				Name  string   `yaml:"dependency-name"`
				Types []string `yaml:"update-types"`
			}
		}
	}
	if err := yaml.Unmarshal([]byte(read(t, repoRoot+"/.github/dependabot.yml")), &d); err != nil {
		t.Fatal(err)
	}
	for _, u := range d.Updates {
		if u.Ecosystem != "gomod" {
			continue
		}
		for _, ig := range u.Ignore {
			if ig.Name == "golang.org/x/*" && strings.Join(ig.Types, ",") == "version-update:semver-minor,version-update:semver-major" {
				return
			}
		}
	}
	t.Error("dependabot.yml must ignore minor and major updates of golang.org/x/* (see this test's comment)")
}

// The minimum Go version is a decision, not a side effect. A merged x/net bump
// once raised go.mod to "go 1.25.0" and CI stayed green, because CI installs
// whatever go.mod asks for. This test turns that into a red check. To move the
// minimum on purpose, change it here, in README.md, SPEC.md, CONTRIBUTING.md
// and the notes that name it (grep for "1.23").
func TestGoModKeepsTheMinimumGoVersion(t *testing.T) {
	const want = "1.23.0"
	s := read(t, repoRoot+"/go.mod")
	m := regexp.MustCompile(`(?m)^go (\S+)$`).FindStringSubmatch(s)
	if m == nil || m[1] != want {
		t.Errorf("go.mod has go line %v, want %q. A dependency bump probably raised it; see this test's comment", m, want)
	}
	if regexp.MustCompile(`(?m)^toolchain `).MatchString(s) {
		t.Error("go.mod has a toolchain line; remove it so the minimum stays what the go line says")
	}
}

// go.mod's go line is the oldest Go we support, not the one we build releases
// with: that Go is out of support and gets no security fixes (an eMudhra CA
// distrust fix, for one, only went into the two newest Go releases). Releases
// are built with the current stable Go, and one CI job proves the minimum
// still builds, without being allowed to download a newer Go.
func TestReleasesAreBuiltWithASupportedGoAndTheMinimumStillBuilds(t *testing.T) {
	rel := read(t, repoRoot+"/.github/workflows/release.yml")
	if !strings.Contains(rel, "go-version: stable") || strings.Contains(rel, "go-version-file") {
		t.Error("release.yml must build with go-version: stable, not the go line in go.mod")
	}
	ci := read(t, repoRoot+"/.github/workflows/ci.yml")
	i := strings.Index(ci, "minimum-go:")
	if i < 0 {
		t.Fatal("ci.yml has no minimum-go job")
	}
	job := ci[i:]
	if j := strings.Index(job, "\n  govulncheck:"); j > 0 {
		job = job[:j]
	}
	for _, want := range []string{"GOTOOLCHAIN: local", "go-version-file: go.mod"} {
		if !strings.Contains(job, want) {
			t.Errorf("the minimum-go job must contain %q", want)
		}
	}
}
