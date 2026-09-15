//go:build !linux

package cgroup

import "fmt"

// Create is a stub on non-Linux builds.
func (c *Controller) Create(name string, lim Limits) error {
	return fmt.Errorf("create cgroup %q: %w", c.Path(name), ErrUnsupported)
}

// AddPID is a stub on non-Linux builds.
func (c *Controller) AddPID(name string, pid int) error {
	return fmt.Errorf("add pid %d to cgroup %q: %w", pid, c.Path(name), ErrUnsupported)
}

// Delete is a stub on non-Linux builds.
func (c *Controller) Delete(name string) error {
	return fmt.Errorf("delete cgroup %q: %w", c.Path(name), ErrUnsupported)
}

// MemoryStats is a stub on non-Linux builds.
func (c *Controller) MemoryStats(name string) (MemoryStats, error) {
	return MemoryStats{}, fmt.Errorf("read memory stats for %q: %w", c.Path(name), ErrUnsupported)
}

// CPUStats is a stub on non-Linux builds.
func (c *Controller) CPUStats(name string) (CPUStats, error) {
	return CPUStats{}, fmt.Errorf("read cpu stats for %q: %w", c.Path(name), ErrUnsupported)
}
