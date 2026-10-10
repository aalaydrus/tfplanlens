// Command tfplanlens reads a Terraform plan the way a reviewer needs to.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
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
			// TODO(v0.1): load plan, analyze, render, evaluate --fail-on.
			return fmt.Errorf("report: %w", errNotImplemented)
		},
	}
	f := cmd.Flags()
	f.StringSliceVar(&o.formats, "format", []string{"text"}, "output format: text|html|markdown|json (repeatable)")
	f.StringVar(&o.out, "out", "", "output path (html, markdown, json)")
	f.StringVar(&o.config, "config", "", "config file (default .tfplanlens.yaml if present)")
	f.StringVar(&o.failOn, "fail-on", "", "comma-separated rules, e.g. delete,replace,drift,replace:aws_db_instance")
	f.IntVar(&o.maxLine, "max-line", 120, "truncation width for terminal output")
	f.BoolVar(&o.noColor, "no-color", false, "disable colored output")
	return cmd
}
