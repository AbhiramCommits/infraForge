package cli

import "github.com/spf13/cobra"

// newLimitCommand builds the "limit" subcommand.
func newLimitCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "limit",
		Short: "Apply cgroup v2 resource limits to hosts",
		Long:  "Reads per-host limits from the config file and applies them through Linux cgroup v2.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Logger.Info("limit not yet implemented",
				"dry_run", opts.DryRun,
				"inventory", opts.Inventory,
				"config", opts.Config,
			)
			return nil
		},
	}
}
