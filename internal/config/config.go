// Package config loads tfplanlens.config.yaml.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultPath is read when --config is not given and the file exists.
const DefaultPath = "tfplanlens.config.yaml"

// DefaultMaxLine is the terminal truncation width when max_line is unset.
const DefaultMaxLine = 120

// Formats are the valid output formats.
var Formats = []string{"text", "html", "markdown", "json"}

// Config is the contents of tfplanlens.config.yaml (SPEC 5.9). Every key is
// optional; command-line flags override the file.
type Config struct {
	// TFApplyAccountID is the account terraform apply runs from. When unset it
	// is inferred from the plan's default provider.
	TFApplyAccountID string `yaml:"tfapply_account_id"`
	// Accounts maps a provider alias (or full provider config key such as
	// aws.prod) to its account ID.
	Accounts map[string]string `yaml:"accounts"`

	// Format is the default output format(s); the --format flag overrides it.
	Format []string `yaml:"format"`
	// FailOn is the default --fail-on rules; the --fail-on flag overrides it.
	FailOn []string `yaml:"fail_on"`
	// MaxLine is the terminal truncation width (default 120).
	MaxLine int `yaml:"max_line"`

	// RedactSecrets hides values that look like secrets but are not marked
	// sensitive by Terraform (default true). DecodeValues decodes base64, gzip
	// and JSON in long values before diffing them (default true).
	RedactSecrets *bool `yaml:"redact_secrets"`
	DecodeValues  *bool `yaml:"decode_values"`

	StatefulTypes []string `yaml:"stateful_types"`
	LongValueKeys []string `yaml:"long_value_keys"`
	IgnoreTypes   []string `yaml:"ignore_types"`
}

// RedactSecretsEnabled reports whether secret-looking values are hidden (the
// default; only an explicit redact_secrets: false turns it off).
func (c *Config) RedactSecretsEnabled() bool {
	return c == nil || c.RedactSecrets == nil || *c.RedactSecrets
}

// DecodeValuesEnabled reports whether long values are decoded before diffing
// (the default; only an explicit decode_values: false turns it off).
func (c *Config) DecodeValuesEnabled() bool {
	return c == nil || c.DecodeValues == nil || *c.DecodeValues
}

// Load reads path. An empty path means DefaultPath, which is optional; an
// explicitly given path must exist.
func Load(path string) (*Config, error) {
	explicit := path != ""
	if !explicit {
		path = DefaultPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !explicit && errors.Is(err, fs.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	c, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// Parse decodes config YAML, rejecting unknown keys so typos are caught, and
// validates the values.
func Parse(data []byte) (*Config, error) {
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &c, c.validate()
}

func (c *Config) validate() error {
	for _, f := range c.Format {
		if !contains(Formats, f) {
			return fmt.Errorf("format %q is not one of %v", f, Formats)
		}
	}
	if c.MaxLine < 0 {
		return fmt.Errorf("max_line must not be negative, got %d", c.MaxLine)
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
