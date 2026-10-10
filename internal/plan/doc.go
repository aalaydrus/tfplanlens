// Package plan loads `terraform show -json` output and normalizes it into the
// internal model. Fields missing from terraform-json (for example
// replace_paths) are read from the raw JSON.
package plan
