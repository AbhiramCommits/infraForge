// Package virt manages virtual machines through pluggable backends.
package virt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// State is a VM lifecycle state.
type State string

const (
	// StateRunning means the VM is executing.
	StateRunning State = "running"
	// StateStopped means the VM is shut off.
	StateStopped State = "stopped"
	// StatePaused means the VM is suspended.
	StatePaused State = "paused"
	// StateUnknown means the state could not be determined.
	StateUnknown State = "unknown"
)

// VM describes a virtual machine managed by a backend.
type VM struct {
	Name string
	// Image is the container image or disk backing the VM.
	Image string
	State State
	// PID is the host PID of the VM's main process; 0 when unavailable.
	PID int
	// Namespaces maps namespace kinds (pid, uts, net, ...) to their IDs,
	// read from /proc/<pid>/ns/*.
	Namespaces map[string]string
}

// Backend manages VMs.
type Backend interface {
	Start(ctx context.Context, vm VM) (*VM, error)
	Stop(ctx context.Context, name string) error
	Inspect(ctx context.Context, name string) (*VM, error)
	List(ctx context.Context) ([]VM, error)
}

// New selects a backend by name ("docker" or "libvirt"). The empty name
// auto-detects: docker when its daemon is reachable, otherwise libvirt when
// it is compiled in.
func New(name string) (Backend, error) {
	switch name {
	case "":
		return autoSelect()
	case "docker":
		return newDockerBackend()
	case "libvirt":
		return newLibvirtBackend()
	default:
		return nil, fmt.Errorf("unknown backend %q (available: docker, libvirt)", name)
	}
}

func autoSelect() (Backend, error) {
	db, dockerErr := dockerBackendAvailable()
	if dockerErr == nil {
		return db, nil
	}
	lb, libvirtErr := newLibvirtBackend()
	if libvirtErr == nil {
		return lb, nil
	}
	return nil, fmt.Errorf("no VM backend available: docker: %v; libvirt: %v", dockerErr, libvirtErr)
}

// dockerBackendAvailable returns a docker backend whose daemon answers a
// ping within a short timeout.
func dockerBackendAvailable() (Backend, error) {
	db, err := newDockerBackend()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := db.client.Ping(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

// readNamespaces returns the namespace IDs of pid, reading /proc.
func readNamespaces(pid int) (map[string]string, error) {
	return readNamespacesAt("/proc", pid)
}

// readNamespacesAt reads the namespace symlinks under procRoot/<pid>/ns/.
// Each entry looks like /proc/<pid>/ns/pid -> pid:[4026531836].
func readNamespacesAt(procRoot string, pid int) (map[string]string, error) {
	nsDir := filepath.Join(procRoot, strconv.Itoa(pid), "ns")
	entries, err := os.ReadDir(nsDir)
	if err != nil {
		return nil, fmt.Errorf("read namespace dir %q: %w", nsDir, err)
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		target, err := os.Readlink(filepath.Join(nsDir, e.Name()))
		if err != nil {
			continue
		}
		target = strings.TrimPrefix(target, e.Name()+":[")
		target = strings.TrimSuffix(target, "]")
		out[e.Name()] = target
	}
	return out, nil
}

// SortedNamespaceNames returns namespace kinds in stable order for display.
func SortedNamespaceNames(ns map[string]string) []string {
	names := make([]string, 0, len(ns))
	for k := range ns {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
