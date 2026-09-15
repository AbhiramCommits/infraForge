// Package cgroup applies Linux cgroup v2 resource limits by writing
// directly to the unified hierarchy filesystem.
package cgroup

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrUnsupported is returned by every operation on non-Linux builds.
var ErrUnsupported = errors.New("cgroups v2 requires Linux")

// DefaultRoot is the standard cgroup v2 unified hierarchy mount point.
const DefaultRoot = "/sys/fs/cgroup"

// DefaultGroup is the parent cgroup all infraforge cgroups live under.
const DefaultGroup = "infraforge"

// Limits describes the resource limits written to a cgroup.
type Limits struct {
	// CPUMax is the cpu.max value, e.g. "50000 100000" for 50% of one core,
	// or "" for no limit.
	CPUMax string
	// MemoryMax is the memory.max value in bytes; 0 means unlimited.
	MemoryMax int64
	// MemoryHigh is the memory.high throttling threshold in bytes; 0 means
	// unlimited. Defaults to MemoryMax when left unset.
	MemoryHigh int64
	// PIDsMax caps the number of processes; 0 means unlimited.
	PIDsMax int64
}

// CPUStats reports CPU throttle accounting for a cgroup.
type CPUStats struct {
	// ThrottledUsec is the total time, in microseconds, the cgroup was
	// throttled by the CPU controller.
	ThrottledUsec int64
}

// MemoryStats reports memory usage and events for a cgroup.
type MemoryStats struct {
	// Current is the current memory usage in bytes (memory.current).
	Current int64
	// Peak is the highest observed usage in bytes (memory.peak).
	Peak int64
	// Events maps memory.events counters (e.g. "oom_kill", "high") to their
	// values.
	Events map[string]int64
}

// Controller manages cgroups under the Root/Group directory.
type Controller struct {
	// Root is the cgroup v2 hierarchy mount point. Defaults to
	// /sys/fs/cgroup.
	Root string
	// Group is the parent cgroup name. Defaults to "infraforge".
	Group string
}

func (c *Controller) root() string {
	if c.Root != "" {
		return c.Root
	}
	return DefaultRoot
}

func (c *Controller) group() string {
	if c.Group != "" {
		return c.Group
	}
	return DefaultGroup
}

// Path returns the absolute filesystem path for a named cgroup.
func (c *Controller) Path(name string) string {
	return filepath.Join(c.root(), c.group(), name)
}

// ParseCPUPercent converts "50%" or "50" into a cpu.max "quota period"
// value representing that share of a single CPU core. The period is fixed
// at 100000 microseconds, so 50% becomes "50000 100000".
func ParseCPUPercent(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	p, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return "", fmt.Errorf("invalid cpu percentage %q", s)
	}
	if p < 0 || p > 100 {
		return "", fmt.Errorf("cpu percentage %v out of range [0, 100]", p)
	}
	quota := int64(math.Round(p * 1000))
	if quota == 0 {
		return "", fmt.Errorf("cpu percentage %q is too small to enforce", s)
	}
	return fmt.Sprintf("%d 100000", quota), nil
}

// parseKeyValue parses "key value" lines like those found in memory.events
// and cpu.stat.
func parseKeyValue(data []byte) (map[string]int64, error) {
	out := make(map[string]int64)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("malformed line %q", line)
		}
		v, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("malformed value in %q: %w", line, err)
		}
		out[fields[0]] = v
	}
	return out, nil
}

// parseIntFile parses a file holding a single decimal integer, such as
// memory.current.
func parseIntFile(data []byte) (int64, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", data, err)
	}
	return v, nil
}
