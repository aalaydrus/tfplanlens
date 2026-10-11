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
	"github.com/aalaydrus/tfplanlens/internal/diff"
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
	root.AddCommand(newReportCmd(), newDiffCmd(), newTUICmd(), newVersionCmd())
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
		Use:   "report <plan.json|-> [more plans...]",
		Short: "Analyze a `terraform show -json` plan and print a report",
		Long: `Analyze a ` + "`terraform show -json`" + ` plan and print a report. Pass - to read stdin.

Pass several plans (the root modules of a monorepo, say) to get one merged
report: every address is prefixed with the plan's label, which is
its file name, its directory when the file is called plan.json, or label=path.
Each plan is analyzed on its own, so accounts and waves stay per plan.

Settings come from ./` + config.DefaultPath + ` when it exists; flags override it.
Accounts default to what the plan's own provider configuration says; set
tfapply_account_id and accounts: in the config file to fill in the rest.`,
		Example: `  tfplanlens report plan.json
  tfplanlens report plan.json -f html -o report.html
  terraform show -json plan.out | tfplanlens report - -f markdown
  tfplanlens report network/plan.json app/plan.json -f html -o all.html
  tfplanlens report net=a.json app=b.json
  tfplanlens report plan.json --fail-on delete,replace:aws_db_instance`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReport(cmd, args, &o)
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
type failOnError struct {
	matches   []analyze.Match
	newDanger []string // diff --fail-on-new-danger
}

func (e *failOnError) Error() string {
	if len(e.newDanger) > 0 {
		return "new danger: " + strings.Join(e.newDanger, ", ") + " deleted or replaced by the new plan only"
	}
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

func runReport(cmd *cobra.Command, paths []string, o *reportOptions) error {
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

	a, checkFailOn, err := loadAll(cmd, paths, cfg, o)
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
	if m := checkFailOn(rules); len(m) > 0 {
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
		Use:   "tui <plan.json> [more plans...]",
		Short: "Explore a plan interactively in the terminal",
		Long: `Explore a plan interactively: expand and collapse changes, search, filter by
action, group by account or module, and see the apply/destroy waves.
Account settings come from the same config file and flags as report.`,
		Example: "  tfplanlens tui plan.json\n  tfplanlens tui network/plan.json app/plan.json",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, a := range args {
				if a == "-" {
					return errors.New("tui needs plan files, not stdin (stdin is the keyboard)")
				}
			}
			cfg, err := config.Load(o.config)
			if err != nil {
				return err
			}
			a, _, err := loadAll(cmd, args, cfg, &o)
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

func newDiffCmd() *cobra.Command {
	var o reportOptions
	var failNew bool
	cmd := &cobra.Command{
		Use:   "diff <old-plan.json> <new-plan.json>",
		Short: "Show what changed between two plans of the same configuration",
		Long: `Compare two plans: what is new, what no longer changes, and what changes
differently (action, reason, target account, attributes). Use it to review only
what a new push changed. Resources are matched by address; a plain no-op counts
as not changing. Account settings come from the same config file and flags as
report, applied to both plans.`,
		Example: `  tfplanlens diff before.json after.json
  tfplanlens diff before.json after.json -f markdown -o diff.md
  tfplanlens diff before.json after.json --fail-on-new-danger`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "-" && args[1] == "-" {
				return errors.New("only one plan can be read from stdin")
			}
			if len(o.formats) > 1 {
				return errors.New("diff takes one --format")
			}
			format := "text"
			if len(o.formats) == 1 {
				format = o.formats[0]
			}
			switch format {
			case "text", "markdown", "json":
			default:
				return fmt.Errorf("unsupported --format %q for diff (want text, markdown or json)", format)
			}
			cfg, err := config.Load(o.config)
			if err != nil {
				return err
			}
			_, oldA, err := loadAnalysis(cmd, args[0], cfg, &o)
			if err != nil {
				return fmt.Errorf("%s: %w", args[0], err)
			}
			_, newA, err := loadAnalysis(cmd, args[1], cfg, &o)
			if err != nil {
				return fmt.Errorf("%s: %w", args[1], err)
			}
			res := diff.Compare(oldA, newA)

			var w io.Writer = cmd.OutOrStdout()
			var file *os.File
			if o.out != "" {
				if file, err = os.Create(o.out); err != nil {
					return err
				}
				w = file
			}
			switch format {
			case "text":
				err = diff.RenderText(w, res)
			case "markdown":
				err = diff.RenderMarkdown(w, res)
			case "json":
				err = diff.RenderJSON(w, res)
			}
			if file != nil {
				if cerr := file.Close(); err == nil {
					err = cerr
				}
			}
			if err != nil {
				return err
			}
			if failNew && res.Dangerous() {
				return &failOnError{matches: nil, newDanger: res.NewDanger}
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringSliceVarP(&o.formats, "format", "f", nil, "text, markdown or json (default text)")
	f.StringVarP(&o.out, "out", "o", "", "write to this file instead of stdout")
	f.StringVarP(&o.config, "config", "c", "", "config file (default ./"+config.DefaultPath+" if present)")
	f.BoolVar(&failNew, "fail-on-new-danger", false, "exit 1 when the new plan deletes or replaces something the old one did not")
	f.StringVar(&o.applyAccount, "tfapply-account-id", "", "account terraform apply runs from (env TFPLANLENS_TFAPPLY_ACCOUNT_ID)")
	f.StringArrayVar(&o.accounts, "account", nil, "target account for a provider alias, alias=id; repeatable")
	return cmd
}
