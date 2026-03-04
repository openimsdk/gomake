package mageutil

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/openimsdk/gomake/internal/util"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

func OsArch() string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	if os == "windows" {
		return fmt.Sprintf("%s\\%s", os, arch)
	}
	return fmt.Sprintf("%s/%s", os, arch)
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
	} else {
		return fmt.Errorf("%s expected %d processes, but %d running", processPath, expectedCount, runningCount)
	}
}

// FetchProcesses returns a map of executable paths to their running count.
func FetchProcesses() (map[string]int, error) {
	processMap := make(map[string]int)
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %v", err)
	}

	for _, p := range processes {
		exePath, err := p.Exe()
		if err != nil {
			continue // Skip processes where the executable path cannot be determined
		}
		exePath = util.NormalizeExePath(exePath)
		processMap[exePath]++
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
	pidMap := make(map[string][]int)
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %v", err)
	}

	for _, proc := range processes {
		exePath, err := proc.Exe()
		if err != nil {
			// Ignore processes where the executable path cannot be determined
			continue
		}

		exePath = util.NormalizeExePath(exePath)
		pidMap[exePath] = append(pidMap[exePath], int(proc.Pid))
	}

	return pidMap, nil
}
func PrintBinaryPorts(binaryPath string, pidMap map[string][]int) {
	pids, exists := pidMap[binaryPath]
	if !exists || len(pids) == 0 {
		fmt.Printf("No running processes found for binary: %s\n", binaryPath)
		return
	}

	for _, pid := range pids {
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			fmt.Printf("Failed to create process object for PID %d: %v\n", pid, err)
			continue
		}

		cmdline, err := proc.Cmdline()
		if err != nil {
			fmt.Printf("Failed to get command line for PID %d: %v\n", pid, err)
			continue
		}

		connections, err := net.ConnectionsPid("all", int32(pid))
		if err != nil {
			fmt.Printf("Error getting connections for PID %d: %v\n", pid, err)
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
			ports := make([]string, 0, len(portsMap))
			for port := range portsMap {
				ports = append(ports, port)
			}
			PrintGreen(fmt.Sprintf("Cmdline: %s, PID: %d is listening on ports: %s", cmdline, pid, strings.Join(ports, ", ")))
		}
	}
}

func BatchKillExistBinaries(binaryPaths []string) {
	processes, err := process.Processes()
	if err != nil {
		fmt.Printf("Failed to get processes: %v\n", err)
		return
	}

	exePathMap := make(map[string][]*process.Process)
	for _, p := range processes {
		exePath, err := p.Exe()
		if err != nil {
			continue // Skip processes where the executable path cannot be determined
		}
		exePath = util.NormalizeExePath(exePath)
		exePathMap[exePath] = append(exePathMap[exePath], p)
	}

	for _, binaryPath := range binaryPaths {
		if procs, found := exePathMap[binaryPath]; found {
			fmt.Println("binaryPath  found ", binaryPath)
			for _, p := range procs {
				terminateAndKillProcess(p)
			}
		}
	}
}

func terminateAndKillProcess(p *process.Process) {
	cmdline, err := p.Cmdline()
	if err != nil {
		fmt.Printf("Failed to get command line for process %d: %v\n", p.Pid, err)
		return
	}

	err = p.Terminate()
	if err != nil {
		err = p.Kill() // Fallback to kill if terminate fails
		if err != nil {
			fmt.Printf("Failed to kill process cmdline: %s, pid: %d, err: %v\n", cmdline, p.Pid, err)
		} else {
			fmt.Printf("Killed process cmdline: %s, pid: %d\n", cmdline, p.Pid)
		}
	} else {
		fmt.Printf("Terminated process cmdline: %s, pid: %d\n", cmdline, p.Pid)
	}
}

// KillExistBinary kills all processes matching the given binary file path.
func KillExistBinary(binaryPath string) {
	processes, err := process.Processes()
	if err != nil {
		fmt.Printf("Failed to get processes: %v\n", err)
		return
	}

	for _, p := range processes {
		exePath, err := p.Exe()
		if err != nil {
			continue
		}

		exePath = util.NormalizeExePath(exePath)
		if strings.Contains(exePath, binaryPath) {

			//if strings.EqualFold(exePath, binaryPath) {
			cmdline, err := p.Cmdline()
			if err != nil {
				fmt.Printf("Failed to get command line for process %d: %v\n", p.Pid, err)
				continue
			}

			err = p.Terminate()
			if err != nil {

				err = p.Kill()
				if err != nil {
					fmt.Printf("Failed to kill process cmdline: %s, pid: %d, err: %v\n", cmdline, p.Pid, err)
				} else {
					fmt.Printf("Killed process cmdline: %s, pid: %d\n", cmdline, p.Pid)
				}
			} else {
				fmt.Printf("Terminated process cmdline: %s, pid: %d\n", cmdline, p.Pid)
			}
		}
	}
}

// DetectPlatform detects the operating system and architecture.
func DetectPlatform() string {
	targetOS, targetArch := runtime.GOOS, runtime.GOARCH
	switch targetArch {
	case "amd64", "arm64":
	default:
		fmt.Printf("Unsupported architecture: %s\n", targetArch)
		os.Exit(1)
	}
	return fmt.Sprintf("%s_%s", targetOS, targetArch)
}

// rootDir gets the absolute path of the current directory.
func rootDir() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Failed to get current directory:", err)
		os.Exit(1)
	}
	return dir
}

var rootDirPath = rootDir()

// var platformsOutputBase = filepath.Join(rootDirPath, "_output/bin/platforms")
// var toolsOutputBase = filepath.Join(rootDirPath, "_output/bin/tools")
