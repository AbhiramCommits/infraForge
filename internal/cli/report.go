package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/infraforge/infraforge/internal/report"
	"github.com/spf13/cobra"
)

// newReportCommand builds the "report" subcommand.
func newReportCommand(opts *Options) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Summarize the most recent runs",
		Long:  "Reads the reports written by provision and limit runs and prints a summary of hosts provisioned, per-host task counts, tasks that changed state, limits applied, and throttle/OOM counters.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReport(cmd, opts, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "output format: table or json")
	return cmd
}

func runReport(cmd *cobra.Command, opts *Options, format string) error {
	logger := opts.Logger.With("command", "report")
	out := cmd.OutOrStdout()
	reportsDir := reportsDir(opts)

	if format != "table" && format != "json" {
		return exitErrorf(ExitConfigError, "invalid --format %q: must be \"table\" or \"json\"", format)
	}

	reports, err := report.LoadAll(reportsDir)
	if err != nil {
		return fmt.Errorf("load reports: %w", err)
	}
	logger.Info("reports loaded", "count", len(reports), "dir", reportsDir)

	summary := report.Summary{}
	for _, r := range reports {
		switch r.Kind {
		case report.KindProvision:
			if summary.Provision == nil {
				summary.Provision = r
			}
		case report.KindLimit:
			if summary.Limit == nil {
				summary.Limit = r
			}
		}
	}
	if summary.Provision == nil && summary.Limit == nil {
		fmt.Fprintln(out, "no reports found")
		return nil
	}

	if format == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(summary); err != nil {
			return fmt.Errorf("encode report: %w", err)
		}
		return nil
	}

	printProvisionTable(out, summary.Provision)
	printLimitTable(out, summary.Limit)
	return nil
}

func printProvisionTable(out io.Writer, r *report.Report) {
	if r == nil {
		return
	}
	p := r.Provision
	provisioned := 0
	for _, h := range p.Hosts {
		if h.OK > 0 {
			provisioned++
		}
	}
	fmt.Fprintf(out, "provision %s (wall time %.1fs)\n", r.RunID, r.DurationSeconds)
	fmt.Fprintf(out, "  hosts provisioned: %d\n", provisioned)
	fmt.Fprintf(out, "  per-host task counts:\n")
	for _, h := range p.Hosts {
		fmt.Fprintf(out, "    %-16s ok=%-3d changed=%-3d failed=%-3d unreachable=%d\n",
			h.Name, h.OK, h.Changed, h.Failed, h.Unreachable)
	}
	fmt.Fprintf(out, "  tasks that changed state (%d):\n", len(p.ChangedTasks))
	if len(p.ChangedTasks) == 0 {
		fmt.Fprintln(out, "    (none)")
	}
	for _, name := range p.ChangedTasks {
		fmt.Fprintf(out, "    - %s\n", name)
	}
}

func printLimitTable(out io.Writer, r *report.Report) {
	if r == nil {
		return
	}
	l := r.Limit
	cpu := l.Limits.CPUMax
	if cpu == "" {
		cpu = "max"
	}
	pids := l.Limits.PIDsMax
	fmt.Fprintf(out, "limit %s (wall time %.1fs)\n", r.RunID, r.DurationSeconds)
	fmt.Fprintf(out, "  cgroup:      %s\n", l.Cgroup)
	fmt.Fprintf(out, "  limits:      cpu=%s memory=%s pids=%d\n", cpu, report.FormatBytes(l.Limits.MemoryMax), pids)
	fmt.Fprintf(out, "  throttled:   %d us\n", l.ThrottledUsec)
	fmt.Fprintf(out, "  oom kills:   %d\n", l.OOMKills)
	fmt.Fprintf(out, "  high events: %d\n", l.HighEvents)
	fmt.Fprintf(out, "  peak memory: %s\n", report.FormatBytes(l.PeakMemory))
	fmt.Fprintf(out, "  verdict:     %s\n", l.Verdict)
}
