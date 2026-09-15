package cli

import "github.com/spf13/cobra"

// newReportCommand builds the "report" subcommand.
func newReportCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Report host state and applied limits",
		Long:  "Prints a report of every host in the config and the cgroup v2 limits applied to it.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Logger.Info("report not yet implemented",
				"dry_run", opts.DryRun,
				"inventory", opts.Inventory,
				"config", opts.Config,
			)
			return nil
		},
	}
}
