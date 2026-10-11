// Package redact hides values that look like secrets but that Terraform did
// not mark as sensitive: attributes named like a password or token, key
// material, and "NAME=value" assignments inside long values such as decoded
// user_data. It is a best-effort filter, not a guarantee: a secret with an
// innocent name and no recognizable shape is not caught.
package redact

import (
	"regexp"
	"strings"
)

// Marker replaces a hidden value.
const Marker = "[REDACTED]"

// Redactor hides secret-looking text and counts what it hid. A nil *Redactor
// hides nothing, so callers can switch redaction off by passing nil.
type Redactor struct{ n int }

// New returns a Redactor.
func New() *Redactor { return &Redactor{} }

// Count is the number of values hidden so far.
func (r *Redactor) Count() int {
	if r == nil {
		return 0
	}
	return r.n
}

// Hit records one hidden value (used when the caller hides a whole attribute).
func (r *Redactor) Hit() {
	if r != nil {
		r.n++
	}
}

// secretWord matches the words that make a name or a key look like a secret.
const secretWord = `(?:pass(?:word|wd|phrase)|secret|token|api[_-]?key|private[_-]?key|access[_-]?key|credential|authorization|bearer)`

var (
	nameRe = regexp.MustCompile(`(?i)` + secretWord)
	// Names that contain a secret word but describe the secret, not hold it
	// (an ARN, an id, a length, a count, a version...).
	notSecretSuffix = regexp.MustCompile(`(?i)[_-](?:arns?|ids?|names?|lengths?|versions?|enabled|types?|counts?|days|seconds)$`)

	// key=value, key: value and "key":"value" assignments. A bare colon is not
	// treated as an assignment (secretsmanager:GetSecretValue is an IAM action).
	keyChars = `[A-Za-z0-9_.-]*`
	assignEq = regexp.MustCompile(`(?i)(` + keyChars + secretWord + keyChars + `)(["']?\s*=\s*)("[^"\n]*"|'[^'\n]*'|[^\s"';,]+)`)
	assignSp = regexp.MustCompile(`(?i)(` + keyChars + secretWord + keyChars + `)(["']?\s*:\s+)("[^"\n]*"|'[^'\n]*'|[^\s"';,]+)`)
	assignQ  = regexp.MustCompile(`(?i)(` + keyChars + secretWord + keyChars + `)(["']?:)("[^"\n]*"|'[^'\n]*')`)

	privateKey = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?(?:-----END [A-Z0-9 ]*PRIVATE KEY-----|$)`)
	tokens     = regexp.MustCompile(strings.Join([]string{
		`\b(?:AKIA|ASIA|AGPA|AIDA|AROA|ANPA|ANVA|AIPA)[0-9A-Z]{16}\b`, // AWS access key id
		`\bgh[pousr]_[A-Za-z0-9]{30,}\b`,                              // GitHub
		`\bgithub_pat_[A-Za-z0-9_]{20,}\b`,
		`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`,                               // Slack
		`\bsk[-_](?:live|test)?[-_]?[A-Za-z0-9]{24,}\b`,                  // Stripe / API-style
		`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]+`, // JWT
	}, "|"))
	urlAuth = regexp.MustCompile(`(://[^/\s:@]+:)([^/\s@]+)(@)`)
)

// SecretName reports whether an attribute name looks like it holds a secret.
func SecretName(key string) bool {
	k := strings.ToLower(key)
	return nameRe.MatchString(k) && !notSecretSuffix.MatchString(k)
}

// isReference is true for values that point at a secret instead of being one.
func isReference(v string) bool {
	v = strings.Trim(v, `"'`)
	return v == "" || v == Marker || strings.HasPrefix(v, "$") || strings.HasPrefix(v, "{{") ||
		strings.HasPrefix(v, "<") || strings.HasPrefix(v, "(known after apply")
}

// Text hides secret-looking content in s: key material, known token shapes,
// credentials in URLs and the values of secret-named assignments.
func (r *Redactor) Text(s string) string {
	if r == nil || s == "" {
		return s
	}
	hide := func(re *regexp.Regexp, repl func(string) string) {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			r.n++
			return repl(m)
		})
	}
	hide(privateKey, func(string) string { return Marker + " (private key)" })
	hide(tokens, func(string) string { return Marker })
	s = urlAuth.ReplaceAllStringFunc(s, func(m string) string {
		sub := urlAuth.FindStringSubmatch(m)
		if isReference(sub[2]) {
			return m
		}
		r.n++
		return sub[1] + Marker + sub[3]
	})
	for _, re := range []*regexp.Regexp{assignEq, assignSp, assignQ} {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			sub := re.FindStringSubmatch(m)
			if notSecretSuffix.MatchString(strings.ToLower(sub[1])) || isReference(sub[3]) {
				return m
			}
			r.n++
			if strings.HasPrefix(sub[3], `"`) {
				return sub[1] + sub[2] + `"` + Marker + `"`
			}
			if strings.HasPrefix(sub[3], `'`) {
				return sub[1] + sub[2] + `'` + Marker + `'`
			}
			return sub[1] + sub[2] + Marker
		})
	}
	return s
}

// Value hides secret-looking content in a plan value. Only strings can carry
// text, so every other kind of value is returned unchanged.
func (r *Redactor) Value(v any) any {
	if str, ok := v.(string); ok {
		return r.Text(str)
	}
	return v
}
