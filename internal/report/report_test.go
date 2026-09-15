package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveAndLoadAll(t *testing.T) {
	dir := t.TempDir()

	older := &Report{
		RunID:           "older",
		Kind:            KindProvision,
		StartedAt:       time.Now().Add(-time.Hour),
		DurationSeconds: 12.5,
		Provision: &Provision{
			Hosts: []HostResult{
				{Name: "web-01", OK: 5, Changed: 2},
				{Name: "db-01", OK: 4, Changed: 1, Failed: 1},
			},
			ChangedTasks: []string{"Install base packages"},
		},
	}
	newer := &Report{
		RunID:           "newer",
		Kind:            KindLimit,
		StartedAt:       time.Now(),
		DurationSeconds: 4.25,
		Limit: &Limit{
			Cgroup:        "demo-oom",
			Limits:        Limits{MemoryMax: 128 << 20},
			OOMKills:      1,
			ThrottledUsec: 0,
			Verdict:       "oom-killed",
		},
	}

	if err := Save(dir, older); err != nil {
		t.Fatalf("Save(older) error = %v", err)
	}
	if err := Save(dir, newer); err != nil {
		t.Fatalf("Save(newer) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "older.json")); err != nil {
		t.Fatalf("expected reports/older.json: %v", err)
	}

	reports, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("LoadAll() returned %d reports, want 2", len(reports))
	}
	if reports[0].RunID != "newer" || reports[1].RunID != "older" {
		t.Errorf("LoadAll() order = [%s %s], want [newer older]",
			reports[0].RunID, reports[1].RunID)
	}

	got := reports[0]
	if got.Kind != KindLimit || got.Limit == nil || got.Limit.OOMKills != 1 {
		t.Errorf("newer report mangled: %+v", got)
	}
	if got.Limit.Limits.MemoryMax != 128<<20 {
		t.Errorf("newer memory limit = %d, want %d", got.Limit.Limits.MemoryMax, 128<<20)
	}
	got = reports[1]
	if got.Kind != KindProvision || got.Provision == nil || len(got.Provision.Hosts) != 2 {
		t.Errorf("older report mangled: %+v", got)
	}
	if got.DurationSeconds != 12.5 {
		t.Errorf("duration = %v, want 12.5", got.DurationSeconds)
	}
}

func TestLoadAllMissingDir(t *testing.T) {
	reports, err := LoadAll(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("LoadAll() error = %v, want nil for missing dir", err)
	}
	if reports != nil {
		t.Fatalf("LoadAll() = %v, want nil", reports)
	}
}

func TestLoadAllIgnoresNonJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(reports) != 0 {
		t.Fatalf("LoadAll() = %v, want empty", reports)
	}
}

func TestLoadAllInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAll(dir)
	if err == nil {
		t.Fatal("LoadAll() succeeded, want error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "bad.json") {
		t.Errorf("LoadAll() error = %q, want file name", err)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{in: 0, want: "0 B"},
		{in: 1023, want: "1023 B"},
		{in: 1024, want: "1.0 KiB"},
		{in: 268435456, want: "256.0 MiB"},
		{in: 1 << 30, want: "1.0 GiB"},
	}
	for _, tc := range tests {
		if got := FormatBytes(tc.in); got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
