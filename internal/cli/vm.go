package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/infraforge/infraforge/internal/config"
	"github.com/infraforge/infraforge/internal/virt"
	"github.com/spf13/cobra"
)

// newVMCommand builds the "vm" subcommand with its own subcommands.
func newVMCommand(opts *Options) *cobra.Command {
	var backendName string
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "Manage virtual machines",
		Long:  "Starts, stops, inspects, and lists virtual machines via a pluggable backend.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.PersistentFlags().StringVar(&backendName, "backend", "", "VM backend: docker or libvirt (default: auto-detect)")
	cmd.AddCommand(
		newVMStartCommand(opts, &backendName),
		newVMStopCommand(opts, &backendName),
		newVMInspectCommand(opts, &backendName),
		newVMListCommand(opts, &backendName),
	)
	return cmd
}

func newVMStartCommand(opts *Options, backendName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a VM defined in the config file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVMStart(cmd.Context(), cmd, opts, *backendName, args[0])
		},
	}
}

func runVMStart(ctx context.Context, cmd *cobra.Command, opts *Options, backendName, name string) error {
	logger := opts.Logger.With("command", "vm", "subcommand", "start")

	cfg, err := config.Load(opts.Config)
	if err != nil {
		return exitErrorf(ExitConfigError, "load config: %w", err)
	}
	host, err := findHost(cfg, name)
	if err != nil {
		return exitErrorf(ExitConfigError, "%w", err)
	}
	backend, err := virt.New(backendName)
	if err != nil {
		return err
	}
	vm, err := backend.Start(ctx, virt.VM{Name: host.Name, Image: host.Image})
	if err != nil {
		return err
	}
	logger.Info("vm started", "name", vm.Name, "state", vm.State, "pid", vm.PID, "namespaces", len(vm.Namespaces))
	printVM(cmd.OutOrStdout(), vm)
	return nil
}

func newVMStopCommand(opts *Options, backendName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := opts.Logger.With("command", "vm", "subcommand", "stop")
			backend, err := virt.New(*backendName)
			if err != nil {
				return err
			}
			if err := backend.Stop(cmd.Context(), args[0]); err != nil {
				return err
			}
			logger.Info("vm stopped", "name", args[0])
			fmt.Fprintf(cmd.OutOrStdout(), "stopped %s\n", args[0])
			return nil
		},
	}
}

func newVMInspectCommand(opts *Options, backendName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <name>",
		Short: "Inspect a VM, including its namespace IDs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := opts.Logger.With("command", "vm", "subcommand", "inspect")
			backend, err := virt.New(*backendName)
			if err != nil {
				return err
			}
			vm, err := backend.Inspect(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			logger.Info("vm inspected", "name", vm.Name, "state", vm.State, "pid", vm.PID, "namespaces", len(vm.Namespaces))
			printVM(cmd.OutOrStdout(), vm)
			return nil
		},
	}
}

func newVMListCommand(opts *Options, backendName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List VMs managed by the backend",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger := opts.Logger.With("command", "vm", "subcommand", "list")
			backend, err := virt.New(*backendName)
			if err != nil {
				return err
			}
			vms, err := backend.List(cmd.Context())
			if err != nil {
				return err
			}
			logger.Info("vms listed", "count", len(vms))
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "NAME\tSTATE\tIMAGE")
			for _, vm := range vms {
				fmt.Fprintf(out, "%s\t%s\t%s\n", vm.Name, vm.State, vm.Image)
			}
			return nil
		},
	}
}

func printVM(out io.Writer, vm *virt.VM) {
	fmt.Fprintf(out, "name:  %s\n", vm.Name)
	fmt.Fprintf(out, "state: %s\n", vm.State)
	fmt.Fprintf(out, "image: %s\n", vm.Image)
	fmt.Fprintf(out, "pid:   %d\n", vm.PID)
	if len(vm.Namespaces) == 0 {
		fmt.Fprintln(out, "namespaces: n/a")
		return
	}
	fmt.Fprintln(out, "namespaces:")
	for _, kind := range virt.SortedNamespaceNames(vm.Namespaces) {
		fmt.Fprintf(out, "  %s: %s\n", kind, vm.Namespaces[kind])
	}
}

func findHost(cfg *config.Config, name string) (*config.Host, error) {
	for i := range cfg.Hosts {
		if cfg.Hosts[i].Name == name {
			return &cfg.Hosts[i], nil
		}
	}
	return nil, fmt.Errorf("host %q not found in config", name)
}
