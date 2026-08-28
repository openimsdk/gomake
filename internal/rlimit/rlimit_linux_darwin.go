//go:build linux || darwin

package rlimit

import (
	"syscall"
)

func SetMaxOpenFiles(num int) error {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}
	rLimit.Max = uint64(num)
	rLimit.Cur = uint64(num)
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
}
