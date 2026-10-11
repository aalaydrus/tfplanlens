package main

import (
	"os"
	"strings"
	"testing"
)

func TestTUIRefusesStdinAndNonTerminals(t *testing.T) {
	code, _, errs := runCLI("tui", "-")
	if code == exitOK || !strings.Contains(errs, "not stdin") {
		t.Errorf("tui -: exit %d, %q", code, errs)
	}
	code, _, errs = runCLI("tui", "../../testdata/plans/basic-create.json")
	if code == exitOK || !strings.Contains(errs, "interactive terminal") {
		t.Errorf("tui without a terminal: exit %d, %q", code, errs)
	}
}

// Bubble Tea's init() queries the terminal's background color in every
// command, which stalls `report` for 5s on terminals that do not answer
// (verified in a pty). The TUI is built on x/term instead; keep it that way.
func TestNoBubbleTeaDependency(t *testing.T) {
	b, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "bubbletea") {
		t.Error("go.mod requires bubbletea; see this test's comment")
	}
}
