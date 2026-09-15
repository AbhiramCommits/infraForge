package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"time"

	"github.com/infraforge/infraforge/internal/ansible"
	"github.com/infraforge/infraforge/internal/config"
	"github.com/infraforge/infraforge/internal/report"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProvision(cmd.Context(), opts)
		},
	}
}

func runProvision(ctx context.Context, opts *Options) error {
	started := time.Now()
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
	result, runErr := runner.Run(ctx)
	if result != nil {
		saveProvisionReport(opts, started, result, logger)
	}
	if runErr != nil {
		if result != nil {
			logger.Error("provisioning completed with problems",
				"hosts", len(result.Hosts),
				"changed_tasks", len(result.ChangedTasks),
			)
		}
		return runErr
	}
	logger.Info("provisioning complete",
		"hosts", len(result.Hosts),
		"changed_tasks", len(result.ChangedTasks),
		"changed_task_names", result.ChangedTasks,
	)
	return nil
}

func saveProvisionReport(opts *Options, started time.Time, result *ansible.Result, logger *slog.Logger) {
	hosts := make([]report.HostResult, 0, len(result.Hosts))
	for name, stats := range result.Hosts {
		hosts = append(hosts, report.HostResult{
			Name:        name,
			OK:          stats.OK,
			Changed:     stats.Changed,
			Unreachable: stats.Unreachable,
			Failed:      stats.Failed,
		})
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })

	dir := reportsDir(opts)
	rec := &report.Report{
		RunID:           opts.RunID,
		Kind:            report.KindProvision,
		StartedAt:       started,
		DurationSeconds: time.Since(started).Seconds(),
		Provision: &report.Provision{
			Hosts:        hosts,
			ChangedTasks: result.ChangedTasks,
		},
	}
	if err := report.Save(dir, rec); err != nil {
		logger.Warn("failed to write report", "error", err)
		return
	}
	logger.Info("report written", "path", filepath.Join(dir, opts.RunID+".json"))
}

// reportsDir returns the directory run reports are written to.
func reportsDir(opts *Options) string {
	if opts.ReportsDir != "" {
		return opts.ReportsDir
	}
	return "reports"
}
