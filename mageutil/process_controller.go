package mageutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// StopBinaries iterates over all binary files and terminates their corresponding processes.
func StopBinaries() {
	for binary := range serviceBinaries {
		fullPath := GetBinFullPath(binary)
		KillExistBinary(fullPath)
	}
}

// StartBinaries Start all binary services or specified ones.
func StartBinaries(specificBinaries ...string) error {
	var binariesToStart map[string]int
	if len(specificBinaries) > 0 {
		binariesToStart = make(map[string]int)
		for _, binary := range specificBinaries {
			if count, exists := serviceBinaries[binary]; exists {
				binariesToStart[binary] = count
			} else {
				binariesToStart[binary] = 1
				// PrintYellow(fmt.Sprintf("Binary %s not found in config, starting with default count 1", binary))
			}
		}
	} else {
		binariesToStart = serviceBinaries
	}

	for binary, count := range binariesToStart {
		binFullPath := filepath.Join(Paths.OutputHostBin, binary)

		if _, err := os.Stat(binFullPath); err != nil {
			PrintRed(fmt.Sprintf("Binary not found: %s. Please build first.", binFullPath))
			continue
		}

		for i := 0; i < count; i++ {
			configPath := Paths.Config
			if os.Getenv(DeploymentType) == KUBERNETES {
				configPath = Paths.K8sConfig
			}
			args := []string{"-i", strconv.Itoa(i), "-c", configPath}
			cmd := exec.Command(binFullPath, args...)
			fmt.Printf("Starting %s\n", cmd.String())
			cmd.Dir = Paths.OutputHostBin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start %s with args %v: %v", binFullPath, args, err)
			}
		}
	}
	return nil
}

// StartTools starts all tool binaries or specified ones.
func StartTools(specificTools ...string) error {
	var toolsToStart []string
	if len(specificTools) > 0 {
		for _, tool := range specificTools {
			found := slices.Contains(toolBinaries, tool)
			if !found {
				PrintYellow(fmt.Sprintf("Tool %s not found in config, but will try to start", tool))
			}
			toolsToStart = append(toolsToStart, tool)
		}
	} else {
		toolsToStart = toolBinaries
	}

	for _, tool := range toolsToStart {
		toolFullPath := GetBinToolsFullPath(tool)

		if _, err := os.Stat(toolFullPath); err != nil {
			PrintRed(fmt.Sprintf("Tool not found: %s. Please build first.", toolFullPath))
			continue
		}

		configPath := Paths.Config
		if os.Getenv(DeploymentType) == KUBERNETES {
			configPath = Paths.K8sConfig
		}

		cmd := exec.Command(toolFullPath, "-c", configPath)
		fmt.Printf("Starting %s\n", cmd.String())
		cmd.Dir = Paths.OutputHostBinTools
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start %s with error: %v", toolFullPath, err)
		}

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("failed to execute %s with exit code: %v", toolFullPath, err)
		}
		fmt.Printf("Starting %s successfully \n", cmd.String())
	}
	return nil
}

// KillExistBinaries iterates over all binary files and kills their corresponding processes.
func KillExistBinaries() {
	var paths []string
	for binary := range serviceBinaries {
		fullPath := GetBinFullPath(binary)
		paths = append(paths, fullPath)
	}
	BatchKillExistBinaries(paths)
}

// CheckBinariesStop checks if all binary files have stopped and returns an error if there are any binaries still running.
func CheckBinariesStop() error {
	var runningBinaries []string

	ps, err := FetchProcesses()
	if err != nil {
		return err
	}

	for binary := range serviceBinaries {
		fullPath := GetBinFullPath(binary)
		if CheckProcessInMap(ps, fullPath) {
			runningBinaries = append(runningBinaries, binary)
		}
	}

	if len(runningBinaries) > 0 {
		return fmt.Errorf("the following binaries are still running: %s", strings.Join(runningBinaries, ", "))
	}

	return nil
}

// CheckBinariesRunning checks if all binary files are running as expected and returns any errors encountered.
func CheckBinariesRunning() error {
	var errorMessages []string

	ps, err := FetchProcesses()
	if err != nil {
		return err
	}

	for binary, expectedCount := range serviceBinaries {
		fullPath := GetBinFullPath(binary)
		err := CheckProcessNames(fullPath, expectedCount, ps)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("binary %s is not running as expected: %v", binary, err))
		}
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("%s", strings.Join(errorMessages, "\n"))
	}

	return nil
}

// PrintListenedPortsByBinaries iterates over all binary files and prints the ports they are listening on.
func PrintListenedPortsByBinaries() error {
	ps, err := FindPIDsByBinaryPath()
	if err != nil {
		return err
	}
	for binary := range serviceBinaries {
		basePath := GetBinFullPath(binary)
		fullPath := basePath
		PrintBinaryPorts(fullPath, ps)
	}
	return nil
}
