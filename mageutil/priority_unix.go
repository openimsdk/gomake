//go:build !windows

package mageutil

import (
	"syscall"
)

func SetPriority(pid int, level PriorityLevel) error {
	var nice int
	switch level {
	case PriorityLow:
		nice = 19
	case PriorityBelowNormal:
		nice = 10
	case PriorityHigh:
		nice = -10
	default:
		nice = 0
	}
	return syscall.Setpriority(syscall.PRIO_PROCESS, pid, nice)
}
