package cli

import "github.com/spf13/cobra"

// newProvisionCommand builds the "provision" subcommand.
func newProvisionCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "provision",
		Short: "Provision hosts using Ansible",
		Long:  "Reads the config file and drives Ansible to bring each host into the desired state.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Logger.Info("provision not yet implemented",
				"dry_run", opts.DryRun,
				"inventory", opts.Inventory,
				"config", opts.Config,
			)
			return nil
		},
	}
}
