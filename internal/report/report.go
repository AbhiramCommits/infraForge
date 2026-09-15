// Package report persists per-run summaries and renders them for display.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Run kinds written by the provision and limit commands.
const (
	// KindProvision marks records written by `infraforge provision`.
	KindProvision = "provision"
	// KindLimit marks records written by `infraforge limit`.
	KindLimit = "limit"
)

// HostResult summarizes one host's ansible-playbook run.
type HostResult struct {
	Name        string `json:"name"`
	OK          int    `json:"ok"`
	Changed     int    `json:"changed"`
	Unreachable int    `json:"unreachable"`
	Failed      int    `json:"failed"`
}

// Limits are the cgroup v2 limits applied to a workload.
type Limits struct {
	// CPUMax is the cpu.max value, e.g. "50000 100000".
	CPUMax string `json:"cpu_max,omitempty"`
	// MemoryMax is the memory.max value in bytes.
	MemoryMax int64 `json:"memory_max,omitempty"`
	// PIDsMax is the pids.max value.
	PIDsMax int64 `json:"pids_max,omitempty"`
}

// Provision holds the ansible-side results of a provision run.
type Provision struct {
	Hosts        []HostResult `json:"hosts"`
	ChangedTasks []string     `json:"changed_tasks"`
}

// Limit holds the cgroup-side results of a limit run.
type Limit struct {
	Cgroup string `json:"cgroup"`
	Limits Limits `json:"limits"`
	// ThrottledUsec is the cpu.stat throttled_usec counter.
	ThrottledUsec int64 `json:"throttled_usec"`
	// OOMKills is the memory.events oom_kill counter.
	OOMKills int64 `json:"oom_kills"`
	// HighEvents is the memory.events high counter.
	HighEvents int64 `json:"high_events"`
	// PeakMemory is the highest memory.current sampled during the run.
	PeakMemory int64  `json:"peak_memory"`
	Verdict    string `json:"verdict"`
}

// Report is a single run record written to reports/<run_id>.json.
type Report struct {
	RunID           string     `json:"run_id"`
	Kind            string     `json:"kind"`
	StartedAt       time.Time  `json:"started_at"`
	DurationSeconds float64    `json:"duration_seconds"`
	Provision       *Provision `json:"provision,omitempty"`
	Limit           *Limit     `json:"limit,omitempty"`
}

// Summary pairs the most recent provision and limit records for display.
type Summary struct {
	Provision *Report `json:"provision,omitempty"`
	Limit     *Report `json:"limit,omitempty"`
}

// Save writes the report to dir/<run_id>.json, creating dir if needed.
func Save(dir string, r *Report) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create reports dir %q: %w", dir, err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	path := filepath.Join(dir, r.RunID+".json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write report %q: %w", path, err)
	}
	return nil
}

// LoadAll reads every report in dir, newest first. A missing dir yields an
// empty list.
func LoadAll(dir string) ([]*Report, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read reports dir %q: %w", dir, err)
	}
	reports := make([]*Report, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read report %q: %w", e.Name(), err)
		}
		var r Report
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("parse report %q: %w", e.Name(), err)
		}
		reports = append(reports, &r)
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].StartedAt.After(reports[j].StartedAt)
	})
	return reports, nil
}

// FormatBytes renders a byte count in human units.
func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
