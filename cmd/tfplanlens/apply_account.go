package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/config"
)

// envApplyAccount is the environment variable for the apply account.
const envApplyAccount = "TFPLANLENS_TFAPPLY_ACCOUNT_ID"

// isTerminal reports whether x is a real terminal. A character-device check
// is not enough: /dev/null is a character device too, so
// `tfplanlens ... >/dev/null` would be treated as interactive.
func isTerminal(x any) bool {
	f, ok := x.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func checkAccountID(what, id string) error {
	if !analyze.ValidAccountID(id) {
		return fmt.Errorf("%s: %q is not a 12-digit account ID", what, id)
	}
	return nil
}

// explicitApplyAccount returns the apply account given by the flag, the
// environment or the config file, highest precedence first. An empty id means
// none was given; the analysis then infers it from the plan's default provider.
func explicitApplyAccount(o *reportOptions, cfg *config.Config) (id, source string, err error) {
	switch {
	case o.applyAccount != "":
		id, source = o.applyAccount, analyze.SourceFlag
		err = checkAccountID("--tfapply-account-id", id)
	case os.Getenv(envApplyAccount) != "":
		id, source = os.Getenv(envApplyAccount), analyze.SourceEnv
		err = checkAccountID(envApplyAccount, id)
	case cfg.TFApplyAccountID != "":
		id, source = cfg.TFApplyAccountID, analyze.SourceConfig
		err = checkAccountID("tfapply_account_id", id)
	}
	if err != nil {
		return "", "", err
	}
	return id, source, nil
}

// parseAccountOverrides parses repeated --account alias=id values.
func parseAccountOverrides(vals []string) (map[string]string, error) {
	out := map[string]string{}
	for _, v := range vals {
		alias, id, ok := strings.Cut(v, "=")
		alias = strings.TrimSpace(alias)
		if !ok || alias == "" {
			return nil, fmt.Errorf("--account %q: want alias=id", v)
		}
		if err := checkAccountID("--account "+alias, strings.TrimSpace(id)); err != nil {
			return nil, err
		}
		out[alias] = strings.TrimSpace(id)
	}
	return out, nil
}
