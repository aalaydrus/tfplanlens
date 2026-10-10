package analyze

import (
	"encoding/json"

	"github.com/aalaydrus/tfplanlens/internal/valuediff"
)

// SchemaVersion is the version of the JSON form of Analysis.
const SchemaVersion = 1

// Analysis is the single input to every renderer (SPEC section 6). Renderers
// never see raw plan JSON. Slices are never nil so JSON output has [].
type Analysis struct {
	SchemaVersion int `json:"schema_version"`
	// ApplyAccount is where terraform apply runs from ("executed from").
	ApplyAccount ApplyAccount `json:"apply_account"`
	// Notices are plan-wide warnings, e.g. an unresolved apply account.
	Notices  []string    `json:"notices"`
	Summary  Summary     `json:"summary"`
	Findings []Finding   `json:"findings"`
	Drift    []DriftItem `json:"drift"`
	Groups   []Group     `json:"groups"`
	Changes  []Change    `json:"changes"`
}

// Account sources, recorded wherever an account is resolved.
const (
	SourceFlag             = "flag"
	SourceEnv              = "env"
	SourcePrompt           = "prompt"
	SourceConfig           = "config"
	SourceRoleARN          = "role_arn"
	SourceRoleARNVariable  = "role_arn_variable"
	SourceAllowedAccountID = "allowed_account_ids"
	SourceCallerIdentity   = "caller_identity"
	SourceApplyAccount     = "apply_account"
	SourceUnresolved       = "unresolved"
)

// ApplyAccount is the account terraform apply runs from (SPEC 5.5).
type ApplyAccount struct {
	ID       string `json:"id,omitempty"`
	Source   string `json:"source"` // flag | env | config | prompt | unresolved
	Resolved bool   `json:"resolved"`
}

// Warning is a per-resource warning.
type Warning struct {
	Kind    string `json:"kind"` // arn_account_mismatch
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// ActionCounts counts changes by action.
type ActionCounts struct {
	Create  int `json:"create"`
	Update  int `json:"update"`
	Replace int `json:"replace"`
	Delete  int `json:"delete"`
	Read    int `json:"read"`
	NoOp    int `json:"no_op"`
	Forget  int `json:"forget"`
}

// Summary holds plan-wide counts (SPEC 5.1).
type Summary struct {
	Mode             string         `json:"mode"` // normal | refresh-only | destroy
	Applyable        bool           `json:"applyable"`
	Errored          bool           `json:"errored"`
	TerraformVersion string         `json:"terraform_version,omitempty"`
	Total            int            `json:"total"`
	Actions          ActionCounts   `json:"actions"`
	ByType           map[string]int `json:"by_type"`
	ByModule         map[string]int `json:"by_module"` // "root" for the root module
	// KnownAfterApply is a count of attributes, never listed.
	KnownAfterApply int `json:"known_after_apply"`
	DriftCount      int `json:"drift_count"`
	// Warnings counts per-resource warnings.
	Warnings int `json:"warnings"`
	// Hidden counts changes removed by ignore_types.
	Hidden int `json:"hidden,omitempty"`
}

// Finding is a danger finding: a delete or replace (SPEC 5.2).
type Finding struct {
	Address       string   `json:"address"`
	Type          string   `json:"type"`
	Action        string   `json:"action"` // delete | replace
	Provider      string   `json:"provider"`
	Account       string   `json:"account"`
	AccountSource string   `json:"account_source"`
	Stateful      bool     `json:"stateful"`
	Reason        string   `json:"reason"`      // raw action_reason
	ReasonText    string   `json:"reason_text"` // plain English
	ReplacePaths  []string `json:"replace_paths,omitempty"`
	// IndexShift is set for deletes that look like a count/for_each shift.
	IndexShift bool   `json:"index_shift"`
	Warning    string `json:"warning,omitempty"`
	// DriftAlso is set when the resource also drifted (high risk).
	DriftAlso bool `json:"drift_also"`
}

// Attr is one changed attribute.
type Attr struct {
	Path string `json:"path"`
	// Kind is value | sensitive | unknown | long.
	Kind          string            `json:"kind"`
	Before        json.RawMessage   `json:"before,omitempty"`
	After         json.RawMessage   `json:"after,omitempty"`
	ForcesReplace bool              `json:"forces_replacement,omitempty"`
	Diff          *valuediff.Result `json:"diff,omitempty"`
}

// DriftItem is a change made outside Terraform (SPEC 5.3).
type DriftItem struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Action  string `json:"action"` // update | delete
	Attrs   []Attr `json:"attrs"`
	// HighRisk is set when the resource is also replaced or deleted.
	HighRisk bool   `json:"high_risk"`
	Note     string `json:"note,omitempty"`
}

// Group is a provider-config grouping of changes (SPEC 5.5).
type Group struct {
	Key      string `json:"key"` // provider config key, e.g. aws.prod
	Provider string `json:"provider"`
	Alias    string `json:"alias,omitempty"`
	Account  string `json:"account"` // ID or "unresolved (alias: x)"
	Resolved bool   `json:"resolved"`
	Source   string `json:"source"`  // see Source* constants
	Default  bool   `json:"default"` // the plan's default provider
	// Cross is set when the target account differs from the apply account.
	Cross     bool         `json:"cross_account"`
	Counts    ActionCounts `json:"counts"`
	Addresses []string     `json:"addresses"`
}

// Change is a single resource instance change.
type Change struct {
	Address         string    `json:"address"`
	Type            string    `json:"type"`
	Module          string    `json:"module"`
	Mode            string    `json:"mode"`
	Action          string    `json:"action"`
	Provider        string    `json:"provider"` // provider config key
	Account         string    `json:"account"`  // target account
	AccountSource   string    `json:"account_source"`
	CrossAccount    bool      `json:"cross_account"` // target differs from apply account
	Warnings        []Warning `json:"warnings"`
	Stateful        bool      `json:"stateful"`
	Reason          string    `json:"reason,omitempty"`
	ReasonText      string    `json:"reason_text,omitempty"`
	ReplacePaths    []string  `json:"replace_paths,omitempty"`
	MovedFrom       string    `json:"moved_from,omitempty"`
	Import          bool      `json:"import"`
	ImportID        string    `json:"import_id,omitempty"`
	StateOnly       bool      `json:"removed_from_state_only"`
	Attrs           []Attr    `json:"attrs"`
	KnownAfterApply int       `json:"known_after_apply"`
}
