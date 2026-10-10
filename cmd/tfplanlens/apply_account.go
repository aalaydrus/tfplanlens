package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/config"
)

// envApplyAccount is the environment variable for the apply account.
const envApplyAccount = "TFPLANLENS_TFAPPLY_ACCOUNT_ID"

// errApplyAccountRequired is returned by --require-apply-account when the
// apply account cannot be resolved.
var errApplyAccountRequired = errors.New("apply account is unresolved and --require-apply-account is set")

// terminal describes the streams used for the interactive prompt.
type terminal struct {
	in        io.Reader
	prompt    io.Writer // questions go to stderr, never into the report
	stdinTTY  bool
	stdoutTTY bool
}

// isTerminal reports whether x is a real terminal. A character-device check
// is not enough: /dev/null is a character device too, so `tfplanlens ... </dev/null`
// (cron, CI, scripts) would be treated as interactive and prompt.
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

// resolveApplyAccount applies the precedence: flag, env var, config file,
// interactive prompt, unresolved. An empty id means unresolved.
func resolveApplyAccount(o *reportOptions, cfg *config.Config, term terminal, warn io.Writer) (id, source string, err error) {
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
	if id != "" {
		return id, source, nil
	}

	// Prompt only when a person is clearly at the keyboard.
	if !o.noPrompt && term.stdinTTY && term.stdoutTTY {
		if id = ask(term, o); id != "" {
			return id, analyze.SourcePrompt, nil
		}
	}
	if o.requireApplyAccount {
		return "", "", errApplyAccountRequired
	}
	fmt.Fprintln(warn, "warning: apply account is unresolved (use --tfapply-account-id, "+envApplyAccount+" or tfapply_account_id); "+
		"providers without assume_role stay unresolved and cross-account flags are off")
	return "", analyze.SourceUnresolved, nil
}

// ask prompts for the apply account and offers to save it. It returns "" if
// the user gives no valid answer.
func ask(term terminal, o *reportOptions) string {
	r := bufio.NewReader(term.in)
	var id string
	for try := 0; try < 3; try++ {
		fmt.Fprint(term.prompt, "Which account ID does terraform apply run from? (12 digits, empty to skip) ")
		line, err := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return ""
		}
		if analyze.ValidAccountID(line) {
			id = line
			break
		}
		fmt.Fprintf(term.prompt, "%q is not a 12-digit account ID.\n", line)
		if err != nil {
			return ""
		}
	}
	if id == "" {
		return ""
	}
	path := o.config
	if path == "" {
		path = config.DefaultPath
	}
	fmt.Fprintf(term.prompt, "Save it to %s? [y/N] ", path)
	ans, _ := r.ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(ans)); a == "y" || a == "yes" {
		if err := config.SaveApplyAccount(path, id); err != nil {
			fmt.Fprintf(term.prompt, "could not save: %v\n", err)
		} else {
			fmt.Fprintf(term.prompt, "Saved tfapply_account_id to %s\n", path)
		}
	}
	return id
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
