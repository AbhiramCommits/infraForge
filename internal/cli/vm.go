package cli

import "github.com/spf13/cobra"

// newVMCommand builds the "vm" subcommand.
func newVMCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "vm",
		Short: "Manage virtual machines",
		Long:  "Creates, starts, and stops the virtual machines described in the config file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Logger.Info("vm not yet implemented",
				"dry_run", opts.DryRun,
				"inventory", opts.Inventory,
				"config", opts.Config,
			)
			return nil
		},
	}
}
