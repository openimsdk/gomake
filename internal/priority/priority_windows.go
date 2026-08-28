//go:build windows

package priority

import (
	"golang.org/x/sys/windows"
)

func Set(pid int, level Level) error {
	var class uint32
	switch level {
	case Low:
		class = windows.IDLE_PRIORITY_CLASS
	case BelowNormal:
		class = windows.BELOW_NORMAL_PRIORITY_CLASS
	case High:
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
