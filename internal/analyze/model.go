package analyze

import (
	"encoding/json"
	"fmt"

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
	// Parents groups the instances of count/for_each resources.
	Parents []Parent `json:"parents"`
	// Waves layer the changing resources by dependency depth: possible
	// parallelism, not a strict order. WaveNotes list what the plan JSON
	// cannot tell us. Cycles lists resources that depend on each other.
	Waves     []Wave   `json:"waves"`
	WaveNotes []string `json:"wave_notes"`
	Cycles    []Cycle  `json:"cycles"`
	// Units lists the plans of a merged report (one per root module). It is empty for a single plan.
	Units []Unit `json:"units"`
}

// Unit is one plan of a merged report.
type Unit struct {
	Label            string       `json:"label"`
	Mode             string       `json:"mode"`
	TerraformVersion string       `json:"terraform_version,omitempty"`
	ApplyAccount     ApplyAccount `json:"apply_account"`
	Actions          ActionCounts `json:"actions"`
}

// Account sources, recorded wherever an account is resolved.
const (
	SourceFlag             = "flag"
	SourceEnv              = "env"
	SourceConfig           = "config"
	SourceRoleARN          = "role_arn"
	SourceRoleARNVariable  = "role_arn_variable"
	SourceAllowedAccountID = "allowed_account_ids"
	SourceCallerIdentity   = "caller_identity"
	SourceApplyAccount     = "apply_account"
	SourceUnresolved       = "unresolved"
	// SourceInferred marks an apply account taken from the plan's default
	// provider; ApplyAccount.Via says which source resolved that provider.
	SourceInferred = "inferred"
	// SourceMixed marks the apply account of a merged report whose plans
	// run from different accounts.
	SourceMixed = "mixed"
)

// ApplyAccount is the account terraform apply runs from (SPEC 5.5).
type ApplyAccount struct {
	ID       string `json:"id,omitempty"`
	Source   string `json:"source"` // flag | env | config | inferred | unresolved
	Resolved bool   `json:"resolved"`
	// Via is, for an inferred account, how the default provider's account was
	// found (config, allowed_account_ids or caller_identity).
	Via string `json:"via,omitempty"`
}

// Describe says where the apply account came from, e.g. "source: flag" or
// "source: inferred from the default provider via caller_identity".
func (a ApplyAccount) Describe() string {
	if a.Source == SourceMixed {
		return "source: mixed; see each plan under Plans"
	}
	if a.Source == SourceInferred {
		return "source: inferred from the default provider via " + a.Via
	}
	return "source: " + a.Source
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
	// ApplyWaves and DestroyWaves are the number of waves in each phase.
	ApplyWaves   int `json:"apply_waves"`
	DestroyWaves int `json:"destroy_waves"`

	// Hidden counts changes removed by ignore_types.
	Hidden int `json:"hidden,omitempty"`
	// Redacted counts values hidden because they look like secrets that
	// Terraform did not mark sensitive (see internal/redact).
	Redacted int `json:"redacted,omitempty"`
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
	// Unit is the plan this finding came from in a merged report.
	Unit string `json:"unit,omitempty"`
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
	// Unit is the plan this drift came from in a merged report.
	Unit string `json:"unit,omitempty"`
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
	// Parent and Key identify a count/for_each instance: its configuration
	// address and index key.
	Parent string `json:"parent,omitempty"`
	Key    string `json:"key,omitempty"`
	// Wave is the 1-based wave of this change within WavePhase.
	Wave      int    `json:"wave,omitempty"`
	WavePhase string `json:"wave_phase,omitempty"`
	// Unit is the plan this change came from in a merged report.
	Unit string `json:"unit,omitempty"`
}

// Listed reports whether a change is worth listing in a report: it does
// something, is moved or imported, or carries a warning. Plain no-ops are
// counted but not listed.
func (c Change) Listed() bool {
	return c.Action != "no-op" || c.MovedFrom != "" || c.Import || len(c.Warnings) > 0
}

// Label describes what happens to the resource in a few words.
func (c Change) Label() string {
	label := c.Action
	switch {
	case c.StateOnly:
		label = "removed from state only"
	case c.Import && c.Action == "no-op":
		label = "import"
	case c.Import:
		label = "import + " + c.Action
	}
	if c.MovedFrom != "" {
		if c.Action == "no-op" {
			label = "moved from " + c.MovedFrom
		} else {
			label += ", moved from " + c.MovedFrom
		}
	}
	if c.KnownAfterApply > 0 {
		label += fmt.Sprintf(", %d known after apply", c.KnownAfterApply)
	}
	return label
}
