// Command tfplanlens reads a Terraform plan the way a reviewer needs to.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/config"
	"github.com/aalaydrus/tfplanlens/internal/plan"
	htmlrender "github.com/aalaydrus/tfplanlens/internal/render/html"
	"github.com/aalaydrus/tfplanlens/internal/render/jsonout"
	"github.com/aalaydrus/tfplanlens/internal/render/markdown"
	"github.com/aalaydrus/tfplanlens/internal/render/text"
	"github.com/aalaydrus/tfplanlens/internal/tui"
)

// version is overridden at release time via -ldflags. Local builds keep "dev"
// and report the git revision they were built from (see versionString).
var version = "dev"

// Exit codes (SPEC section 4).
const (
	exitOK     = 0
	exitFailOn = 1
	exitError  = 2
)

type reportOptions struct {
	formats      []string
	out          string
	config       string
	failOn       string
	noColor      bool
	applyAccount string
	accounts     []string
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
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newReportCmd(), newTUICmd(), newVersionCmd())
	return root
}

// versionString is the injected release version, or for a local build
// ("dev") the short git revision stamped by the Go toolchain, e.g.
// "dev (8814cf7)" or "dev (8814cf7, modified)".
func versionString() string {
	info, _ := debug.ReadBuildInfo()
	return formatVersion(version, info)
}

func formatVersion(v string, info *debug.BuildInfo) string {
	if v != "dev" || info == nil {
		return v
	}
	var rev string
	modified := false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if rev == "" {
		return v
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if modified {
		return fmt.Sprintf("%s (%s, modified)", v, rev)
	}
	return fmt.Sprintf("%s (%s)", v, rev)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the tfplanlens version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), versionString())
		},
	}
}

func newReportCmd() *cobra.Command {
	var o reportOptions
	cmd := &cobra.Command{
		Use:   "report <plan.json|->",
		Short: "Analyze a `terraform show -json` plan and print a report",
		Long: `Analyze a ` + "`terraform show -json`" + ` plan and print a report. Pass - to read stdin.

Settings come from ./` + config.DefaultPath + ` when it exists; flags override it.
Accounts default to what the plan's own provider configuration says; set
tfapply_account_id and accounts: in the config file to fill in the rest.`,
		Example: `  tfplanlens report plan.json
  tfplanlens report plan.json -f html -o report.html
  terraform show -json plan.out | tfplanlens report - -f markdown
  tfplanlens report plan.json --fail-on delete,replace:aws_db_instance`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReport(cmd, args[0], &o)
		},
	}
	f := cmd.Flags()
	f.StringSliceVarP(&o.formats, "format", "f", nil, "text, html, markdown or json; repeatable (default text)")
	f.StringVarP(&o.out, "out", "o", "", "write to this file instead of stdout")
	f.StringVarP(&o.config, "config", "c", "", "config file (default ./"+config.DefaultPath+" if present)")
	f.StringVar(&o.failOn, "fail-on", "", "exit 1 when a rule matches: delete, replace, drift, unresolved, or kind:type, comma-separated")
	f.StringVar(&o.applyAccount, "tfapply-account-id", "", "account terraform apply runs from (env TFPLANLENS_TFAPPLY_ACCOUNT_ID)")
	f.StringArrayVar(&o.accounts, "account", nil, "target account for a provider alias, alias=id; repeatable")
	f.BoolVar(&o.noColor, "no-color", false, "disable colored output")
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
	cfg, err := config.Load(o.config)
	if err != nil {
		return err
	}

	// Flags override the config file, which overrides the defaults.
	formats := o.formats
	if len(formats) == 0 {
		formats = cfg.Format
	}
	if len(formats) == 0 {
		formats = []string{"text"}
	}
	for _, f := range formats {
		switch f {
		case "text", "json", "markdown", "html":
		default:
			return fmt.Errorf("unsupported --format %q (want text, html, markdown or json)", f)
		}
	}
	if len(formats) > 1 && o.out != "" {
		return errors.New("--out cannot be combined with several --format values yet")
	}
	failOn := o.failOn
	if !cmd.Flags().Changed("fail-on") {
		failOn = strings.Join(cfg.FailOn, ",")
	}
	rules, err := analyze.ParseRules(failOn)
	if err != nil {
		return err
	}
	maxLine := cfg.MaxLine
	if maxLine == 0 {
		maxLine = config.DefaultMaxLine
	}

	p, a, err := loadAnalysis(cmd, path, cfg, o)
	if err != nil {
		return err
	}

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
			err = text.Render(w, a, text.Options{Color: useColor(cmd.OutOrStdout(), o.noColor), MaxLine: maxLine})
		case "json":
			err = jsonout.Render(w, a)
		case "markdown":
			err = markdown.Render(w, a)
		case "html":
			err = htmlrender.Render(w, a)
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
	if m := analyze.CheckFailOn(p, a, rules); len(m) > 0 {
		return &failOnError{matches: m}
	}
	return nil
}

// loadAnalysis reads the plan and analyzes it with the account settings from
// the flags and config file.
func loadAnalysis(cmd *cobra.Command, path string, cfg *config.Config, o *reportOptions) (*plan.Plan, *analyze.Analysis, error) {
	overrides, err := parseAccountOverrides(o.accounts)
	if err != nil {
		return nil, nil, err
	}
	applyID, applySource, err := explicitApplyAccount(o, cfg)
	if err != nil {
		return nil, nil, err
	}

	var in io.Reader = cmd.InOrStdin()
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		in = f
	}
	p, err := plan.Load(in)
	if err != nil {
		return nil, nil, err
	}
	a := analyze.Analyze(p, cfg, &analyze.Options{ApplyAccount: applyID, ApplySource: applySource, AccountOverrides: overrides})

	return p, a, nil
}

// useColor enables ANSI color only for terminals, honoring --no-color and
// the NO_COLOR convention.
func useColor(w io.Writer, noColor bool) bool {
	if noColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	return isTerminal(w)
}

func newTUICmd() *cobra.Command {
	var o reportOptions
	cmd := &cobra.Command{
		Use:   "tui <plan.json>",
		Short: "Explore a plan interactively in the terminal",
		Long: `Explore a plan interactively: expand and collapse changes, search, filter by
action, group by account or module, and see the apply/destroy waves.
Account settings come from the same config file and flags as report.`,
		Example: "  tfplanlens tui plan.json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "-" {
				return errors.New("tui needs a plan file, not stdin (stdin is the keyboard)")
			}
			cfg, err := config.Load(o.config)
			if err != nil {
				return err
			}
			_, a, err := loadAnalysis(cmd, args[0], cfg, &o)
			if err != nil {
				return err
			}
			return tui.Run(a)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.config, "config", "c", "", "config file (default ./"+config.DefaultPath+" if present)")
	f.StringVar(&o.applyAccount, "tfapply-account-id", "", "account terraform apply runs from (env TFPLANLENS_TFAPPLY_ACCOUNT_ID)")
	f.StringArrayVar(&o.accounts, "account", nil, "target account for a provider alias, alias=id; repeatable")
	return cmd
}
