package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infraforge/infraforge/internal/report"
)

const provisionTestConfig = `hosts:
  - name: web-01
    image: ubuntu:22.04
    ssh_port: 2222
    limits:
      cpu_max: 50
      memory_max: 256M
      pids_max: 128
`

// fakeAnsiblePlaybook installs a stub ansible-playbook in PATH that replays
// a recorded JSON callback document, so these tests need no Ansible install.
func fakeAnsiblePlaybook(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	script := `#!/bin/sh
cat "$INFRAFORGE_TEST_ANSIBLE_OUTPUT"
exit "${INFRAFORGE_TEST_ANSIBLE_EXIT:-0}"
`
	if err := os.WriteFile(filepath.Join(binDir, "ansible-playbook"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake ansible-playbook: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestProvisionEndToEnd(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		exitCode string
		wantErr  string
	}{
		{name: "clean run", fixture: "ok.json", exitCode: "0"},
		{name: "unreachable host", fixture: "unreachable.json", exitCode: "0", wantErr: "unreachable"},
		{name: "playbook exits non-zero", fixture: "ok.json", exitCode: "2", wantErr: "ansible-playbook failed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeAnsiblePlaybook(t)
			t.Setenv("INFRAFORGE_TEST_ANSIBLE_OUTPUT",
				filepath.Join("..", "..", "testdata", "ansible", tc.fixture))
			t.Setenv("INFRAFORGE_TEST_ANSIBLE_EXIT", tc.exitCode)

			dir := t.TempDir()
			configPath := filepath.Join(dir, "infraforge.yaml")
			if err := os.WriteFile(configPath, []byte(provisionTestConfig), 0o644); err != nil {
				t.Fatal(err)
			}
			inventoryPath := filepath.Join(dir, "deploy", "inventory.yml")
			reportsDir := filepath.Join(dir, "reports")

			opts := &Options{Stderr: io.Discard}
			cmd := newRootCommand(opts)
			cmd.SetArgs([]string{"provision", "--config", configPath, "--inventory", inventoryPath, "--reports-dir", reportsDir})
			err := cmd.Execute()

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Execute() error = %v, want success", err)
				}
			} else {
				if err == nil {
					t.Fatalf("Execute() succeeded, want error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Execute() error = %q, want it to contain %q", err, tc.wantErr)
				}
			}

			data, err := os.ReadFile(inventoryPath)
			if err != nil {
				t.Fatalf("inventory was not generated: %v", err)
			}
			if !strings.Contains(string(data), "web-01") || !strings.Contains(string(data), "2222") {
				t.Errorf("generated inventory missing expected content:\n%s", data)
			}

			if tc.wantErr == "" || tc.wantErr == "unreachable" {
				reports, err := report.LoadAll(reportsDir)
				if err != nil {
					t.Fatalf("load reports: %v", err)
				}
				if len(reports) != 1 {
					t.Fatalf("got %d reports, want 1", len(reports))
				}
				if reports[0].Kind != report.KindProvision {
					t.Errorf("report kind = %q, want provision", reports[0].Kind)
				}
				if len(reports[0].Provision.Hosts) == 0 {
					t.Error("report has no hosts")
				}
			}
		})
	}
}

func TestProvisionErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{name: "missing config file", wantErr: "load config"},
		{name: "no hosts", yaml: "hosts: []\n", wantErr: "no hosts"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var configPath string
			if tc.yaml != "" {
				configPath = filepath.Join(dir, "infraforge.yaml")
				if err := os.WriteFile(configPath, []byte(tc.yaml), 0o644); err != nil {
					t.Fatal(err)
				}
			} else {
				configPath = filepath.Join(dir, "missing.yaml")
			}

			opts := &Options{Stderr: io.Discard}
			cmd := newRootCommand(opts)
			cmd.SetArgs([]string{"provision", "--config", configPath, "--inventory", filepath.Join(dir, "inventory.yml")})
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("Execute() succeeded, want error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Execute() error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
