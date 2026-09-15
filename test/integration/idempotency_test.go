//go:build integration

// Package integration contains end-to-end tests that require Linux with
// cgroups v2, Docker, and Ansible. Run them with:
//
//	make test-integration
package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/infraforge/infraforge/internal/report"
)

const provisionTimeout = 5 * time.Minute

var (
	repoRoot      string
	infraforgeBin string
	burnerBin     string
	reportsDir    string
)

func TestMain(m *testing.M) {
	var err error
	repoRoot, err = filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve repo root:", err)
		os.Exit(1)
	}
	reportsDir = filepath.Join(repoRoot, "reports")

	buildDir, err := os.MkdirTemp("", "infraforge-it-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create build dir:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(buildDir)

	infraforgeBin = filepath.Join(buildDir, "infraforge")
	burnerBin = filepath.Join(buildDir, "burner")
	for _, b := range []struct{ out, pkg string }{
		{infraforgeBin, "./cmd/infraforge"},
		{burnerBin, "./hack"},
	} {
		cmd := exec.Command("go", "build", "-o", b.out, b.pkg)
		cmd.Dir = repoRoot
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "build %s: %v\n", b.pkg, err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

// runCmd executes name with args in repoRoot, capturing combined output.
func runCmd(t *testing.T, timeout time.Duration, name string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = repoRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// latestReport returns the newest report of the given kind, failing the test
// if none exists.
func latestReport(t *testing.T, kind string) *report.Report {
	t.Helper()
	reports, err := report.LoadAll(reportsDir)
	if err != nil {
		t.Fatalf("load reports: %v", err)
	}
	for _, r := range reports {
		if r.Kind == kind {
			return r
		}
	}
	t.Fatalf("no %s report found in %s", kind, reportsDir)
	return nil
}

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Fatalf("docker daemon not reachable: %v", err)
	}
}

func requireCgroupV2(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err != nil {
		t.Fatalf("cgroup v2 unified hierarchy not mounted: %v", err)
	}
}

func writeConfig(t *testing.T, dir string) string {
	t.Helper()
	cfg := `hosts:
  - name: web-01
    image: ubuntu:22.04
    ssh_port: 2250
    limits:
      cpu_max: 50
      memory_max: 256M
      pids_max: 128
  - name: db-01
    image: ubuntu:22.04
    ssh_port: 2251
    limits:
      cpu_max: 25
      memory_max: 128M
`
	path := filepath.Join(dir, "infraforge.yaml")
	if err := os.WriteFile(path, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProvisionIdempotent(t *testing.T) {
	requireDocker(t)

	dir := t.TempDir()
	configPath := writeConfig(t, dir)
	inventoryPath := filepath.Join(dir, "inventory.yml")

	t.Cleanup(func() {
		for _, name := range []string{"web-01", "db-01"} {
			_ = exec.Command("docker", "rm", "-f", name).Run()
		}
		_ = exec.Command("docker", "rmi", "-f", "infraforge/node:ubuntu-22.04").Run()
	})

	// Run 1: everything is created, so every host must report changes.
	out, err := runCmd(t, provisionTimeout, infraforgeBin, "provision",
		"--config", configPath,
		"--inventory", inventoryPath,
		"--reports-dir", reportsDir,
	)
	if err != nil {
		t.Fatalf("provision run 1 failed: %v\n%s", err, out)
	}
	inventory1, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("read inventory after run 1: %v", err)
	}

	rep1 := latestReport(t, report.KindProvision)
	if len(rep1.Provision.Hosts) < 2 {
		t.Fatalf("run 1 report has %d hosts, want >= 2: %+v", len(rep1.Provision.Hosts), rep1.Provision.Hosts)
	}
	for _, h := range rep1.Provision.Hosts {
		if h.Changed == 0 {
			t.Errorf("run 1: host %q changed = 0, want > 0", h.Name)
		}
	}

	// Run 2: must be fully idempotent.
	out, err = runCmd(t, provisionTimeout, infraforgeBin, "provision",
		"--config", configPath,
		"--inventory", inventoryPath,
		"--reports-dir", reportsDir,
	)
	if err != nil {
		t.Fatalf("provision run 2 failed: %v\n%s", err, out)
	}
	inventory2, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("read inventory after run 2: %v", err)
	}
	if !bytes.Equal(inventory1, inventory2) {
		t.Fatalf("generated inventory differs between runs:\n--- run 1\n%s\n--- run 2\n%s", inventory1, inventory2)
	}

	rep2 := latestReport(t, report.KindProvision)
	var offenders []string
	for _, h := range rep2.Provision.Hosts {
		if h.Changed != 0 {
			offenders = append(offenders, fmt.Sprintf("%s (changed=%d)", h.Name, h.Changed))
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("provision is not idempotent; hosts with changes on run 2: %v; offending task names: %v",
			offenders, rep2.Provision.ChangedTasks)
	}
	if len(rep2.Provision.ChangedTasks) > 0 {
		t.Fatalf("provision is not idempotent; tasks that reported changed on run 2: %v", rep2.Provision.ChangedTasks)
	}
}

func TestLimitOOMKill(t *testing.T) {
	requireCgroupV2(t)

	cgroupName := "infraforge-it-oom"
	t.Cleanup(func() {
		_ = os.Remove("/sys/fs/cgroup/infraforge/" + cgroupName)
		_ = os.Remove("/sys/fs/cgroup/infraforge")
	})

	out, err := runCmd(t, 2*time.Minute, infraforgeBin,
		"limit",
		"--name", cgroupName,
		"--memory-max", "128M",
		"--reports-dir", reportsDir,
		"--", burnerBin, "-mem-mb", "256", "-cpu", "0",
	)
	if err == nil {
		t.Fatalf("limit succeeded, want OOM kill:\n%s", out)
	}
	if !strings.Contains(out, "verdict:  oom-killed") {
		t.Fatalf("output missing oom-killed verdict:\n%s", out)
	}

	rep := latestReport(t, report.KindLimit)
	if rep.Limit == nil {
		t.Fatalf("limit report missing limit section: %+v", rep)
	}
	if rep.Limit.OOMKills < 1 {
		t.Fatalf("memory.events oom_kill = %d, want >= 1 (report: %+v)", rep.Limit.OOMKills, rep.Limit)
	}
	if rep.Limit.Limits.MemoryMax != 128<<20 {
		t.Errorf("reported memory limit = %d, want %d", rep.Limit.Limits.MemoryMax, 128<<20)
	}
}
