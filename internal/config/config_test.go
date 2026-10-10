package config

import (
	"os"
	"path/filepath"
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

func TestParseSettings(t *testing.T) {
	c, err := Parse([]byte("format: [html, markdown]\nfail_on: [delete, \"replace:aws_db_instance\"]\nmax_line: 90\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Format) != 2 || c.Format[1] != "markdown" || len(c.FailOn) != 2 || c.MaxLine != 90 {
		t.Errorf("%+v", c)
	}
	if _, err := Parse([]byte("")); err != nil {
		t.Errorf("an empty file is a valid config: %v", err)
	}
	if _, err := Parse([]byte("# only comments\n")); err != nil {
		t.Errorf("comments only: %v", err)
	}
}

func TestParseRejectsBadValues(t *testing.T) {
	for name, y := range map[string]string{
		"format":   "format: [pdf]\n",
		"negative": "max_line: -5\n",
	} {
		if _, err := Parse([]byte(y)); err == nil {
			t.Errorf("%s should fail", name)
		}
	}
}

func TestDefaultConfigFileName(t *testing.T) {
	if DefaultPath != "tfplanlens.config.yaml" {
		t.Errorf("DefaultPath = %q", DefaultPath)
	}
}
