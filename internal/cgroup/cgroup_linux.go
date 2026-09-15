//go:build linux

package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// controllersEnabled are the controllers infraforge enables on its group
// cgroup so nested cgroups expose cpu.max, memory.*, and pids.max.
const controllersEnabled = "+cpu +memory +pids"

// Create creates the cgroup for name and applies the limits. The parent
// group's subtree_control is extended so controller files exist in the new
// cgroup. Creating an existing cgroup is not an error; limits are re-applied.
func (c *Controller) Create(name string, lim Limits) error {
	groupDir := filepath.Join(c.root(), c.group())
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		return fmt.Errorf("create group dir %q: %w", groupDir, err)
	}
	if err := writeFile(filepath.Join(groupDir, "cgroup.subtree_control"), controllersEnabled); err != nil {
		return fmt.Errorf("enable controllers on %q: %w", groupDir, err)
	}

	dir := c.Path(name)
	if err := os.Mkdir(dir, 0o755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("create cgroup %q: %w", dir, err)
	}

	type write struct{ file, value string }
	writes := make([]write, 0, 4)
	if lim.CPUMax != "" {
		writes = append(writes, write{"cpu.max", lim.CPUMax})
	}
	if lim.MemoryMax > 0 {
		writes = append(writes, write{"memory.max", strconv.FormatInt(lim.MemoryMax, 10)})
		high := lim.MemoryHigh
		if high == 0 {
			high = lim.MemoryMax
		}
		writes = append(writes, write{"memory.high", strconv.FormatInt(high, 10)})
	}
	if lim.PIDsMax > 0 {
		writes = append(writes, write{"pids.max", strconv.FormatInt(lim.PIDsMax, 10)})
	}
	for _, w := range writes {
		if err := writeFile(filepath.Join(dir, w.file), w.value); err != nil {
			return fmt.Errorf("write %s to cgroup %q: %w", w.file, dir, err)
		}
	}
	return nil
}

// AddPID moves a process into the cgroup by writing its PID to cgroup.procs.
func (c *Controller) AddPID(name string, pid int) error {
	if err := writeFile(filepath.Join(c.Path(name), "cgroup.procs"), strconv.Itoa(pid)); err != nil {
		return fmt.Errorf("move pid %d into cgroup %q: %w", pid, c.Path(name), err)
	}
	return nil
}

// Delete removes the cgroup. The cgroup must not contain any processes.
func (c *Controller) Delete(name string) error {
	dir := c.Path(name)
	if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove cgroup %q: %w", dir, err)
	}
	return nil
}

// MemoryStats reads memory.current and memory.events for the cgroup.
func (c *Controller) MemoryStats(name string) (MemoryStats, error) {
	var stats MemoryStats
	dir := c.Path(name)
	current, err := os.ReadFile(filepath.Join(dir, "memory.current"))
	if err != nil {
		return stats, fmt.Errorf("read memory.current: %w", err)
	}
	stats.Current, err = parseIntFile(current)
	if err != nil {
		return stats, fmt.Errorf("parse memory.current: %w", err)
	}
	events, err := os.ReadFile(filepath.Join(dir, "memory.events"))
	if err != nil {
		return stats, fmt.Errorf("read memory.events: %w", err)
	}
	stats.Events, err = parseKeyValue(events)
	if err != nil {
		return stats, fmt.Errorf("parse memory.events: %w", err)
	}
	return stats, nil
}

// CPUStats reads cpu.stat for the cgroup.
func (c *Controller) CPUStats(name string) (CPUStats, error) {
	var stats CPUStats
	dir := c.Path(name)
	data, err := os.ReadFile(filepath.Join(dir, "cpu.stat"))
	if err != nil {
		return stats, fmt.Errorf("read cpu.stat: %w", err)
	}
	values, err := parseKeyValue(data)
	if err != nil {
		return stats, fmt.Errorf("parse cpu.stat: %w", err)
	}
	stats.ThrottledUsec = values["throttled_usec"]
	return stats, nil
}

func writeFile(path, value string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(value); err != nil {
		return err
	}
	return nil
}
