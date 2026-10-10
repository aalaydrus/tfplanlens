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
const DefaultPath = ".tfplanlens.yaml"

// Config is the contents of .tfplanlens.yaml (SPEC 5.9).
type Config struct {
	// Accounts maps a provider alias (or full provider config key) to an
	// account ID.
	Accounts map[string]string `yaml:"accounts"`
	// TFApplyAccountID is the account terraform apply runs from.
	TFApplyAccountID string   `yaml:"tfapply_account_id"`
	StatefulTypes    []string `yaml:"stateful_types"`
	LongValueKeys    []string `yaml:"long_value_keys"`
	IgnoreTypes      []string `yaml:"ignore_types"`
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
	return Parse(data)
}

// Parse decodes config YAML, rejecting unknown keys so typos are caught.
func Parse(data []byte) (*Config, error) {
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &c, nil
}

// SaveApplyAccount records tfapply_account_id in the config file at path,
// creating it if needed. Existing content, comments included, is preserved.
func SaveApplyAccount(path, id string) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, fmt.Sprintf("tfapply_account_id: %q\n", id)...)
	return os.WriteFile(path, data, 0o644)
}
