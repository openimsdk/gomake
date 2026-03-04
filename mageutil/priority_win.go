//go:build windows

package mageutil

import (
	"golang.org/x/sys/windows"
)

func SetPriority(pid int, level PriorityLevel) error {
	var class uint32
	switch level {
	case PriorityLow:
		class = windows.IDLE_PRIORITY_CLASS
	case PriorityBelowNormal:
		class = windows.BELOW_NORMAL_PRIORITY_CLASS
	case PriorityHigh:
		class = windows.HIGH_PRIORITY_CLASS
	default:
		class = windows.NORMAL_PRIORITY_CLASS
	}

	handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	return windows.SetPriorityClass(handle, class)
}
