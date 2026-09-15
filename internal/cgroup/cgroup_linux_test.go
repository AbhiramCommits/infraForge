//go:build linux

package cgroup

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// requireCgroupV2 skips unless we run as root on a cgroup v2 host.
func requireCgroupV2(t *testing.T) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("requires root")
	}
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err != nil {
		t.Skip("cgroup v2 unified hierarchy not available")
	}
}

func TestCreateAndApplyLimits(t *testing.T) {
	requireCgroupV2(t)
	c := &Controller{}
	name := "infraforge-test-limits"
	t.Cleanup(func() {
		if err := c.Delete(name); err != nil {
			t.Logf("cleanup: %v", err)
		}
	})

	err := c.Create(name, Limits{
		CPUMax:    "25000 100000",
		MemoryMax: 16 << 20,
		PIDsMax:   64,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	check := func(file, want string) {
		t.Helper()
		data, err := os.ReadFile(c.Path(name) + "/" + file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if got := strings.TrimSpace(string(data)); got != want {
			t.Errorf("%s = %q, want %q", file, got, want)
		}
	}
	check("cpu.max", "25000 100000")
	check("memory.max", "16777216")
	check("memory.high", "16777216")
	check("pids.max", "64")
}

func TestAddPIDAndStats(t *testing.T) {
	requireCgroupV2(t)
	c := &Controller{}
	name := "infraforge-test-pid"
	t.Cleanup(func() {
		if err := c.Delete(name); err != nil {
			t.Logf("cleanup: %v", err)
		}
	})

	if err := c.Create(name, Limits{CPUMax: "25000 100000", MemoryMax: 64 << 20}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	if err := c.AddPID(name, cmd.Process.Pid); err != nil {
		t.Fatalf("AddPID() error = %v", err)
	}
	data, err := os.ReadFile(c.Path(name) + "/cgroup.procs")
	if err != nil {
		t.Fatalf("read cgroup.procs: %v", err)
	}
	if !strings.Contains(string(data), strconv.Itoa(cmd.Process.Pid)) {
		t.Errorf("cgroup.procs = %q, want pid %d", data, cmd.Process.Pid)
	}

	mem, err := c.MemoryStats(name)
	if err != nil {
		t.Fatalf("MemoryStats() error = %v", err)
	}
	if mem.Current <= 0 {
		t.Errorf("memory.current = %d, want > 0", mem.Current)
	}
	if _, ok := mem.Events["oom_kill"]; !ok {
		t.Errorf("memory.events missing oom_kill: %v", mem.Events)
	}

	cpu, err := c.CPUStats(name)
	if err != nil {
		t.Fatalf("CPUStats() error = %v", err)
	}
	if cpu.ThrottledUsec < 0 {
		t.Errorf("throttled_usec = %d, want >= 0", cpu.ThrottledUsec)
	}
}
