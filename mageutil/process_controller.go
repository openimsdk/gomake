package mageutil

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/openimsdk/gomake/internal/util"
)

// StopServices terminates all configured service processes.
func StopServices() error {
	var errs []error
	for service := range startConfig.Services {
		fullPath := Paths.GetBinServiceFullPath(util.BinaryWithExtension(service))
		errs = append(errs, KillExistBinary(fullPath))
	}

	return errors.Join(errs...)
}

// StartServices starts all configured services or the specified ones.
func StartServices(services []string) error {
	if len(services) == 0 {
		return errors.New("must specify at least one service")
	}

	servicesToStart := make(map[string]int)
	for _, service := range services {
		count, found := startConfig.Services[service]
		if !found {
			PrintYellow(fmt.Sprintf("Service %s not found in config, but will try to start", service))
			count = 1
		}
		servicesToStart[service] = count
	}

	for service, count := range servicesToStart {
		binaryPath := Paths.GetBinServiceFullPath(util.BinaryWithExtension(service))

		if _, err := os.Stat(binaryPath); err != nil {
			PrintErr(fmt.Errorf("service executable not found: %s. Please build first", binaryPath))
			continue
		}

		for i := range count {
			configPath := Paths.Config
			if os.Getenv(DeploymentType) == KUBERNETES {
				configPath = Paths.K8sConfig
			}
			args := []string{"-i", strconv.Itoa(i), "-c", configPath}
			cmd := NewCmd(binaryPath).
				WithArgs(args...).
				WithDir(Paths.OutputHostBinService).
				WithStdout(GetSharedLogFileWithoutError()).
				WithStderr(GetSharedLogFileWithoutError())
			PrintBlue(fmt.Sprintf("Starting %s", cmd.String()))
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start service %s with args %v: %v", binaryPath, args, err)
			}
		}
	}
	return nil
}

// StartTools starts all tool binaries or specified ones.
func StartTools(tools []string) error {
	if len(tools) == 0 {
		return errors.New("must specify at least one tool")
	}

	toolsToStart := make([]string, 0)
	for _, tool := range tools {
		found := slices.Contains(startConfig.Tools, tool)
		if !found {
			PrintYellow(fmt.Sprintf("Tool %s not found in config, but will try to start", tool))
		}
		toolsToStart = append(toolsToStart, tool)
	}

	for _, tool := range toolsToStart {
		toolFullPath := Paths.GetBinToolFullPath(util.BinaryWithExtension(tool))

		if _, err := os.Stat(toolFullPath); err != nil {
			PrintErr(fmt.Errorf("tool not found: %s. please build first", toolFullPath))
			continue
		}

		configPath := Paths.Config
		if os.Getenv(DeploymentType) == KUBERNETES {
			configPath = Paths.K8sConfig
		}

		cmd := NewCmd(toolFullPath).
			WithArgs("-c", configPath).
			WithDir(Paths.OutputHostBinTool).
			WithStdout(GetSharedLogFileWithoutError()).
			WithStderr(GetSharedLogFileWithoutError())
		PrintBlue(fmt.Sprintf("Starting %s", cmd.String()))

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run %s with error: %v", toolFullPath, err)
		}

		PrintGreen(fmt.Sprintf("Starting %s successfully", cmd.String()))
	}
	return nil
}

// KillExistingServices kills all configured service processes.
func KillExistingServices() error {
	var paths []string
	for service := range startConfig.Services {
		fullPath := Paths.GetBinServiceFullPath(util.BinaryWithExtension(service))
		paths = append(paths, fullPath)
	}
	return BatchKillExistBinaries(paths)
}

// CheckServicesStopped returns an error if any configured services are still running.
func CheckServicesStopped() error {
	var runningServices []string

	ps, err := FetchProcesses()
	if err != nil {
		return err
	}

	for service := range startConfig.Services {
		fullPath := Paths.GetBinServiceFullPath(util.BinaryWithExtension(service))
		if CheckProcessInMap(ps, fullPath) {
			runningServices = append(runningServices, service)
		}
	}

	if len(runningServices) > 0 {
		return fmt.Errorf("the following services are still running: %s", strings.Join(runningServices, ", "))
	}

	return nil
}

// CheckServicesRunning checks whether all configured services are running as expected.
func CheckServicesRunning() error {
	var errorMessages []string

	ps, err := FetchProcesses()
	if err != nil {
		return err
	}

	for service, expectedCount := range startConfig.Services {
		fullPath := Paths.GetBinServiceFullPath(util.BinaryWithExtension(service))
		err := CheckProcessNames(fullPath, expectedCount, ps)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("service %s is not running as expected: %v", service, err))
		}
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("%s", strings.Join(errorMessages, "\n"))
	}

	return nil
}

// PrintListenedPortsByServices prints the ports listened to by configured services.
func PrintListenedPortsByServices() error {
	ps, err := FindPIDsByBinaryPath()
	if err != nil {
		return err
	}
	for service := range startConfig.Services {
		fullPath := Paths.GetBinServiceFullPath(util.BinaryWithExtension(service))
		PrintBinaryPorts(fullPath, ps)
	}
	return nil
}
