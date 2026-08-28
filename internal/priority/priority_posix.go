//go:build !windows

package priority

import (
	"syscall"
)

func Set(pid int, level Level) error {
	var nice int
	switch level {
	case Low:
		nice = 19
	case BelowNormal:
		nice = 10
	case High:
		nice = -10
	default:
		nice = 0
	}
	return syscall.Setpriority(syscall.PRIO_PROCESS, pid, nice)
}
