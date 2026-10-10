package analyze

import (
	"strings"
	"unicode"
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
