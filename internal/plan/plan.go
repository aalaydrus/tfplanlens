package plan

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

// Action is the normalized action of a resource change.
type Action string

const (
	ActionCreate  Action = "create"
	ActionUpdate  Action = "update"
	ActionReplace Action = "replace"
	ActionDelete  Action = "delete"
	ActionRead    Action = "read"
	ActionNoOp    Action = "no-op"
	// ActionForget is a removed block with destroy = false: the object leaves
	// state but is not destroyed.
	ActionForget Action = "forget"
)

// Mode describes what kind of plan this is.
type Mode string

const (
	ModeNormal      Mode = "normal"
	ModeRefreshOnly Mode = "refresh-only"
	ModeDestroy     Mode = "destroy"
)

// Plan is the normalized plan consumed by analyze.
type Plan struct {
	FormatVersion    string
	TerraformVersion string
	Mode             Mode
	Applyable        bool
	Errored          bool
	Changes          []Change
	Drift            []Change
	// Providers is keyed by provider config key, e.g. "aws" or "aws.prod".
	Providers map[string]Provider
}

// Provider is a provider configuration from the plan's configuration block.
type Provider struct {
	Key   string
	Name  string // "aws"
	Alias string // "" for the default configuration
	// AssumeRoleARN is the literal assume_role.role_arn, or "" if absent or
	// not a literal.
	AssumeRoleARN string
}

// Change is one resource instance change.
type Change struct {
	Address         string
	ModuleAddress   string
	Mode            string // "managed" or "data"
	Type            string
	Name            string
	ProviderName    string
	ProviderKey     string // provider config key, resolved from configuration
	Index           string
	HasIndex        bool
	Action          Action
	Reason          string // raw action_reason
	ReplacePaths    []string
	PreviousAddress string
	Importing       bool
	ImportID        string

	Before          any
	After           any
	BeforeSensitive any
	AfterSensitive  any
	AfterUnknown    any
}

// Load reads and normalizes a `terraform show -json` plan.
func Load(r io.Reader) (*Plan, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading plan: %w", err)
	}
	var tp tfjson.Plan
	if err := json.Unmarshal(raw, &tp); err != nil {
		return nil, fmt.Errorf("parsing plan JSON: %w", err)
	}
	if err := tp.Validate(); err != nil {
		return nil, fmt.Errorf("unsupported plan: %w", err)
	}

	// Fields terraform-json lacks.
	var extra struct {
		Mode      string `json:"mode"`
		Applyable *bool  `json:"applyable"`
		Errored   bool   `json:"errored"`
		Changes   []struct {
			Address string `json:"address"`
			Deposed string `json:"deposed"`
			Change  struct {
				ReplacePaths [][]any `json:"replace_paths"`
			} `json:"change"`
		} `json:"resource_changes"`
	}
	if err := json.Unmarshal(raw, &extra); err != nil {
		return nil, fmt.Errorf("parsing plan JSON: %w", err)
	}

	p := &Plan{
		FormatVersion:    tp.FormatVersion,
		TerraformVersion: tp.TerraformVersion,
		Applyable:        true,
		Errored:          extra.Errored,
		Providers:        providers(&tp),
	}
	if extra.Applyable != nil {
		p.Applyable = *extra.Applyable
	}
	cfgKeys := resourceProviderKeys(&tp)

	replacePaths := map[string][]string{}
	for _, c := range extra.Changes {
		var paths []string
		for _, rp := range c.Change.ReplacePaths {
			paths = append(paths, formatPath(rp))
		}
		replacePaths[c.Address+"|"+c.Deposed] = paths
	}

	for _, rc := range tp.ResourceChanges {
		ch := normalize(rc, cfgKeys)
		ch.ReplacePaths = replacePaths[rc.Address+"|"+rc.DeposedKey]
		p.Changes = append(p.Changes, ch)
	}
	for _, rc := range tp.ResourceDrift {
		p.Drift = append(p.Drift, normalize(rc, cfgKeys))
	}
	p.Mode = detectMode(extra.Mode, p)
	return p, nil
}

func normalize(rc *tfjson.ResourceChange, cfgKeys map[string]string) Change {
	c := Change{
		Address:         rc.Address,
		ModuleAddress:   rc.ModuleAddress,
		Mode:            string(rc.Mode),
		Type:            rc.Type,
		Name:            rc.Name,
		ProviderName:    rc.ProviderName,
		Reason:          string(rc.ActionReason),
		PreviousAddress: rc.PreviousAddress,
	}
	switch v := rc.Index.(type) {
	case nil:
	case float64:
		c.HasIndex, c.Index = true, strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		c.HasIndex, c.Index = true, v
	default:
		c.HasIndex, c.Index = true, fmt.Sprint(v)
	}
	if rc.Change != nil {
		c.Action = action(rc.Change.Actions)
		c.Before, c.After = rc.Change.Before, rc.Change.After
		c.BeforeSensitive, c.AfterSensitive = rc.Change.BeforeSensitive, rc.Change.AfterSensitive
		c.AfterUnknown = rc.Change.AfterUnknown
		if rc.Change.Importing != nil {
			c.Importing, c.ImportID = true, rc.Change.Importing.ID
		}
	}
	c.ProviderKey = cfgKeys[configAddress(rc)]
	if c.ProviderKey == "" {
		c.ProviderKey = defaultProviderKey(rc.ProviderName)
	}
	return c
}

