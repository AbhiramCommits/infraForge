//go:build libvirt

package virt

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"libvirt.org/go/libvirt"
)

type libvirtBackend struct {
	conn *libvirt.Connect
}

func newLibvirtBackend() (Backend, error) {
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, fmt.Errorf("connect to libvirt: %w", err)
	}
	return &libvirtBackend{conn: conn}, nil
}

// Start looks the domain up by name and boots it if it is shut off.
func (b *libvirtBackend) Start(ctx context.Context, vm VM) (*VM, error) {
	dom, err := b.conn.LookupDomainByName(vm.Name)
	if err != nil {
		return nil, fmt.Errorf("lookup domain %q: %w", vm.Name, err)
	}
	defer func() { _ = dom.Free() }()
	if err := dom.Create(); err != nil {
		// "already active" is fine; anything else is a real failure.
		if le, ok := err.(libvirt.Error); !ok || le.Code != libvirt.ERR_OPERATION_INVALID {
			return nil, fmt.Errorf("create domain %q: %w", vm.Name, err)
		}
	}
	return b.inspect(ctx, dom, vm.Name)
}

// Stop sends a graceful shutdown to the domain.
func (b *libvirtBackend) Stop(ctx context.Context, name string) error {
	dom, err := b.conn.LookupDomainByName(name)
	if err != nil {
		return fmt.Errorf("lookup domain %q: %w", name, err)
	}
	defer func() { _ = dom.Free() }()
	if err := dom.ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_DEFAULT); err != nil {
		return fmt.Errorf("shutdown domain %q: %w", name, err)
	}
	return nil
}

// Inspect reports domain state and namespace IDs.
func (b *libvirtBackend) Inspect(ctx context.Context, name string) (*VM, error) {
	dom, err := b.conn.LookupDomainByName(name)
	if err != nil {
		return nil, fmt.Errorf("lookup domain %q: %w", name, err)
	}
	defer func() { _ = dom.Free() }()
	return b.inspect(ctx, dom, name)
}

// List returns all domains, active and inactive.
func (b *libvirtBackend) List(ctx context.Context) ([]VM, error) {
	flags := libvirt.ConnectListAllDomainsFlags(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	domains, err := b.conn.ListAllDomains(flags)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	vms := make([]VM, 0, len(domains))
	for _, dom := range domains {
		name, err := dom.GetName()
		if err != nil {
			_ = dom.Free()
			continue
		}
		vm := &VM{Name: name}
		if info, err := dom.GetInfo(); err == nil {
			vm.State = mapDomainState(info.State)
		}
		vms = append(vms, *vm)
		_ = dom.Free()
	}
	return vms, nil
}

func (b *libvirtBackend) inspect(ctx context.Context, dom *libvirt.Domain, name string) (*VM, error) {
	info, err := dom.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("get domain info %q: %w", name, err)
	}
	vm := &VM{
		Name:  name,
		State: mapDomainState(info.State),
	}
	if pid, err := domainPID(name); err == nil {
		vm.PID = pid
		if ns, err := readNamespaces(pid); err == nil {
			vm.Namespaces = ns
		}
	}
	return vm, nil
}

func mapDomainState(state libvirt.DomainState) State {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return StateRunning
	case libvirt.DOMAIN_PAUSED:
		return StatePaused
	case libvirt.DOMAIN_SHUTOFF:
		return StateStopped
	default:
		return StateUnknown
	}
}

// domainPID reads the QEMU process pid for name from libvirt's run directory.
func domainPID(name string) (int, error) {
	data, err := os.ReadFile("/var/run/libvirt/qemu/" + name + ".pid")
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return pid, nil
}
