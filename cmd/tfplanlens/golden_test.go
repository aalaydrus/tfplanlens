package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/golden")

// Formats covered by golden files (SPEC section 7); html has its own checks.
var goldenFormats = map[string]string{"text": "txt", "markdown": "md", "json": "json"}

// TestGolden runs `report --format <f> --no-color` on every fixture and
// compares stdout with testdata/golden/<fixture>.<ext>. While report is not
// implemented the test skips; once it is, `go test -update ./cmd/...`
// generates the golden files, which must be reviewed before committing.
func TestGolden(t *testing.T) {
	plans, err := filepath.Glob("../../testdata/plans/*.json")
	if err != nil || len(plans) == 0 {
		t.Fatalf("no fixtures found: %v", err)
	}
	for _, plan := range plans {
		name := strings.TrimSuffix(filepath.Base(plan), ".json")
		for format, ext := range goldenFormats {
			t.Run(name+"/"+format, func(t *testing.T) {
				var out, errb bytes.Buffer
				root := newRootCmd(strings.NewReader(""), &out, &errb)
				root.SetArgs([]string{"report", plan, "--format", format, "--no-color"})
				if err := root.Execute(); err != nil {
					if errors.Is(err, errNotImplemented) {
						t.Skip("report not implemented yet")
					}
					t.Fatalf("%v\n%s", err, errb.String())
				}
				golden := filepath.Join("../../testdata/golden", name+"."+ext)
				if *update {
					if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
						t.Fatal(err)
					}
					return
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatalf("missing golden file (run with -update): %v", err)
				}
				if !bytes.Equal(out.Bytes(), want) {
					t.Errorf("output differs from %s (run with -update to accept)", golden)
				}
			})
		}
	}
}

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"version"}, strings.NewReader(""), &out, &out); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out.String()) != version {
		t.Errorf("got %q", out.String())
	}
}

func TestReportNotImplementedExitsWithError(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"report", "../../testdata/plans/basic-create.json"}, strings.NewReader(""), &out, &errb)
	if code != exitError {
		t.Errorf("exit %d, want %d", code, exitError)
	}
}
