//go:build !libvirt

package virt

import "errors"

// newLibvirtBackend is a stub on builds without the libvirt tag. The real
// implementation links libvirt via cgo and lives in backend_libvirt.go.
func newLibvirtBackend() (Backend, error) {
	return nil, errors.New("libvirt backend not compiled in (build with -tags libvirt)")
}
