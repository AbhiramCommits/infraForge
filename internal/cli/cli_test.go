package cli

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestSubcommandsRegistered(t *testing.T) {
	var out bytes.Buffer
	cmd := newRootCommand(&Options{})
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, sub := range []string{"provision", "limit", "vm", "report"} {
		if !strings.Contains(out.String(), sub) {
			t.Errorf("help output missing %q:\n%s", sub, out.String())
		}
	}
}

func TestLogsCarryRunID(t *testing.T) {
	tests := []struct {
		name   string
		format string
		check  func(*testing.T, string)
	}{
		{
			name:   "text",
			format: "text",
			check: func(t *testing.T, out string) {
				if !strings.Contains(out, "run_id=") {
					t.Errorf("text log missing run_id=: %q", out)
				}
			},
		},
		{
			name:   "json",
			format: "json",
			check: func(t *testing.T, out string) {
				if !strings.Contains(out, `"run_id":"`) {
					t.Fatalf("json log missing run_id: %q", out)
				}
				for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
					var m map[string]any
					if err := json.Unmarshal([]byte(line), &m); err != nil {
						t.Errorf("log line is not valid JSON %q: %v", line, err)
					}
					if _, ok := m["run_id"]; !ok {
						t.Errorf("log line missing run_id field: %q", line)
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := &Options{Stderr: &buf}
			cmd := newRootCommand(opts)
			cmd.SetArgs([]string{"report", "--dry-run", "--log-format", tc.format})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if buf.Len() == 0 {
				t.Fatal("no log output produced")
			}
			tc.check(t, buf.String())
		})
	}
}

func TestInvalidLogFormat(t *testing.T) {
	opts := &Options{Stderr: &bytes.Buffer{}}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"report", "--log-format", "yaml"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want error for invalid --log-format")
	}
	if !strings.Contains(err.Error(), "--log-format") {
		t.Errorf("error = %q, want it to mention --log-format", err)
	}
}

func TestUnknownCommand(t *testing.T) {
	opts := &Options{Stderr: &bytes.Buffer{}}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"frobnicate"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error = %q, want it to mention unknown command", err)
	}
}

func TestNewRunID(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	first, err := newRunID()
	if err != nil {
		t.Fatalf("newRunID() error = %v", err)
	}
	if !re.MatchString(first) {
		t.Errorf("newRunID() = %q, want UUID v4 format", first)
	}
	second, err := newRunID()
	if err != nil {
		t.Fatalf("newRunID() error = %v", err)
	}
	if first == second {
		t.Errorf("two invocations produced the same run id %q", first)
	}
}
