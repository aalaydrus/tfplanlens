package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The files an open-source repository needs, kept honest by tests.

func TestOpenSourceFilesExist(t *testing.T) {
	for _, f := range []string{"LICENSE", "SECURITY.md", "CONTRIBUTING.md", "THIRD_PARTY_NOTICES.md"} {
		if !fileExists(repoRoot + "/" + f) {
			t.Errorf("%s is missing", f)
		}
	}
	// The one rule must stay prominent.
	if !strings.Contains(read(t, repoRoot+"/CONTRIBUTING.md"), "never commit a real plan or state file") {
		t.Error("CONTRIBUTING.md lost its rule about real plan and state files")
	}
}

// Every module linked into the binary (on Linux and on Windows) must appear in
// THIRD_PARTY_NOTICES.md, so a new dependency cannot ship without a notice.
func TestThirdPartyNoticesListEveryLinkedModule(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	notices := read(t, repoRoot+"/THIRD_PARTY_NOTICES.md")
	for _, goos := range []string{"linux", "windows", "darwin"} {
		cmd := exec.Command("go", "list", "-deps", "-f", "{{with .Module}}{{.Path}}{{end}}", "./cmd/tfplanlens")
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(), "GOOS="+goos)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("go list (%s): %v", goos, err)
		}
		for _, mod := range strings.Fields(string(out)) {
			if mod == "github.com/aalaydrus/tfplanlens" {
				continue
			}
			if !strings.Contains(notices, mod) {
				t.Errorf("THIRD_PARTY_NOTICES.md does not mention %s (linked on %s)", mod, goos)
			}
		}
	}
}

// Real plans and state hold secrets in plain text. .gitignore must block them
// everywhere except the fabricated examples.
func TestGitignoreBlocksPlanAndStateFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	ignored := func(p string) bool {
		cmd := exec.Command("git", "check-ignore", "-q", "--no-index", p)
		cmd.Dir = repoRoot
		return cmd.Run() == nil // exit 0 = ignored, 1 = not ignored
	}
	for _, p := range []string{"plan.json", "infra/plan.json", "plan.bin", "prod/terraform.tfstate",
		"terraform.tfstate.backup", "prod.tfvars", "plan-new.json", ".terraform/providers/x", "tfplan"} {
		if !ignored(p) {
			t.Errorf("%s should be ignored", p)
		}
	}
	for _, p := range []string{"simulation/assets/input/plan.json", "simulation/assets/input/plan-revised.json",
		"simulation/assets/input/terraform.tfstate", "testdata/plans/real/simple-tf1.16.5.json",
		"testdata/plans/basic-create.json", "testdata/real/simple/v1/main.tf", "main.tf", "go.mod"} {
		if ignored(p) {
			t.Errorf("%s must not be ignored (fabricated example or source)", p)
		}
	}
}
