package ansible

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func fixture(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "testdata", "ansible", name))
	if err != nil {
		t.Fatalf("open fixture %q: %v", name, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name          string
		file          string
		wantHosts     map[string]HostStats
		wantChanged   []string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "all ok",
			file: "ok.json",
			wantHosts: map[string]HostStats{
				"web-01": {OK: 4, Changed: 0, Unreachable: 0, Failed: 0},
			},
		},
		{
			name: "changed tasks collected and deduped",
			file: "changed.json",
			wantHosts: map[string]HostStats{
				"web-01": {OK: 5, Changed: 3, Unreachable: 0, Failed: 0},
				"db-01":  {OK: 5, Changed: 1, Unreachable: 0, Failed: 0},
			},
			wantChanged: []string{
				"Install base packages",
				"Deploy the CPU and memory burner script",
				"Start the burner via supervisor",
			},
		},
		{
			name: "failed host",
			file: "failed.json",
			wantHosts: map[string]HostStats{
				"web-01": {OK: 1, Changed: 0, Unreachable: 0, Failed: 1},
			},
		},
		{
			name: "unreachable host",
			file: "unreachable.json",
			wantHosts: map[string]HostStats{
				"db-01": {OK: 0, Changed: 0, Unreachable: 1, Failed: 0},
			},
		},
		{
			name: "multiple documents, last stats win",
			file: "multi-doc.json",
			wantHosts: map[string]HostStats{
				"web-01": {OK: 4, Changed: 2, Unreachable: 0, Failed: 0},
				"db-01":  {OK: 4, Changed: 1, Unreachable: 0, Failed: 0},
			},
			wantChanged: []string{
				"Ensure container web-01 is running",
				"Install base packages",
			},
		},
		{
			name:      "empty document",
			file:      "empty.json",
			wantHosts: map[string]HostStats{},
		},
		{
			name:          "malformed json",
			file:          "malformed.json",
			wantErr:       true,
			wantErrSubstr: "decode ansible output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ParseJSON(fixture(t, tc.file), discardLogger())
			if tc.wantErr {
				if err == nil {
					t.Fatal("ParseJSON() succeeded, want error")
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("ParseJSON() error = %q, want it to contain %q", err, tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseJSON() unexpected error: %v", err)
			}
			if len(res.Hosts) != len(tc.wantHosts) {
				t.Fatalf("got %d hosts, want %d: %+v", len(res.Hosts), len(tc.wantHosts), res.Hosts)
			}
			for host, want := range tc.wantHosts {
				got, ok := res.Hosts[host]
				if !ok {
					t.Errorf("missing host %q in result", host)
					continue
				}
				if got != want {
					t.Errorf("host %q stats = %+v, want %+v", host, got, want)
				}
			}
			if len(res.ChangedTasks) != len(tc.wantChanged) {
				t.Fatalf("ChangedTasks = %v, want %v", res.ChangedTasks, tc.wantChanged)
			}
			for i := range tc.wantChanged {
				if res.ChangedTasks[i] != tc.wantChanged[i] {
					t.Errorf("ChangedTasks[%d] = %q, want %q", i, res.ChangedTasks[i], tc.wantChanged[i])
				}
			}
		})
	}
}

func TestParseJSONLogsTaskTransitions(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	if _, err := ParseJSON(fixture(t, "changed.json"), logger); err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "task finished") {
		t.Errorf("log output missing task transitions: %q", out)
	}
	if !strings.Contains(out, "Install base packages") {
		t.Errorf("log output missing task name: %q", out)
	}
	if !strings.Contains(out, "changed_hosts=1") {
		t.Errorf("log output missing changed_hosts count: %q", out)
	}
}

func TestResultValidate(t *testing.T) {
	tests := []struct {
		name    string
		hosts   map[string]HostStats
		wantErr string
	}{
		{
			name:  "clean result",
			hosts: map[string]HostStats{"web-01": {OK: 4}},
		},
		{
			name:    "one failure",
			hosts:   map[string]HostStats{"web-01": {Failed: 1}, "db-01": {OK: 2}},
			wantErr: "provisioning failed: 1 host(s) with failures, 0 host(s) unreachable",
		},
		{
			name:    "failure and unreachable",
			hosts:   map[string]HostStats{"web-01": {Failed: 1}, "db-01": {Unreachable: 1}},
			wantErr: "provisioning failed: 1 host(s) with failures, 1 host(s) unreachable",
		},
		{
			name:    "only unreachable",
			hosts:   map[string]HostStats{"db-01": {Unreachable: 1}},
			wantErr: "provisioning failed: 0 host(s) with failures, 1 host(s) unreachable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := &Result{Hosts: tc.hosts}
			err := res.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() = nil, want error")
			}
			if err.Error() != tc.wantErr {
				t.Errorf("Validate() error = %q, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestRunnerBuildArgs(t *testing.T) {
	r := &Runner{Playbook: "deploy/playbooks/site.yml", Inventory: "deploy/inventory.yml"}
	want := []string{"-i", "deploy/inventory.yml", "deploy/playbooks/site.yml"}
	got := r.buildArgs()
	if len(got) != len(want) {
		t.Fatalf("buildArgs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildArgs() = %v, want %v", got, want)
		}
	}

	r.DryRun = true
	want = []string{"-i", "deploy/inventory.yml", "--check", "--diff", "deploy/playbooks/site.yml"}
	got = r.buildArgs()
	if len(got) != len(want) {
		t.Fatalf("buildArgs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildArgs() = %v, want %v", got, want)
		}
	}
}
