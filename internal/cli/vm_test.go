package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVMListUnknownBackend(t *testing.T) {
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"vm", "list", "--backend", "bogus"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "unknown backend") {
		t.Errorf("error = %q, want unknown backend", err)
	}
}

func TestVMListLibvirtNotCompiledIn(t *testing.T) {
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"vm", "list", "--backend", "libvirt"})
	err := cmd.Execute()
	if err == nil {
		t.Skip("libvirt backend compiled in")
	}
	if !strings.Contains(err.Error(), "libvirt") {
		t.Errorf("error = %q, want mention of libvirt", err)
	}
}

func TestVMStartMissingConfig(t *testing.T) {
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"vm", "start", "web-01", "--config", filepath.Join(t.TempDir(), "missing.yaml")})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Errorf("error = %q, want load config", err)
	}
}

func TestVMStartHostNotFound(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "infraforge.yaml")
	cfg := "hosts:\n  - name: web-01\n    image: ubuntu:22.04\n"
	if err := os.WriteFile(configPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetArgs([]string{"vm", "start", "nope", "--config", configPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "not found in config") {
		t.Errorf("error = %q, want not found in config", err)
	}
}

func TestVMHelp(t *testing.T) {
	var out strings.Builder
	opts := &Options{Stderr: io.Discard}
	cmd := newRootCommand(opts)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"vm", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, sub := range []string{"start", "stop", "inspect", "list"} {
		if !strings.Contains(out.String(), sub) {
			t.Errorf("vm help missing %q:\n%s", sub, out.String())
		}
	}
}
