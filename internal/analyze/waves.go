package analyze

import (
	"fmt"
	"sort"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// Wave phases.
const (
	PhaseApply   = "apply"
	PhaseDestroy = "destroy"
)

// Wave is a set of resources that have no unfinished dependency among the
// others at that point: they could run in parallel. Terraform does apply in
// parallel, so this shows possible parallelism, not a strict order.
type Wave struct {
	Phase     string         `json:"phase"` // apply | destroy
	Number    int            `json:"number"`
	Resources []WaveResource `json:"resources"`
}

// WaveResource is one resource (all its changing instances) in a wave.
type WaveResource struct {
	Address string `json:"address"` // configuration address, no instance keys
	// Action is the most significant action among the instances:
	// replace, delete, create, update or read.
	Action string `json:"action"`
	// Actions counts the changing instances by action (a resource can create
	// one key and delete another).
	Actions   map[string]int `json:"actions"`
	Instances int            `json:"instances"`
	// NoConfiguration is set for a resource removed from the configuration:
	// the plan has no dependency information for it, so its destroy order is
	// unknown.
	NoConfiguration bool `json:"no_configuration,omitempty"`
}

// Cycle is a set of resources that depend on each other. Terraform rejects
// real cycles at plan time, so this only shows up in incomplete or hand-made
// plans; the cycle's members are placed in one wave.
type Cycle struct {
	Resources []string `json:"resources"`
}

type nodeInfo struct {
	actions   map[string]int
	instances int
	phase     string // apply | destroy
}

type waveAssign struct {
	phase string
	wave  int
}

var actionRank = map[string]int{"replace": 5, "delete": 4, "create": 3, "update": 2, "read": 1}

// computeWaves layers the changing resources by dependency depth.
func computeWaves(p *plan.Plan) (waves []Wave, cycles []Cycle, notes []string, assign map[string]waveAssign) {
	deps := map[string][]string{}
	if p.Graph != nil {
		deps = p.Graph.Deps
	}

	// Changing resources, per configuration address.
	nodes := map[string]*nodeInfo{}
	var order []string
	for _, c := range p.Changes {
		switch c.Action {
		case plan.ActionCreate, plan.ActionUpdate, plan.ActionReplace, plan.ActionRead, plan.ActionDelete:
		default:
			continue
		}
		n := nodes[c.ConfigAddress]
		if n == nil {
			n = &nodeInfo{actions: map[string]int{}, phase: PhaseDestroy}
			nodes[c.ConfigAddress] = n
			order = append(order, c.ConfigAddress)
		}
		n.actions[string(c.Action)]++
		n.instances++
		if c.Action != plan.ActionDelete {
			n.phase = PhaseApply
		}
	}

	// Strongly connected components: cycles are collapsed into one node.
	all := map[string]bool{}
	for k, ds := range deps {
		all[k] = true
		for _, d := range ds {
			all[d] = true
		}
	}
	comp, comps := tarjan(all, deps)
	for _, members := range comps {
		if len(members) > 1 {
			sort.Strings(members)
			cycles = append(cycles, Cycle{Resources: members})
		}
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].Resources[0] < cycles[j].Resources[0] })

	// Component-level graph and activity.
	compDeps := map[int]map[int]bool{}
	compRev := map[int]map[int]bool{}
	for k, ds := range deps {
		for _, d := range ds {
			a, b := comp[k], comp[d]
			if a == b {
				continue
			}
			if compDeps[a] == nil {
				compDeps[a] = map[int]bool{}
			}
			compDeps[a][b] = true
			if compRev[b] == nil {
				compRev[b] = map[int]bool{}
			}
			compRev[b][a] = true
		}
	}
	activeApply, activeDestroy := map[int]bool{}, map[int]bool{}
	for addr, n := range nodes {
		c, ok := comp[addr]
		if !ok {
			continue
		}
		if n.phase == PhaseApply {
			activeApply[c] = true
		} else {
			activeDestroy[c] = true
		}
	}

	// level(c): wave of an active component, otherwise the level of the work
	// that must finish before anything depending on it can start.
	layer := func(edges map[int]map[int]bool, active map[int]bool) func(int) int {
		memo := map[int]int{}
		var level func(c int) int
		level = func(c int) int {
			if v, ok := memo[c]; ok {
				return v
			}
			memo[c] = 0 // the condensation is acyclic; this only guards bugs
			ready := 0
			for d := range edges[c] {
				if l := level(d); l > ready {
					ready = l
				}
			}
			if active[c] {
				ready++
			}
			memo[c] = ready
			return ready
		}
		return level
	}
	applyLevel := layer(compDeps, activeApply)    // dependencies first
	destroyLevel := layer(compRev, activeDestroy) // dependents first

	assign = map[string]waveAssign{}
	byWave := map[string]map[int][]WaveResource{PhaseApply: {}, PhaseDestroy: {}}
	noConfig := false
	for _, addr := range order {
		n := nodes[addr]
		c, known := comp[addr]
		var w int
		switch {
		case n.phase == PhaseApply && known:
			w = applyLevel(c)
		case n.phase == PhaseApply:
			w = 1
		case known:
			w = destroyLevel(c)
		default:
			w = 1
		}
		wr := WaveResource{Address: addr, Action: topAction(n.actions), Actions: n.actions, Instances: n.instances}
		if n.phase == PhaseDestroy && !known {
			wr.NoConfiguration = true
			noConfig = true
		}
		assign[addr] = waveAssign{phase: n.phase, wave: w}
		byWave[n.phase][w] = append(byWave[n.phase][w], wr)
	}
	for _, phase := range []string{PhaseApply, PhaseDestroy} {
		nums := make([]int, 0, len(byWave[phase]))
		for w := range byWave[phase] {
			nums = append(nums, w)
		}
		sort.Ints(nums)
		for _, w := range nums {
			rs := byWave[phase][w]
			sort.Slice(rs, func(i, j int) bool { return rs[i].Address < rs[j].Address })
			waves = append(waves, Wave{Phase: phase, Number: w, Resources: rs})
		}
	}

	// What the plan JSON cannot tell us.
	if p.Graph != nil && p.Graph.LocalRefs > 0 {
		notes = append(notes, fmt.Sprintf("%d reference(s) to local values are not visible in the plan JSON, so a dependency reached only through a local is missing and a resource can show in an earlier wave than it will run.", p.Graph.LocalRefs))
	}
	for _, c := range p.Changes {
		if c.Reason == "replace_by_triggers" {
			notes = append(notes, "replace_triggered_by is not part of the plan JSON: a resource replaced by a trigger is not ordered after the resource that triggered it.")
			break
		}
	}
	if noConfig {
		notes = append(notes, "Resources removed from the configuration have no dependency information in the plan; their destroy order is unknown and they are shown in wave 1.")
	}
	return waves, cycles, notes, assign
}

func topAction(actions map[string]int) string {
	best, rank := "", 0
	for a := range actions {
		if r := actionRank[a]; r > rank || (r == rank && a < best) {
			best, rank = a, r
		}
	}
	return best
}

// tarjan finds strongly connected components. comp maps each node to its
// component id; comps lists the members per component.
func tarjan(nodes map[string]bool, edges map[string][]string) (comp map[string]int, comps [][]string) {
	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	index, low := map[string]int{}, map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	comp = map[string]int{}
	counter := 0
	var strong func(v string)
	strong = func(v string) {
		counter++
		index[v], low[v] = counter, counter
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range edges[v] {
			if index[w] == 0 {
				strong(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if onStack[w] && index[w] < low[v] {
				low[v] = index[w]
			}
		}
		if low[v] == index[v] {
			id := len(comps)
			var members []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp[w] = id
				members = append(members, w)
				if w == v {
					break
				}
			}
			comps = append(comps, members)
		}
	}
	for _, k := range keys {
		if index[k] == 0 {
			strong(k)
		}
	}
	return comp, comps
}
