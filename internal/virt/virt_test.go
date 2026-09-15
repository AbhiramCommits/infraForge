package virt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestNewUnknownBackend(t *testing.T) {
	_, err := New("bogus")
	if err == nil {
		t.Fatal("New(\"bogus\") succeeded, want error")
	}
	if !strings.Contains(err.Error(), "unknown backend") {
		t.Errorf("error = %q, want unknown backend", err)
	}
}

// TestNewLibvirtStub only holds for default builds; under -tags libvirt the
// stub is replaced by the real backend.
func TestNewLibvirtStub(t *testing.T) {
	_, err := New("libvirt")
	if err == nil {
		t.Skip("libvirt backend compiled in")
	}
	if !strings.Contains(err.Error(), "libvirt") {
		t.Errorf("error = %q, want mention of libvirt", err)
	}
}

func TestReadNamespacesAt(t *testing.T) {
	root := t.TempDir()
	nsDir := filepath.Join(root, "1234", "ns")
	if err := os.MkdirAll(nsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, id := range map[string]string{
		"pid": "4026531836",
		"uts": "4026531838",
		"net": "4026531999",
	} {
		if err := os.Symlink(name+":["+id+"]", filepath.Join(nsDir, name)); err != nil {
			t.Fatal(err)
		}
	}

	got, err := readNamespacesAt(root, 1234)
	if err != nil {
		t.Fatalf("readNamespacesAt() error = %v", err)
	}
	want := map[string]string{
		"pid": "4026531836",
		"uts": "4026531838",
		"net": "4026531999",
	}
	if len(got) != len(want) {
		t.Fatalf("readNamespacesAt() = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("namespace %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestReadNamespacesAtMissing(t *testing.T) {
	if _, err := readNamespacesAt(t.TempDir(), 9999); err == nil {
		t.Fatal("readNamespacesAt() succeeded, want error for missing proc entry")
	}
}

func TestContainerName(t *testing.T) {
	if got := containerName("web-01"); got != "infraforge-vm-web-01" {
		t.Errorf("containerName() = %q, want infraforge-vm-web-01", got)
	}
}

func TestMapContainerState(t *testing.T) {
	tests := []struct {
		name  string
		state *container.State
		want  State
	}{
		{name: "nil", state: nil, want: StateUnknown},
		{name: "running", state: &container.State{Running: true}, want: StateRunning},
		{name: "paused", state: &container.State{Paused: true}, want: StatePaused},
		{name: "exited", state: &container.State{}, want: StateStopped},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapContainerState(tc.state); got != tc.want {
				t.Errorf("mapContainerState() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapContainerStatus(t *testing.T) {
	tests := []struct {
		status string
		want   State
	}{
		{status: "running", want: StateRunning},
		{status: "paused", want: StatePaused},
		{status: "exited", want: StateStopped},
		{status: "created", want: StateStopped},
		{status: "dead", want: StateStopped},
		{status: "restarting", want: StateUnknown},
	}
	for _, tc := range tests {
		if got := mapContainerStatus(tc.status); got != tc.want {
			t.Errorf("mapContainerStatus(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestSortedNamespaceNames(t *testing.T) {
	got := SortedNamespaceNames(map[string]string{"net": "1", "pid": "2", "uts": "3"})
	want := []string{"net", "pid", "uts"}
	if len(got) != len(want) {
		t.Fatalf("SortedNamespaceNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortedNamespaceNames() = %v, want %v", got, want)
		}
	}
}
