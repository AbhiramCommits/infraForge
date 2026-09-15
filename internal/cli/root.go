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

// Execute runs the root command and returns a process exit code:
// 0 ok, 1 provision failure, 2 config error, 3 limit violation.
func Execute() int {
	opts := &Options{}
	cmd := newRootCommand(opts)
	if err := cmd.Execute(); err != nil {
		code := exitCode(err)
		if opts.Logger != nil {
			opts.Logger.Error("command failed", "error", err, "exit_code", code)
		} else {
			fmt.Fprintf(os.Stderr, "infraforge: %v\n", err)
		}
		return code
	}
	return ExitOK
}

// newRootCommand builds the root command with global flags and subcommands.
func newRootCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "infraforge",
		Short: "Provision infrastructure with Ansible and cgroup v2 limits",
		Long: `infraforge provisions multi-node infrastructure with idempotent
Ansible playbooks and enforces per-workload CPU and memory limits with
Linux cgroups v2.`,
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
