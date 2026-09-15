package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/infraforge/infraforge/internal/cgroup"
	"github.com/infraforge/infraforge/internal/config"
	"github.com/spf13/cobra"
)

// sampleInterval is how often the limit command samples cgroup stats while
// the limited command runs.
const sampleInterval = 500 * time.Millisecond

type limitFlags struct {
	name      string
	cpuMax    string
	memoryMax string
	pidsMax   int64
}

// newLimitCommand builds the "limit" subcommand.
func newLimitCommand(opts *Options) *cobra.Command {
	var flags limitFlags
	cmd := &cobra.Command{
		Use:   "limit [flags] -- <command...>",
		Short: "Run a command inside a cgroup v2 with resource limits",
		Long: "Creates a cgroup under /sys/fs/cgroup/infraforge/<name> with the given cpu.max, memory.max/memory.high, and pids.max limits, forks <command...> into it, samples memory.events and cpu.stat while it runs, and reports whether the command was throttled or OOM-killed.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLimit(cmd.Context(), cmd, opts, flags, args)
		},
	}
	cmd.Flags().StringVar(&flags.name, "name", "", "cgroup name (required)")
	cmd.Flags().StringVar(&flags.cpuMax, "cpu-max", "", "CPU limit as a percentage of one core, e.g. 25%")
	cmd.Flags().StringVar(&flags.memoryMax, "memory-max", "", "memory limit as a byte string, e.g. 256M")
	cmd.Flags().Int64Var(&flags.pidsMax, "pids-max", 0, "maximum number of processes (0 = unlimited)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func runLimit(ctx context.Context, cmd *cobra.Command, opts *Options, flags limitFlags, args []string) error {
	logger := opts.Logger.With("command", "limit", "cgroup", flags.name)
	out := cmd.OutOrStdout()

	limits := cgroup.Limits{PIDsMax: flags.pidsMax}
	if flags.cpuMax != "" {
		v, err := cgroup.ParseCPUPercent(flags.cpuMax)
		if err != nil {
			return fmt.Errorf("invalid --cpu-max: %w", err)
		}
		limits.CPUMax = v
	}
	if flags.memoryMax != "" {
		bytes, err := config.ParseMemory(flags.memoryMax)
		if err != nil {
			return fmt.Errorf("invalid --memory-max: %w", err)
		}
		limits.MemoryMax = bytes
	}

	ctrl := &cgroup.Controller{}
	if err := ctrl.Create(flags.name, limits); err != nil {
		return err
	}
	defer func() {
		if err := ctrl.Delete(flags.name); err != nil {
			logger.Warn("failed to remove cgroup", "error", err)
		}
	}()
	logger.Info("cgroup created",
		"path", ctrl.Path(flags.name),
		"cpu_max", limits.CPUMax,
		"memory_max", limits.MemoryMax,
		"pids_max", limits.PIDsMax,
	)

	child := exec.CommandContext(ctx, args[0], args[1:]...)
	child.Stdin = cmd.InOrStdin()
	child.Stdout = out
	child.Stderr = cmd.ErrOrStderr()
	if err := child.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}
	if err := ctrl.AddPID(flags.name, child.Process.Pid); err != nil {
		_ = child.Process.Kill()
		_, _ = child.Process.Wait()
		return fmt.Errorf("move pid %d into cgroup: %w", child.Process.Pid, err)
	}

	sampler := newSampler(ctrl, flags.name, logger, sampleInterval)
	sampler.start()
	runErr := child.Wait()
	sampler.stop()

	mem, memErr := ctrl.MemoryStats(flags.name)
	cpu, cpuErr := ctrl.CPUStats(flags.name)

	oomKills := int64(0)
	highEvents := int64(0)
	current := int64(0)
	if memErr == nil {
		current = mem.Current
		oomKills = mem.Events["oom_kill"]
		highEvents = mem.Events["high"]
	} else {
		logger.Warn("failed to read memory stats", "error", memErr)
	}
	throttled := int64(0)
	if cpuErr == nil {
		throttled = cpu.ThrottledUsec
	} else {
		logger.Warn("failed to read cpu stats", "error", cpuErr)
	}

	exitLine := "0"
	if runErr != nil {
		exitLine = runErr.Error()
	}
	verdict := "ok"
	switch {
	case oomKills > 0:
		verdict = "oom-killed"
	case throttled > 0:
		verdict = "throttled"
	}

	fmt.Fprintf(out, "limit summary\n")
	fmt.Fprintf(out, "  cgroup:   %s\n", ctrl.Path(flags.name))
	fmt.Fprintf(out, "  command:  %s\n", strings.Join(args, " "))
	fmt.Fprintf(out, "  exit:     %s\n", exitLine)
	fmt.Fprintf(out, "  cpu:      throttled_usec=%d\n", throttled)
	fmt.Fprintf(out, "  memory:   current=%d peak=%d oom_kill=%d high_events=%d\n",
		current, sampler.peakCurrent(), oomKills, highEvents)
	fmt.Fprintf(out, "  verdict:  %s\n", verdict)

	logger.Info("limit complete",
		"verdict", verdict,
		"throttled_usec", throttled,
		"oom_kill", oomKills,
		"memory_current", current,
		"memory_peak", sampler.peakCurrent(),
	)

	switch {
	case oomKills > 0:
		return fmt.Errorf("command was OOM-killed in cgroup %s", ctrl.Path(flags.name))
	case runErr != nil:
		return fmt.Errorf("limited command failed: %w", runErr)
	}
	return nil
}

// sampler periodically reads cgroup stats and logs them, tracking the peak
// memory usage observed.
type sampler struct {
	ctrl     *cgroup.Controller
	name     string
	logger   *slog.Logger
	interval time.Duration

	stopCh chan struct{}
	done   chan struct{}

	mu   sync.Mutex
	peak int64
}

func newSampler(ctrl *cgroup.Controller, name string, logger *slog.Logger, interval time.Duration) *sampler {
	return &sampler{
		ctrl:     ctrl,
		name:     name,
		logger:   logger,
		interval: interval,
	}
}

func (s *sampler) start() {
	s.stopCh = make(chan struct{})
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
			}
			mem, memErr := s.ctrl.MemoryStats(s.name)
			cpu, cpuErr := s.ctrl.CPUStats(s.name)
			if memErr != nil || cpuErr != nil {
				s.logger.Debug("cgroup sample failed", "mem_error", memErr, "cpu_error", cpuErr)
				continue
			}
			s.mu.Lock()
			if mem.Current > s.peak {
				s.peak = mem.Current
			}
			s.mu.Unlock()
			s.logger.Info("cgroup sampled",
				"memory_current", mem.Current,
				"oom_kill", mem.Events["oom_kill"],
				"memory_high_events", mem.Events["high"],
				"throttled_usec", cpu.ThrottledUsec,
			)
		}
	}()
}

func (s *sampler) stop() {
	close(s.stopCh)
	<-s.done
}

func (s *sampler) peakCurrent() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peak
}
