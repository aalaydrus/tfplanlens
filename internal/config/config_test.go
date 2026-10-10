package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAccountIDsKeepLeadingZeros(t *testing.T) {
	// Unquoted IDs are YAML integers; they must still come through as the
	// literal 12-digit text, leading zeros included.
	c, err := Parse([]byte("tfapply_account_id: 012345678901\naccounts:\n  a: 000000000001\n  b: \"222222222222\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.TFApplyAccountID != "012345678901" || c.Accounts["a"] != "000000000001" || c.Accounts["b"] != "222222222222" {
		t.Errorf("%+v", c)
	}
}

func TestParseRejectsUnknownKeys(t *testing.T) {
	if _, err := Parse([]byte("tfapply_account: 123456789012\n")); err == nil {
		t.Error("typo in key should be an error")
	}
}

func TestLoadDefaultMissingIsEmptyButExplicitMissingFails(t *testing.T) {
	// t.Chdir needs Go 1.24; go.mod targets 1.23.
	old, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	if c, err := Load(""); err != nil || c == nil {
		t.Errorf("default missing: %v", err)
	}
	if _, err := Load(filepath.Join("nope", "x.yaml")); err == nil {
		t.Error("explicit missing file should fail")
	}
}

func TestSaveApplyAccountCreatesAndPreserves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.yaml")
	if err := SaveApplyAccount(path, "123456789012"); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil || c.TFApplyAccountID != "123456789012" {
		t.Fatalf("%+v %v", c, err)
	}

	// No trailing newline in the existing file must not glue lines together.
	path = filepath.Join(t.TempDir(), "d.yaml")
	if err := os.WriteFile(path, []byte("# keep me\nignore_types: [aws_x]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveApplyAccount(path, "123456789012"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	c, err = Load(path)
	if err != nil || c.TFApplyAccountID != "123456789012" || len(c.IgnoreTypes) != 1 || !strings.HasPrefix(string(raw), "# keep me") {
		t.Errorf("%q %+v %v", raw, c, err)
	}
}
