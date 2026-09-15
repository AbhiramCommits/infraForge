// Package cli wires up the infraforge command line interface.
package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

const (
	defaultInventory = "deploy/inventory.yml"
	defaultConfig    = "infraforge.yaml"
)

// Options holds global flag values shared by all commands.
type Options struct {
	// Inventory is the path to the Ansible inventory.
	Inventory string
	// Config is the path to the infraforge configuration file.
	Config string
	// DryRun reports whether the invocation should only print planned actions.
	DryRun bool
	// LogFormat selects the log output format ("text" or "json").
	LogFormat string

	// RunID identifies this invocation in every log line.
	RunID string
	// Logger is the structured logger carrying the run ID.
	Logger *slog.Logger
	// Stderr receives log output; defaults to os.Stderr.
	Stderr io.Writer
	// ReportsDir is where run reports are written; defaults to "reports".
	ReportsDir string
}

// Execute runs the root command and returns a process exit code.
func Execute() int {
	opts := &Options{}
	cmd := newRootCommand(opts)
	if err := cmd.Execute(); err != nil {
		if opts.Logger != nil {
			opts.Logger.Error("command failed", "error", err)
		} else {
			fmt.Fprintf(os.Stderr, "infraforge: %v\n", err)
		}
		return 1
	}
	return 0
}

// newRootCommand builds the root command with global flags and subcommands.
func newRootCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "infraforge",
		Short:         "Provision infrastructure with Ansible and cgroup v2 limits",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return initializeRun(opts)
		},
	}

	pf := cmd.PersistentFlags()
	pf.StringVar(&opts.Inventory, "inventory", defaultInventory, "path to the Ansible inventory")
	pf.StringVar(&opts.Config, "config", defaultConfig, "path to the infraforge configuration file")
	pf.StringVar(&opts.ReportsDir, "reports-dir", "reports", "directory for run reports")
	pf.BoolVar(&opts.DryRun, "dry-run", false, "print planned actions without executing them")
	pf.StringVar(&opts.LogFormat, "log-format", "text", "log output format: text or json")

	cmd.AddCommand(
		newProvisionCommand(opts),
		newLimitCommand(opts),
		newVMCommand(opts),
		newReportCommand(opts),
	)

	return cmd
}
