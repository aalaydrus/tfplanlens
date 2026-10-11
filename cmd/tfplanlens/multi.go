package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/config"
	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// loaded is one analyzed plan; several make a merged report.
type loaded struct {
	label string
	plan  *plan.Plan
	a     *analyze.Analysis
}

// loadAll reads and analyzes every plan argument. One plan is reported as it
// is; several are merged, each under its own label. checkFailOn evaluates
// the --fail-on rules against each plan on its own (including ignored
// types), naming the plan in each match when there are several.
func loadAll(cmd *cobra.Command, args []string, cfg *config.Config, o *reportOptions) (a *analyze.Analysis, checkFailOn func([]analyze.Rule) []analyze.Match, err error) {
	if len(args) > 1 {
		for _, s := range args {
			if s == "-" {
				return nil, nil, errors.New("stdin (-) can only be used with a single plan")
			}
		}
	}
	paths, labels := make([]string, len(args)), make([]string, len(args))
	explicit := make([]bool, len(args))
	for i, s := range args {
		paths[i], labels[i], explicit[i] = splitLabel(s)
	}
	if len(args) > 1 {
		if labels, err = deriveLabels(paths, labels, explicit); err != nil {
			return nil, nil, err
		}
	}

	var all []loaded
	for i, path := range paths {
		p, an, err := loadAnalysis(cmd, path, cfg, o)
		if err != nil {
			if len(paths) > 1 {
				err = fmt.Errorf("%s: %w", path, err)
			}
			return nil, nil, err
		}
		all = append(all, loaded{labels[i], p, an})
	}
	if len(all) == 1 {
		one := all[0]
		return one.a, func(rules []analyze.Rule) []analyze.Match { return analyze.CheckFailOn(one.plan, one.a, rules) }, nil
	}

	analyses := make([]*analyze.Analysis, len(all))
	for i, l := range all {
		analyses[i] = l.a
	}
	return analyze.Merge(labels, analyses), func(rules []analyze.Rule) []analyze.Match {
		var out []analyze.Match
		for _, l := range all {
			for _, m := range analyze.CheckFailOn(l.plan, l.a, rules) {
				m.Address = analyze.Prefix(l.label, m.Address)
				out = append(out, m)
			}
		}
		return out
	}, nil
}

// splitLabel splits an optional "label=path" argument.
func splitLabel(s string) (path, label string, explicit bool) {
	if i := strings.Index(s, "="); i > 0 && !strings.ContainsAny(s[:i], `/\`) {
		return s[i+1:], s[:i], true
	}
	return s, "", false
}

// genericNames are file names that say nothing about which unit a plan is
// (Terragrunt and CI usually write plan.json into a directory per unit).
var genericNames = map[string]bool{"plan": true, "tfplan": true, "plan.out": true}

// deriveLabels names each plan after its file, or after as many trailing
// path elements as it takes to tell the plans apart; "live/prod/vpc/plan.json"
// is "vpc". Explicit labels (label=path) are kept.
func deriveLabels(paths, labels []string, explicit []bool) ([]string, error) {
	parts := make([][]string, len(paths))
	for i, p := range paths {
		p = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(p)), filepath.Ext(p))
		var ps []string
		for _, s := range strings.Split(p, "/") {
			if s != "" && s != "." && s != ".." {
				ps = append(ps, s)
			}
		}
		if len(ps) > 1 && genericNames[ps[len(ps)-1]] {
			ps = ps[:len(ps)-1]
		}
		parts[i] = ps
	}
	for k := 1; ; k++ {
		out := make([]string, len(paths))
		seen := map[string]bool{}
		unique, more := true, false
		for i := range paths {
			if explicit[i] {
				out[i] = labels[i]
			} else {
				ps := parts[i]
				n := min(k, len(ps))
				out[i] = strings.Join(ps[len(ps)-n:], "/")
				more = more || len(ps) > k
			}
			if out[i] == "" || seen[out[i]] {
				unique = false
			}
			seen[out[i]] = true
		}
		if unique {
			return out, nil
		}
		if !more {
			return nil, fmt.Errorf("cannot tell the plans apart by name (%s); give each a label: label=path", strings.Join(paths, ", "))
		}
	}
}
