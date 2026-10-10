package analyze

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/aalaydrus/tfplanlens/internal/valuediff"
)

type leaf struct {
	val     any
	sens    bool
	unknown bool
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func sub(node any, key string) any {
	if m, ok := node.(map[string]any); ok {
		return m[key]
	}
	return nil
}

func subIdx(node any, i int) any {
	if l, ok := node.([]any); ok && i < len(l) {
		return l[i]
	}
	return nil
}

// hasMarker reports whether a sensitive/unknown tree marks anything.
func hasMarker(n any) bool {
	switch v := n.(type) {
	case bool:
		return v
	case map[string]any:
		for _, c := range v {
			if hasMarker(c) {
				return true
			}
		}
	case []any:
		for _, c := range v {
			if hasMarker(c) {
				return true
			}
		}
	}
	return false
}

// flatten walks a value together with its parallel sensitive and unknown
// trees into leaf paths. A marked subtree collapses into a single leaf.
func flatten(val, sens, unk any, path string, out map[string]leaf) {
	if b, ok := sens.(bool); ok && b {
		out[path] = leaf{val: val, sens: true}
		return
	}
	if b, ok := unk.(bool); ok && b {
		out[path] = leaf{unknown: true}
		return
	}
	switch v := val.(type) {
	case map[string]any:
		keys := map[string]bool{}
		for k := range v {
			keys[k] = true
		}
		for _, t := range []any{sens, unk} {
			if m, ok := t.(map[string]any); ok {
				for k, c := range m {
					if hasMarker(c) {
						keys[k] = true
					}
				}
			}
		}
		if len(keys) == 0 {
			out[path] = leaf{val: val}
			return
		}
		for k := range keys {
			flatten(v[k], sub(sens, k), sub(unk, k), join(path, k), out)
		}
	case []any:
		n := len(v)
		for _, t := range []any{sens, unk} {
			if l, ok := t.([]any); ok && len(l) > n {
				n = len(l)
			}
		}
		if n == 0 {
			out[path] = leaf{val: val}
			return
		}
		for i := 0; i < n; i++ {
			var e any
			if i < len(v) {
				e = v[i]
			}
			flatten(e, subIdx(sens, i), subIdx(unk, i), path+"["+strconv.Itoa(i)+"]", out)
		}
	case nil:
		// May be absent, or unknown/sensitive inside a container.
		switch {
		case hasMarker(sens) || hasMarker(unk):
			switch s := sens.(type) {
			case map[string]any:
				flatten(map[string]any{}, s, unk, path, out)
				return
			case []any:
				flatten([]any{}, s, unk, path, out)
				return
			}
			switch u := unk.(type) {
			case map[string]any:
				flatten(map[string]any{}, sens, u, path, out)
				return
			case []any:
				flatten([]any{}, sens, u, path, out)
				return
			}
			out[path] = leaf{val: nil}
		default:
			out[path] = leaf{val: nil}
		}
	default:
		out[path] = leaf{val: val}
	}
}

func countUnknown(n any) int {
	switch v := n.(type) {
	case bool:
		if v {
			return 1
		}
	case map[string]any:
		t := 0
		for _, c := range v {
			t += countUnknown(c)
		}
		return t
	case []any:
		t := 0
		for _, c := range v {
			t += countUnknown(c)
		}
		return t
	}
	return 0
}

func format(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "?"
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func lastKey(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		path = path[i+1:]
	}
	if i := strings.IndexByte(path, '['); i >= 0 {
		path = path[:i]
	}
	return path
}

// diffAttrs lists changed attributes between before and after. Sensitive
// values are never copied into the result.
func diffAttrs(before, after, bsens, asens, aunk any, replacePaths []string, longKeys []string) []Attr {
	bl, al := map[string]leaf{}, map[string]leaf{}
	flatten(before, bsens, nil, "", bl)
	flatten(after, asens, aunk, "", al)
	// A nil root yields a "" leaf for an absent side; drop it.
	if before == nil {
		delete(bl, "")
	}
	if after == nil && !hasMarker(aunk) {
		delete(al, "")
	}
	dropEmptyShadowed(bl, al)
	dropEmptyShadowed(al, bl)
	// A marked (unknown or sensitive) leaf stands for everything beneath it,
	// so the other side's expanded children are not separate changes.
	collapseUnderMarked(bl, al)
	collapseUnderMarked(al, bl)
	maskDerivedSecrets(bl, al)
	paths := make([]string, 0, len(bl)+len(al))
	seen := map[string]bool{}
	for p := range bl {
		seen[p] = true
		paths = append(paths, p)
	}
	for p := range al {
		if !seen[p] {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	forces := map[string]bool{}
	for _, p := range replacePaths {
		forces[p] = true
	}

	attrs := []Attr{}
	for _, p := range paths {
		b, bok := bl[p]
		a, aok := al[p]
		if bok && aok && b.sens == a.sens && b.unknown == a.unknown && reflect.DeepEqual(b.val, a.val) {
			continue
		}
		at := Attr{Path: p, ForcesReplace: forces[p]}
		switch {
		case a.unknown:
			// Known-after-apply attributes are counted (Change.KnownAfterApply),
			// never listed (SPEC 5.1): their old values are often huge or
			// derived from secrets, and they are noise in a review.
			continue
		case b.sens || a.sens:
			at.Kind = "sensitive"
		default:
			bs, bIsStr := b.val.(string)
			as, aIsStr := a.val.(string)
			key := lastKey(p)
			if (bIsStr && valuediff.IsLong(key, bs, longKeys)) || (aIsStr && valuediff.IsLong(key, as, longKeys)) {
				at.Kind = "long"
				d := valuediff.Diff(p, bs, as)
				at.Diff = &d
				break
			}
			at.Kind = "value"
			if bok {
				at.Before = json.RawMessage(format(b.val))
			}
			if aok {
				at.After = json.RawMessage(format(a.val))
			}
		}
		attrs = append(attrs, at)
	}
	return attrs
}

// dropEmptyShadowed removes empty-container leaves ({} or []) from m when the
// other side has children under the same path, so "tags: {} -> (absent)" does
// not appear next to "tags.owner: (absent) -> ...".
func dropEmptyShadowed(m, other map[string]leaf) {
	for p, l := range m {
		if !isEmptyContainer(l.val) {
			continue
		}
		for q := range other {
			if strings.HasPrefix(q, p+".") || strings.HasPrefix(q, p+"[") {
				delete(m, p)
				break
			}
		}
	}
}

func isEmptyContainer(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		return len(x) == 0
	case []any:
		return len(x) == 0
	}
	return false
}

// collapseUnderMarked removes leaves of m that sit beneath an unknown or
// sensitive leaf of other: "output" becoming (known after apply) must not
// also report output[0], output[1], ... as removed.
func collapseUnderMarked(m, other map[string]leaf) {
	for p, l := range other {
		if !l.unknown && !l.sens {
			continue
		}
		for q := range m {
			if strings.HasPrefix(q, p+".") || strings.HasPrefix(q, p+"[") {
				delete(m, q)
			}
		}
	}
}

// minDerivedSecretLen is the shortest sensitive value that is also masked
// where it reappears; shorter values ("1", "on") would mask innocent fields.
const minDerivedSecretLen = 4

// maskDerivedSecrets marks a leaf sensitive when its string value equals the
// value of a sensitive leaf in the same change. Providers often mirror a
// sensitive input into a computed attribute (terraform_data.output) without
// marking the copy, and Terraform's own plan JSON then carries the secret in
// the clear.
func maskDerivedSecrets(bl, al map[string]leaf) {
	secrets := map[string]bool{}
	for _, m := range []map[string]leaf{bl, al} {
		for _, l := range m {
			if s, ok := l.val.(string); ok && l.sens && len(s) >= minDerivedSecretLen {
				secrets[s] = true
			}
		}
	}
	if len(secrets) == 0 {
		return
	}
	for _, m := range []map[string]leaf{bl, al} {
		for p, l := range m {
			if s, ok := l.val.(string); ok && !l.sens && secrets[s] {
				l.sens = true
				m[p] = l
			}
		}
	}
}
