package mageutil

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/openimsdk/gomake/internal/util"
	"github.com/openimsdk/tools/utils/datautil"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

func OsArch() string {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	if goos == "windows" {
		return fmt.Sprintf("%s\\%s", goos, arch)
	}
	return fmt.Sprintf("%s/%s", goos, arch)
}

// CheckProcessNames checks if the number of processes running that match the specified path equals the expected count.
func CheckProcessNames(processPath string, expectedCount int, processMap map[string]int) error {
	// Retrieve the count of running processes from the map
	runningCount, exists := processMap[processPath]
	if !exists {
		runningCount = 0 // No processes are running if the path isn't found in the map
	}

	if runningCount == expectedCount {
		return nil
	}

	return fmt.Errorf("%s expected %d processes, but %d running", processPath, expectedCount, runningCount)
}

// FetchProcesses returns a map of executable paths to their running count.
func FetchProcesses() (map[string]int, error) {
	processMap, err := util.ProcessCountByExePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %v", err)
	}

	return processMap, nil
}

func CheckProcessInMap(processMap map[string]int, processPath string) bool {
	if _, exists := processMap[processPath]; exists {
		return true
	}
	return false
}

// FindPIDsByBinaryPath returns a map of executable paths to slices of PIDs.
func FindPIDsByBinaryPath() (map[string][]int, error) {
	pidMap, err := util.PIDsByExePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %v", err)
	}

	return pidMap, nil
}

func PrintBinaryPorts(binaryPath string, pidMap map[string][]int) {
	pids, exists := pidMap[binaryPath]
	if !exists || len(pids) == 0 {
		PrintYellow(fmt.Sprintf("No running processes found for binary: %s", binaryPath))
		return
	}

	for _, pid := range pids {
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			PrintYellow(fmt.Sprintf("Failed to create process object for PID %d: %v", pid, err))
			continue
		}

		cmdline, err := proc.Cmdline()
		if err != nil {
			PrintYellow(fmt.Sprintf("Failed to get command line for PID %d: %v", pid, err))
			continue
		}

		connections, err := net.ConnectionsPid("all", int32(pid))
		if err != nil {
			PrintYellow(fmt.Sprintf("Error getting connections for PID %d: %v", pid, err))
			continue
		}

		portsMap := make(map[string]struct{})
		for _, conn := range connections {
			if conn.Status == "LISTEN" {
				port := fmt.Sprintf("%d", conn.Laddr.Port)
				portsMap[port] = struct{}{}
			}
		}

		if len(portsMap) == 0 {
			PrintGreen(fmt.Sprintf("Cmdline: %s, PID: %d is not listening on any ports.", cmdline, pid))
		} else {
			ports := datautil.Keys(portsMap)
			PrintGreen(fmt.Sprintf("Cmdline: %s, PID: %d is listening on ports: %s", cmdline, pid, strings.Join(ports, ", ")))
		}
	}
}

func BatchKillExistBinaries(binaryPaths []string) error {
	exePathMap, err := util.ProcessesByExePath()
	if err != nil {
		return fmt.Errorf("failed to get processes: %w", err)
	}

	for _, binaryPath := range binaryPaths {
		if procs, found := exePathMap[binaryPath]; found {
			PrintBlue(fmt.Sprintf("binaryPath found %s", binaryPath))
			for _, p := range procs {
				terminateAndKillProcess(p)
			}
		}
	}

	return nil
}

func terminateAndKillProcess(p *process.Process) {
	cmdline, err := p.Cmdline()
	if err != nil {
		PrintYellow(fmt.Sprintf("Failed to get command line for process %d: %v", p.Pid, err))
		return
	}

	err = p.Terminate()
	if err != nil {
		err = p.Kill() // Fallback to kill if terminate fails
		if err != nil {
			PrintErr(fmt.Errorf("failed to kill process cmdline: %s, pid: %d, err: %w", cmdline, p.Pid, err))
		} else {
			PrintYellow(fmt.Sprintf("Killed process cmdline: %s, pid: %d", cmdline, p.Pid))
		}
	} else {
		PrintGreen(fmt.Sprintf("Terminated process cmdline: %s, pid: %d", cmdline, p.Pid))
	}
}

// KillExistBinary kills all processes matching the given binary file path.
func KillExistBinary(binaryPath string) error {
	exePathMap, err := util.ProcessesByExePath()
	if err != nil {
		return fmt.Errorf("failed to get processes: %w", err)
	}

	for exePath, procs := range exePathMap {
		if strings.Contains(exePath, binaryPath) {
			for _, p := range procs {
				terminateAndKillProcess(p)
			}
		}
	}

	return nil
}

// DetectPlatform detects the operating system and architecture.
func DetectPlatform() (string, error) {
	targetOS, targetArch := runtime.GOOS, runtime.GOARCH
	switch targetArch {
	case "amd64", "arm64":
	default:
		err := fmt.Errorf("unsupported architecture: %s", targetArch)
		return "", err
	}
	return fmt.Sprintf("%s_%s", targetOS, targetArch), nil
}

// var platformsOutputBase = filepath.Join(rootDirPath, "_output/bin/platforms")
// var toolsOutputBase = filepath.Join(rootDirPath, "_output/bin/tools")
