package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/infraforge/infraforge/internal/ansible"
	"github.com/infraforge/infraforge/internal/config"
	"github.com/spf13/cobra"
)

const defaultPlaybook = "deploy/playbooks/site.yml"

// newProvisionCommand builds the "provision" subcommand.
func newProvisionCommand(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "provision",
		Short: "Provision hosts using Ansible",
		Long:  "Reads the config file, generates the Ansible inventory, and runs the site playbook to bring each host into the desired state.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProvision(cmd.Context(), opts)
		},
	}
}

func runProvision(ctx context.Context, opts *Options) error {
	logger := opts.Logger.With("command", "provision")

	cfg, err := config.Load(opts.Config)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if len(cfg.Hosts) == 0 {
		return errors.New("config defines no hosts; nothing to provision")
	}

	if err := ansible.GenerateInventory(cfg, opts.Inventory); err != nil {
		return fmt.Errorf("generate inventory: %w", err)
	}
	logger.Info("inventory generated", "path", opts.Inventory, "hosts", len(cfg.Hosts))

	runner := &ansible.Runner{
		Playbook:  defaultPlaybook,
		Inventory: opts.Inventory,
		DryRun:    opts.DryRun,
		Logger:    logger,
	}
	result, err := runner.Run(ctx)
	if err != nil {
		if result != nil {
			logger.Error("provisioning completed with problems",
				"hosts", len(result.Hosts),
				"changed_tasks", len(result.ChangedTasks),
			)
		}
		return err
	}
	logger.Info("provisioning complete",
		"hosts", len(result.Hosts),
		"changed_tasks", len(result.ChangedTasks),
		"changed_task_names", result.ChangedTasks,
	)
	return nil
}
