package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/infraforge/infraforge/internal/report"
)

func seedReports(t *testing.T, dir string) {
	t.Helper()
	provision := &report.Report{
		RunID:           "run-provision",
		Kind:            report.KindProvision,
		StartedAt:       time.Now().Add(-2 * time.Minute),
		DurationSeconds: 42.5,
		Provision: &report.Provision{
			Hosts: []report.HostResult{
				{Name: "db-01", OK: 5, Changed: 1},
				{Name: "web-01", OK: 5, Changed: 3},
			},
			ChangedTasks: []string{"Install base packages", "Start the burner via supervisor"},
		},
	}
	limit := &report.Report{
		RunID:           "run-limit",
		Kind:            report.KindLimit,
		StartedAt:       time.Now().Add(-time.Minute),
		DurationSeconds: 4.25,
		Limit: &report.Limit{
			Cgroup:        "demo-oom",
			Limits:        report.Limits{CPUMax: "25000 100000", MemoryMax: 128 << 20, PIDsMax: 64},
			ThrottledUsec: 1200000,
			OOMKills:      1,
			HighEvents:    17,
			PeakMemory:    128 << 20,
			Verdict:       "oom-killed",
		},
	}
	if err := report.Save(dir, provision); err != nil {
		t.Fatalf("seed provision report: %v", err)
	}
	if err := report.Save(dir, limit); err != nil {
		t.Fatalf("seed limit report: %v", err)
	}
}

func TestReportTable(t *testing.T) {
	dir := t.TempDir()
	seedReports(t, dir)

	var out bytes.Buffer
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"report", "--reports-dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, want := range []string{
		"provision run-provision",
		"hosts provisioned: 2",
		"web-01", "db-01",
		"changed=3",
		"Install base packages",
		"Start the burner via supervisor",
		"limit run-limit",
		"cgroup:      demo-oom",
		"cpu=25000 100000",
		"memory=128.0 MiB",
		"pids=64",
		"oom kills:   1",
		"high events: 17",
		"verdict:     oom-killed",
		"wall time 42.5s",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report output missing %q:\n%s", want, out.String())
		}
	}
}

func TestReportJSON(t *testing.T) {
	dir := t.TempDir()
	seedReports(t, dir)

	var out bytes.Buffer
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"report", "--reports-dir", dir, "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var summary report.Summary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if summary.Provision == nil || summary.Provision.RunID != "run-provision" {
		t.Errorf("summary missing provision record: %+v", summary)
	}
	if summary.Provision.Provision.Hosts[0].Name != "db-01" {
		t.Errorf("host order = %v, want sorted", summary.Provision.Provision.Hosts)
	}
	if summary.Limit == nil || summary.Limit.Limit.OOMKills != 1 {
		t.Errorf("summary missing limit record: %+v", summary)
	}
}

func TestReportEmpty(t *testing.T) {
	var out bytes.Buffer
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"report", "--reports-dir", t.TempDir()})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "no reports found") {
		t.Errorf("output = %q, want \"no reports found\"", out.String())
	}
}

func TestReportInvalidFormat(t *testing.T) {
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"report", "--format", "yaml"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want error for invalid --format")
	}
	if !strings.Contains(err.Error(), "--format") {
		t.Errorf("error = %q, want it to mention --format", err)
	}
	assertExitCode(t, err, ExitConfigError)
}
