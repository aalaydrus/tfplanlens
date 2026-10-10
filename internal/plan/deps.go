package plan

import (
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

// DepGraph is the resource-level dependency graph recovered from the plan's
// configuration: references in attribute expressions, count/for_each
// expressions, depends_on, provider configuration, and module inputs/outputs.
//
// Nodes are configuration addresses without instance keys, e.g.
// "aws_instance.web", "data.aws_caller_identity.current" or
// "module.net.aws_vpc.main". Dependencies that pass through module variables,
// module outputs or a whole-module depends_on are resolved to the resources
// behind them.
type DepGraph struct {
	// Deps maps each resource (and data source) to the ones it depends on.
	Deps map[string][]string
	// LocalRefs counts distinct references to local values. The plan JSON does
	// not expand locals, so a dependency that is only reachable through one is
	// not in Deps.
	LocalRefs int
}

type builder struct {
	raw    map[string]map[string]bool // node -> targets
	real   map[string]bool            // resources and data sources
	pseudo map[string]bool            // module variables, outputs and whole modules
	locals map[string]bool
	// providerRefs maps a provider config key to the nodes its configuration
	// references; resources using that provider depend on them.
	providerRefs map[string][]string
}

func buildDeps(cfg *tfjson.Config) *DepGraph {
	b := &builder{
		raw: map[string]map[string]bool{}, real: map[string]bool{}, pseudo: map[string]bool{},
		locals: map[string]bool{}, providerRefs: map[string][]string{},
	}
	if cfg == nil || cfg.RootModule == nil {
		return &DepGraph{Deps: map[string][]string{}}
	}
	for key, pc := range cfg.ProviderConfigs {
		var targets []string
		for _, ref := range refsOf(pc.Expressions) {
			if n, ok := b.resolve("", ref); ok {
				targets = append(targets, n)
			}
		}
		b.providerRefs[key] = targets
	}
	b.walk(cfg.RootModule, "", nil)
	return b.finish()
}

func (b *builder) edge(from, to string) {
	if from == to {
		return
	}
	if b.raw[from] == nil {
		b.raw[from] = map[string]bool{}
	}
	b.raw[from][to] = true
}

// walk adds the resources, outputs and module calls of one module. prefix is
// "" for the root or "module.a.module.b." for nested modules; inherited holds
// nodes (resolved in the parent) that everything in this module depends on.
func (b *builder) walk(mod *tfjson.ConfigModule, prefix string, inherited []string) {
	for _, r := range mod.Resources {
		node := prefix + r.Address
		b.real[node] = true
		var targets []string
		add := func(refs []string) {
			for _, ref := range refs {
				if n, ok := b.resolve(prefix, ref); ok {
					targets = append(targets, n)
				}
			}
		}
		add(refsOf(r.Expressions))
		add(exprRefs(r.CountExpression))
		add(exprRefs(r.ForEachExpression))
		add(r.DependsOn)
		targets = append(targets, inherited...)
		key := r.ProviderConfigKey
		if _, ok := b.providerRefs[key]; !ok {
			if i := strings.LastIndex(key, ":"); i >= 0 {
				key = key[i+1:]
			}
		}
		targets = append(targets, b.providerRefs[key]...)
		for _, t := range targets {
			b.edge(node, t)
		}
	}
	for name, out := range mod.Outputs {
		node := prefix + "output." + name
		b.pseudo[node] = true
		if out == nil {
			continue
		}
		for _, ref := range append(exprRefs(out.Expression), out.DependsOn...) {
			if n, ok := b.resolve(prefix, ref); ok {
				b.edge(node, n)
			}
		}
	}
	for name, call := range mod.ModuleCalls {
		if call == nil || call.Module == nil {
			continue
		}
		child := prefix + "module." + name + "."
		modNode := prefix + "module." + name
		b.pseudo[modNode] = true

		// Module inputs: a variable depends on whatever its argument references.
		for varName, e := range call.Expressions {
			vnode := child + "var." + varName
			b.pseudo[vnode] = true
			for _, ref := range exprRefs(e) {
				if n, ok := b.resolve(prefix, ref); ok {
					b.edge(vnode, n)
				}
			}
		}
		// depends_on, count and for_each on the call apply to everything inside.
		inh := append([]string(nil), inherited...)
		for _, ref := range append(append(append([]string(nil), call.DependsOn...), exprRefs(call.CountExpression)...), exprRefs(call.ForEachExpression)...) {
			if n, ok := b.resolve(prefix, ref); ok {
				inh = append(inh, n)
			}
		}
		b.walk(call.Module, child, inh)

		// "module.x" as a whole stands for everything inside it.
		for _, r := range call.Module.Resources {
			b.edge(modNode, child+r.Address)
		}
		for out := range call.Module.Outputs {
			b.edge(modNode, child+"output."+out)
		}
		for sub := range call.Module.ModuleCalls {
			b.edge(modNode, child+"module."+sub)
		}
	}
}

// resolve turns a reference string into a node id, relative to the module at
// prefix. Locals, each/count/path/terraform/self are not nodes.
func (b *builder) resolve(prefix, ref string) (string, bool) {
	parts := strings.Split(stripIndexes(ref), ".")
	switch parts[0] {
	case "local":
		if len(parts) > 1 {
			b.locals[prefix+"local."+parts[1]] = true
		}
		return "", false
	case "each", "count", "path", "terraform", "self", "":
		return "", false
	case "var":
		if prefix == "" || len(parts) < 2 {
			return "", false
		}
		return prefix + "var." + parts[1], true
	case "module":
		switch {
		case len(parts) >= 3:
			return prefix + "module." + parts[1] + ".output." + parts[2], true
		case len(parts) == 2:
			return prefix + "module." + parts[1], true
		}
		return "", false
	case "data":
		if len(parts) >= 3 {
			return prefix + "data." + parts[1] + "." + parts[2], true
		}
		return "", false
	}
	if len(parts) >= 2 {
		return prefix + parts[0] + "." + parts[1], true
	}
	return "", false
}

// finish expands module variables, outputs and whole-module references so
// that Deps only links real resources.
func (b *builder) finish() *DepGraph {
	g := &DepGraph{Deps: map[string][]string{}, LocalRefs: len(b.locals)}
	for node := range b.real {
		set := map[string]bool{}
		seen := map[string]bool{}
		var expand func(n string)
		expand = func(n string) {
			for t := range b.raw[n] {
				switch {
				case b.real[t]:
					if t != node {
						set[t] = true
					}
				case b.pseudo[t] && !seen[t]:
					seen[t] = true
					expand(t)
				}
			}
		}
		expand(node)
		deps := make([]string, 0, len(set))
		for t := range set {
			deps = append(deps, t)
		}
		sort.Strings(deps)
		g.Deps[node] = deps
	}
	return g
}

func exprRefs(e *tfjson.Expression) []string {
	if e == nil || e.ExpressionData == nil {
		return nil
	}
	refs := append([]string(nil), e.References...)
	for _, blk := range e.NestedBlocks {
		refs = append(refs, refsOf(blk)...)
	}
	return refs
}

func refsOf(m map[string]*tfjson.Expression) []string {
	var refs []string
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		refs = append(refs, exprRefs(m[k])...)
	}
	return refs
}
