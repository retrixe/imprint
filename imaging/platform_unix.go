//go:build !windows

package imaging

import "syscall"

type UnixPlatform interface {
	Platform
	SyscallUnmount(target string, flags int) error
}

var UnixSystemPlatform UnixPlatform = systemPlatform{}

func (p systemPlatform) SyscallUnmount(target string, flags int) error {
	return syscall.Unmount(target, flags)
}
