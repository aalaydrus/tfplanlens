// Command tfplanlens reads a Terraform plan the way a reviewer needs to.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/config"
	"github.com/aalaydrus/tfplanlens/internal/plan"
	"github.com/aalaydrus/tfplanlens/internal/render/jsonout"
	"github.com/aalaydrus/tfplanlens/internal/render/text"
)

// version is overridden at release time via -ldflags.
var version = "dev"

// Exit codes (SPEC section 4).
const (
	exitOK     = 0
	exitFailOn = 1
	exitError  = 2
)

// errNotImplemented is returned by commands whose analysis is not built yet.
var errNotImplemented = errors.New("not implemented")

type reportOptions struct {
	formats []string
	out     string
	config  string
	failOn  string
	maxLine int
	noColor bool

	applyAccount        string
	accounts            []string
	noPrompt            bool
	requireApplyAccount bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the CLI and returns the process exit code. It exists so tests
// can drive the CLI without spawning a process.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root := newRootCmd(stdin, stdout, stderr)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		var fo *failOnError
		if errors.As(err, &fo) {
			fmt.Fprintln(stderr, "fail-on:", fo.Error())
			return exitFailOn
		}
		fmt.Fprintln(stderr, "error:", err)
		return exitError
	}
	return exitOK
}

func newRootCmd(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "tfplanlens",
		Short:         "Read a Terraform plan the way a reviewer needs to",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(newReportCmd(), newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the tfplanlens version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}

func newReportCmd() *cobra.Command {
	var o reportOptions
	cmd := &cobra.Command{
		Use:   "report <plan.json|->",
		Short: "Analyze a `terraform show -json` plan and print a report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReport(cmd, args[0], &o)
		},
	}
	f := cmd.Flags()
	f.StringSliceVar(&o.formats, "format", []string{"text"}, "output format: text|html|markdown|json (repeatable)")
	f.StringVar(&o.out, "out", "", "output path (html, markdown, json)")
	f.StringVar(&o.config, "config", "", "config file (default .tfplanlens.yaml if present)")
	f.StringVar(&o.failOn, "fail-on", "", "comma-separated rules, e.g. delete,replace,drift,replace:aws_db_instance")
	f.IntVar(&o.maxLine, "max-line", 120, "truncation width for terminal output")
	f.BoolVar(&o.noColor, "no-color", false, "disable colored output")
	f.StringVar(&o.applyAccount, "tfapply-account-id", "", "12-digit ID of the account terraform apply runs from (env TFPLANLENS_TFAPPLY_ACCOUNT_ID, config tfapply_account_id)")
	f.StringArrayVar(&o.accounts, "account", nil, "target account for a provider alias, alias=id (repeatable; overrides every other source)")
	f.BoolVar(&o.noPrompt, "no-prompt", false, "never ask interactively for the apply account")
	f.BoolVar(&o.requireApplyAccount, "require-apply-account", false, "exit 2 if the apply account is unresolved (for CI)")
	return cmd
}

// failOnError is returned when a --fail-on rule matched; the report has
// already been written.
type failOnError struct{ matches []analyze.Match }

func (e *failOnError) Error() string {
	seen := map[string]bool{}
	var parts []string
	for _, m := range e.matches {
		k := m.Rule.String() + " " + m.Address
		if !seen[k] {
			seen[k] = true
			parts = append(parts, fmt.Sprintf("%s matched %s", m.Rule, m.Address))
		}
	}
	return strings.Join(parts, "; ")
}

func runReport(cmd *cobra.Command, path string, o *reportOptions) error {
	rules, err := analyze.ParseRules(o.failOn)
	if err != nil {
		return err
	}
	formats := o.formats
	for _, f := range formats {
		switch f {
		case "text", "json":
		case "html", "markdown":
			return fmt.Errorf("--format %s: %w", f, errNotImplemented)
		default:
			return fmt.Errorf("unsupported --format %q (want text, html, markdown or json)", f)
		}
	}
	if len(formats) > 1 && o.out != "" {
		return errors.New("--out cannot be combined with several --format values yet")
	}

	cfg, err := config.Load(o.config)
	if err != nil {
		return err
	}
	overrides, err := parseAccountOverrides(o.accounts)
	if err != nil {
		return err
	}
	applyID, applySource, err := resolveApplyAccount(o, cfg, terminal{
		in:        cmd.InOrStdin(),
		prompt:    cmd.ErrOrStderr(),
		stdinTTY:  isTerminal(cmd.InOrStdin()) && path != "-",
		stdoutTTY: isTerminal(cmd.OutOrStdout()),
	}, cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	var in io.Reader = cmd.InOrStdin()
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}
	p, err := plan.Load(in)
	if err != nil {
		return err
	}
	a := analyze.Analyze(p, cfg, &analyze.Options{ApplyAccount: applyID, ApplySource: applySource, AccountOverrides: overrides})

	for i, f := range formats {
		var w io.Writer = cmd.OutOrStdout()
		var file *os.File
		if o.out != "" {
			if file, err = os.Create(o.out); err != nil {
				return err
			}
			w = file
		} else if i > 0 {
			fmt.Fprintln(w)
		}
		switch f {
		case "text":
			err = text.Render(w, a, text.Options{Color: useColor(cmd.OutOrStdout(), o.noColor), MaxLine: o.maxLine})
		case "json":
			err = jsonout.Render(w, a)
		}
		if file != nil {
			if cerr := file.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			return err
		}
	}

	// --fail-on is evaluated on the whole plan, including ignored types.
	if m := analyze.CheckFailOn(p, rules); len(m) > 0 {
		return &failOnError{matches: m}
	}
	return nil
}

// useColor enables ANSI color only for terminals, honoring --no-color and
// the NO_COLOR convention.
func useColor(w io.Writer, noColor bool) bool {
	if noColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
