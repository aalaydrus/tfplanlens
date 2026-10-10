package analyze

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// ParseARNAccount returns the account ID of an ARN
// (arn:partition:service:region:account:resource), or "" if absent.
func ParseARNAccount(arn string) string {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 || parts[0] != "arn" {
		return ""
	}
	acct := parts[4]
	if len(acct) != 12 || strings.IndexFunc(acct, func(r rune) bool { return !unicode.IsDigit(r) }) >= 0 {
		return ""
	}
	return acct
}

// ValidAccountID reports whether s is a 12-digit AWS account ID.
func ValidAccountID(s string) bool { return validAccountID(s) }

func validAccountID(s string) bool {
	return len(s) == 12 && strings.IndexFunc(s, func(r rune) bool { return !unicode.IsDigit(r) }) < 0
}

// isARNKey reports whether an attribute name is ARN-typed: "arn", or ending
// in "_arn" / "_arns".
func isARNKey(key string) bool {
	return key == "arn" || key == "arns" || strings.HasSuffix(key, "_arn") || strings.HasSuffix(key, "_arns")
}

// arnWarnings flags ARN-typed attributes whose account ID disagrees with the
// provider's resolved target account.
func arnWarnings(c plan.Change, acct account) []Warning {
	var val, sens, unk any = c.After, c.AfterSensitive, c.AfterUnknown
	if c.Action == plan.ActionDelete || c.After == nil {
		val, sens, unk = c.Before, c.BeforeSensitive, nil
	}
	leaves := map[string]leaf{}
	flatten(val, sens, unk, "", leaves)
	paths := make([]string, 0, len(leaves))
	for p := range leaves {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var out []Warning
	for _, p := range paths {
		l := leaves[p]
		s, ok := l.val.(string)
		if !ok || l.sens || l.unknown || !isARNKey(lastKey(p)) {
			continue
		}
		id := ParseARNAccount(s)
		if id == "" || id == acct.label {
			continue
		}
		out = append(out, Warning{
			Kind: "arn_account_mismatch",
			Path: p,
			Message: fmt.Sprintf("%s is in account %s but provider %s targets %s (%s)",
				p, id, c.ProviderKey, acct.label, acct.source),
		})
	}
	return out
}
