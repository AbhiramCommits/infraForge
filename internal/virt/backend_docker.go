package virt

import (
	"context"
	"fmt"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// vmContainerPrefix keeps VM containers distinguishable from provisioned
// node containers.
const vmContainerPrefix = "infraforge-vm-"

func containerName(host string) string {
	return vmContainerPrefix + host
}

type dockerBackend struct {
	client *client.Client
}

func newDockerBackend() (*dockerBackend, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &dockerBackend{client: cli}, nil
}

// Start creates and starts a container with its own PID, UTS, and network
// namespaces (Docker's defaults) and returns the running VM with namespace
// IDs read from /proc.
func (b *dockerBackend) Start(ctx context.Context, vm VM) (*VM, error) {
	name := containerName(vm.Name)
	if existing, err := b.client.ContainerInspect(ctx, name); err == nil {
		if !existing.State.Running && !existing.State.Paused {
			if err := b.client.ContainerStart(ctx, existing.ID, container.StartOptions{}); err != nil {
				return nil, fmt.Errorf("start existing container %q: %w", name, err)
			}
		}
		return b.Inspect(ctx, vm.Name)
	} else if !cerrdefs.IsNotFound(err) {
		return nil, fmt.Errorf("inspect container %q: %w", name, err)
	}

	created, err := b.client.ContainerCreate(ctx,
		&container.Config{Image: vm.Image},
		&container.HostConfig{},
		nil, nil, name)
	if err != nil {
		return nil, fmt.Errorf("create container %q: %w", name, err)
	}
	if err := b.client.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("start container %q: %w", name, err)
	}
	return b.Inspect(ctx, vm.Name)
}

// Stop gracefully stops the VM's container.
func (b *dockerBackend) Stop(ctx context.Context, name string) error {
	cname := containerName(name)
	if err := b.client.ContainerStop(ctx, cname, container.StopOptions{}); err != nil {
		return fmt.Errorf("stop container %q: %w", cname, err)
	}
	return nil
}

// Inspect reports the container state and its namespace IDs.
func (b *dockerBackend) Inspect(ctx context.Context, name string) (*VM, error) {
	cname := containerName(name)
	raw, err := b.client.ContainerInspect(ctx, cname)
	if err != nil {
		return nil, fmt.Errorf("inspect container %q: %w", cname, err)
	}
	vm := &VM{
		Name:  name,
		Image: raw.Config.Image,
		State: mapContainerState(raw.State),
		PID:   raw.State.Pid,
	}
	if raw.State.Pid > 0 {
		if ns, err := readNamespaces(raw.State.Pid); err == nil {
			vm.Namespaces = ns
		}
	}
	return vm, nil
}

// List returns the VMs managed by this backend.
func (b *dockerBackend) List(ctx context.Context) ([]VM, error) {
	containers, err := b.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	vms := make([]VM, 0, len(containers))
	for _, c := range containers {
		if len(c.Names) == 0 || !strings.HasPrefix(c.Names[0], "/"+vmContainerPrefix) {
			continue
		}
		vms = append(vms, VM{
			Name:  strings.TrimPrefix(c.Names[0], "/"+vmContainerPrefix),
			Image: c.Image,
			State: mapContainerStatus(c.State),
		})
	}
	return vms, nil
}

func mapContainerState(state *container.State) State {
	if state == nil {
		return StateUnknown
	}
	switch {
	case state.Running:
		return StateRunning
	case state.Paused:
		return StatePaused
	default:
		return StateStopped
	}
}

func mapContainerStatus(status string) State {
	switch status {
	case "running":
		return StateRunning
	case "paused":
		return StatePaused
	case "exited", "dead", "created":
		return StateStopped
	default:
		return StateUnknown
	}
}
