// Package valuediff decodes long attribute values (base64, gzip, JSON, IAM
// policies) and produces compact line-level diffs (SPEC 5.4).
package valuediff

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/pmezard/go-difflib/difflib"
)

const (
	// LongThreshold is the string length above which a value is "long".
	LongThreshold = 200
	// Context is the number of unchanged lines shown around a change.
	Context = 2
	// MaxDiffLines caps rendered diff lines per attribute so a value can
	// never be printed in full, even when every line changed.
	MaxDiffLines = 60
	// maxDecoded bounds gzip decompression.
	maxDecoded = 16 << 20
)

// BuiltinKeys are attribute names always treated as long values.
var BuiltinKeys = []string{
	"user_data", "user_data_base64", "policy", "assume_role_policy",
	"inline_policy", "container_definitions", "definition", "body",
}

// IsLong reports whether the attribute with this key and string value gets
// long-value handling. extraKeys comes from the config file.
func IsLong(key, value string, extraKeys []string) bool {
	if len(value) > LongThreshold {
		return true
	}
	for _, k := range BuiltinKeys {
		if key == k {
			return true
		}
	}
	for _, k := range extraKeys {
		if key == k {
			return true
		}
	}
	return false
}

// Result is a rendered diff of one long value.
type Result struct {
	// Header looks like "user_data: 3 lines changed of 412 (+2 -1)".
	Header  string   `json:"header"`
	Lines   []string `json:"lines"`
	Added   int      `json:"added"`
	Removed int      `json:"removed"`
	Total   int      `json:"total_lines"`
	// Truncated is the number of diff lines hidden by MaxDiffLines.
	Truncated int `json:"truncated_lines,omitempty"`
	// FullBefore and FullAfter are the decoded values. Only the HTML report
	// uses them (behind an "expand full value" toggle); they are never
	// serialized and the terminal and markdown renderers never print them.
	FullBefore string `json:"-"`
	FullAfter  string `json:"-"`
}

// Decode normalizes a value for diffing: base64 is decoded, gzip is
// decompressed, JSON is pretty-printed with sorted keys. Steps lists what was
// applied, e.g. ["base64", "gzip", "json"].
func Decode(s string) (out string, steps []string) {
	out = s
	if b, ok := decodeBase64(out); ok {
		out = string(b)
		steps = append(steps, "base64")
	}
	if bs := []byte(out); isGzip(bs) {
		if d, err := gunzip(bs); err == nil {
			out = d
			steps = append(steps, "gzip")
		}
	}
	if p, ok := prettyJSON(out); ok {
		out = p
		steps = append(steps, "json")
	}
	return out, steps
}

func decodeBase64(s string) ([]byte, bool) {
	t := strings.TrimSpace(s)
	if len(t) < 8 || len(t)%4 != 0 {
		return nil, false
	}
	b, err := base64.StdEncoding.DecodeString(t)
	if err != nil {
		return nil, false
	}
	if isGzip(b) {
		return b, true
	}
	// Plain words can be valid base64; require the result to look like text.
	if !utf8.Valid(b) {
		return nil, false
	}
	for _, c := range b {
		if c < 0x20 && c != '\n' && c != '\r' && c != '\t' {
			return nil, false
		}
	}
	return b, true
}

func isGzip(b []byte) bool { return len(b) > 2 && b[0] == 0x1f && b[1] == 0x8b }

func gunzip(b []byte) (string, error) {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(zr, maxDecoded))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func prettyJSON(s string) (string, bool) {
	t := strings.TrimSpace(s)
	if t == "" || (t[0] != '{' && t[0] != '[') {
		return "", false
	}
	var v any
	dec := json.NewDecoder(strings.NewReader(t))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil || dec.More() {
		return "", false
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil { // map keys are sorted by encoding/json
		return "", false
	}
	return strings.TrimSuffix(buf.String(), "\n"), true
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// Options tune Diff.
type Options struct {
	// NoDecode compares the values as they are, without decoding base64, gzip
	// or JSON first.
	NoDecode bool
	// Redact, if set, hides secret-looking text in both sides before they are
	// compared, so neither the diff lines nor FullBefore/FullAfter contain it.
	Redact func(string) string
}

// Diff decodes both sides and returns only changed lines plus context. Pass
// an empty string for a missing side.
func Diff(key, before, after string) Result { return DiffWith(key, before, after, Options{}) }

// DiffWith is Diff with options.
func DiffWith(key, before, after string, o Options) Result {
	b, a := before, after
	if !o.NoDecode {
		b, _ = Decode(before)
		a, _ = Decode(after)
	}
	if o.Redact != nil {
		b, a = o.Redact(b), o.Redact(a)
	}
	res := diffLines(key, splitLines(b), splitLines(a))
	res.FullBefore, res.FullAfter = b, a

	// IAM policies additionally get a per-statement summary.
	if bs, as := PolicyStatements(b), PolicyStatements(a); bs != nil || as != nil {
		res.Lines = append(res.Lines, "statements:")
		res.Lines = append(res.Lines, statementDiff(bs, as)...)
	}
	return res
}

func diffLines(key string, a, b []string) Result {
	m := difflib.NewMatcher(a, b)
	res := Result{Total: max(len(a), len(b)), Lines: []string{}}
	for _, g := range m.GetGroupedOpCodes(Context) {
		first, last := g[0], g[len(g)-1]
		hunk := []string{fmt.Sprintf("@@ -%d,%d +%d,%d @@", first.I1+1, last.I2-first.I1, first.J1+1, last.J2-first.J1)}
		for _, op := range g {
			switch op.Tag {
			case 'e':
				for _, l := range a[op.I1:op.I2] {
					hunk = append(hunk, " "+l)
				}
			case 'd', 'r':
				for _, l := range a[op.I1:op.I2] {
					hunk = append(hunk, "-"+l)
				}
				res.Removed += op.I2 - op.I1
				fallthrough
			case 'i':
				if op.Tag != 'd' {
					for _, l := range b[op.J1:op.J2] {
						hunk = append(hunk, "+"+l)
					}
					res.Added += op.J2 - op.J1
				}
			}
		}
		res.Lines = append(res.Lines, hunk...)
	}
	if len(res.Lines) > MaxDiffLines {
		res.Truncated = len(res.Lines) - MaxDiffLines
		res.Lines = res.Lines[:MaxDiffLines]
	}
	res.Header = fmt.Sprintf("%s: %d lines changed of %d (+%d -%d)", key, res.Added+res.Removed, res.Total, res.Added, res.Removed)
	return res
}
