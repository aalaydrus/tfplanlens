package plan

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
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
	// Graph is the dependency graph recovered from the configuration.
	Graph *DepGraph
}

// Role ARN sources recorded on Provider.RoleARNVia.
const (
	ViaLiteral  = "literal"
	ViaVariable = "variable"
)

// Provider is a provider configuration from the plan's configuration block.
type Provider struct {
	Key   string
	Name  string // "aws"
	Alias string // "" for the default configuration
	// AssumeRole is true if the provider has an assume_role block (or
	// assume_role_with_web_identity), whether or not its ARN is known.
	AssumeRole bool
	// AssumeRoleARN is assume_role.role_arn when it is a literal or resolves
	// through the plan's variables; "" otherwise. RoleARNVia says which.
	AssumeRoleARN string
	RoleARNVia    string
	// AllowedAccountIDs is allowed_account_ids when it is a literal or
	// resolves through the plan's variables.
	AllowedAccountIDs []string
	// CallerIdentityIDs are the distinct account_id values of
	// aws_caller_identity data sources in prior_state that use this provider.
	CallerIdentityIDs []string
}

// Change is one resource instance change.
type Change struct {
	Address string
	// ConfigAddress is Address without instance keys: the resource's block in
	// the configuration, e.g. "module.net.aws_vpc.main".
	ConfigAddress   string
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
	}
	cfgKeys := resourceProviderKeys(&tp)
	p.Providers = providers(&tp, cfgKeys)
	p.Graph = buildDeps(tp.Config)
	if extra.Applyable != nil {
		p.Applyable = *extra.Applyable
	}

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
		ConfigAddress:   configAddress(rc),
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

func providers(tp *tfjson.Plan, cfgKeys map[string]string) map[string]Provider {
	out := map[string]Provider{}
	if tp.Config == nil {
		return out
	}
	vars := map[string]any{}
	for name, v := range tp.Variables {
		if v != nil {
			vars[name] = v.Value
		}
	}
	for key, pc := range tp.Config.ProviderConfigs {
		pr := Provider{Key: key, Name: pc.Name, Alias: pc.Alias}
		for _, blockName := range []string{"assume_role", "assume_role_with_web_identity"} {
			ar := pc.Expressions[blockName]
			if ar == nil || ar.ExpressionData == nil {
				continue
			}
			pr.AssumeRole = true
			for _, blk := range ar.NestedBlocks {
				if e := blk["role_arn"]; e != nil && pr.AssumeRoleARN == "" {
					if v, via, ok := resolve(e, vars); ok {
						if s, isStr := v.(string); isStr {
							pr.AssumeRoleARN, pr.RoleARNVia = s, via
						}
					}
				}
			}
		}
		if e := pc.Expressions["allowed_account_ids"]; e != nil {
			if v, _, ok := resolve(e, vars); ok {
				if list, isList := v.([]any); isList {
					for _, x := range list {
						if s, isStr := x.(string); isStr {
							pr.AllowedAccountIDs = append(pr.AllowedAccountIDs, s)
						}
					}
				}
			}
		}
		out[key] = pr
	}
	for key, ids := range callerIdentities(tp, cfgKeys) {
		if pr, ok := out[key]; ok {
			pr.CallerIdentityIDs = ids
			out[key] = pr
		} else {
			out[key] = Provider{Key: key, Name: defaultProviderKey(key), CallerIdentityIDs: ids}
		}
	}
	return out
}

// resolve returns an expression's value if it is a constant or a plain
// variable reference (var.x) whose value is in the plan.
func resolve(e *tfjson.Expression, vars map[string]any) (value any, via string, ok bool) {
	if e == nil || e.ExpressionData == nil {
		return nil, "", false
	}
	// terraform-json stores a sentinel in ConstantValue whenever an expression
	// has references, so references must be checked first.
	if len(e.References) == 0 {
		if e.ConstantValue == nil {
			return nil, "", false
		}
		return e.ConstantValue, ViaLiteral, true
	}
	// Terraform lists both "var.x" and its attribute paths; accept only
	// references that all point at one variable.
	name := ""
	for _, ref := range e.References {
		base, ok := strings.CutPrefix(ref, "var.")
		if !ok {
			return nil, "", false
		}
		base, _, _ = strings.Cut(base, ".")
		base, _, _ = strings.Cut(base, "[")
		if name != "" && base != name {
			return nil, "", false
		}
		name = base
	}
	if name == "" {
		return nil, "", false
	}
	v, found := vars[name]
	if !found || v == nil {
		return nil, "", false
	}
	return v, ViaVariable, true
}

// callerIdentities maps provider config key to the account IDs reported by
// aws_caller_identity data sources in prior_state. Resource-to-provider
// links come from configuration, because prior_state carries no alias.
func callerIdentities(tp *tfjson.Plan, cfgKeys map[string]string) map[string][]string {
	out := map[string][]string{}
	if tp.PriorState == nil || tp.PriorState.Values == nil {
		return out
	}
	var walk func(m *tfjson.StateModule)
	walk = func(m *tfjson.StateModule) {
		if m == nil {
			return
		}
		for _, r := range m.Resources {
			if r.Mode != tfjson.DataResourceMode || r.Type != "aws_caller_identity" {
				continue
			}
			id, _ := r.AttributeValues["account_id"].(string)
			if id == "" {
				continue
			}
			key := cfgKeys[stripIndexes(r.Address)]
			if key == "" {
				key = defaultProviderKey(r.ProviderName)
			}
			if !slices.Contains(out[key], id) {
				out[key] = append(out[key], id)
			}
		}
		for _, c := range m.ChildModules {
			walk(c)
		}
	}
	walk(tp.PriorState.Values.RootModule)
	return out
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
