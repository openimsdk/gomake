//go:build freebsd || dragonfly

package rlimit

import "syscall"

func SetMaxOpenFiles(num int) error {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}
	rLimit.Max = int64(num)
	rLimit.Cur = int64(num)
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
}