func action(a tfjson.Actions) Action {
	switch {
	case a.Replace():
		return ActionReplace
	case a.Delete():
		return ActionDelete
	case a.Create():
		return ActionCreate
	case a.Update():
		return ActionUpdate
	case a.Read():
		return ActionRead
	case a.Forget():
		return ActionForget
	default:
		return ActionNoOp
	}
}

// detectMode uses the plan's own "mode" when present, otherwise infers it.
func detectMode(raw string, p *Plan) Mode {
	switch raw {
	case "destroy":
		return ModeDestroy
	case "refresh-only":
		return ModeRefreshOnly
	case "normal":
		return ModeNormal
	}
	managed, deletes, actionable := 0, 0, 0
	for _, c := range p.Changes {
		if c.Mode != "managed" {
			continue
		}
		managed++
		switch c.Action {
		case ActionDelete:
			deletes++
			actionable++
		case ActionCreate, ActionUpdate, ActionReplace, ActionForget:
			actionable++
		}
	}
	switch {
	case managed > 0 && deletes == managed:
		return ModeDestroy
	case actionable == 0 && len(p.Drift) > 0:
		return ModeRefreshOnly
	}
	return ModeNormal
}

func defaultProviderKey(providerName string) string {
	if i := strings.LastIndex(providerName, "/"); i >= 0 {
		return providerName[i+1:]
	}
	return providerName
}

// configAddress is the address of a resource's configuration block: module
// instance keys and resource index removed.
func configAddress(rc *tfjson.ResourceChange) string {
	addr := rc.Type + "." + rc.Name
	if rc.Mode == tfjson.DataResourceMode {
		addr = "data." + addr
	}
	if rc.ModuleAddress != "" {
		addr = stripIndexes(rc.ModuleAddress) + "." + addr
	}
	return addr
}

func stripIndexes(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '[':
			depth++
		case r == ']':
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func resourceProviderKeys(tp *tfjson.Plan) map[string]string {
	m := map[string]string{}
	if tp.Config == nil {
		return m
	}
	var walk func(mod *tfjson.ConfigModule, prefix string)
	walk = func(mod *tfjson.ConfigModule, prefix string) {
		if mod == nil {
			return
		}
		for _, r := range mod.Resources {
			key := r.ProviderConfigKey
			// Module-scoped keys look like "net:aws"; fall back to the root config.
			if _, ok := tp.Config.ProviderConfigs[key]; !ok {
				if i := strings.LastIndex(key, ":"); i >= 0 {
					key = key[i+1:]
				}
			}
			m[prefix+r.Address] = key
		}
		for name, call := range mod.ModuleCalls {
			if call != nil {
				walk(call.Module, prefix+"module."+name+".")
			}
		}
	}
	walk(tp.Config.RootModule, "")
	return m
}

func providers(tp *tfjson.Plan) map[string]Provider {
	out := map[string]Provider{}
	if tp.Config == nil {
		return out
	}
	for key, pc := range tp.Config.ProviderConfigs {
		out[key] = Provider{Key: key, Name: pc.Name, Alias: pc.Alias, AssumeRoleARN: literalRoleARN(pc)}
	}
	return out
}

func literalRoleARN(pc *tfjson.ProviderConfig) string {
	ar := pc.Expressions["assume_role"]
	if ar == nil || ar.ExpressionData == nil {
		return ""
	}
	for _, blk := range ar.NestedBlocks {
		if e := blk["role_arn"]; e != nil && e.ExpressionData != nil {
			if s, ok := e.ConstantValue.(string); ok {
				return s
			}
		}
	}
	return ""
}

func formatPath(p []any) string {
	var b strings.Builder
	for i, seg := range p {
		switch v := seg.(type) {
		case float64:
			fmt.Fprintf(&b, "[%d]", int(v))
		default:
			if i > 0 {
				b.WriteByte('.')
			}
			fmt.Fprint(&b, v)
		}
	}
	return b.String()
}

// SortedProviderKeys returns provider config keys in stable order.
func (p *Plan) SortedProviderKeys() []string {
	keys := make([]string, 0, len(p.Providers))
	for k := range p.Providers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
