package mageutil

import (
	"fmt"
	"os"
	"time"

	"github.com/openimsdk/gomake/internal/rlimit"
	"github.com/openimsdk/gomake/internal/util"
	"github.com/openimsdk/tools/utils/datautil"
)

const checkDelay = 3 * time.Second

func CheckAndReportServicesStatus() error {
	if err := LoadStartConfig(); err != nil {
		return err
	}
	err := CheckServicesRunning()
	if err != nil {
		return fmt.Errorf("some programs are not running properly: %w", err)
	}
	PrintGreen("All services are running normally.")
	PrintGreen(fmt.Sprintf("Waiting for %v to check listened ports...", checkDelay))
	time.Sleep(checkDelay)
	PrintBlue("Display details of the ports listened to by the service:")
	err = PrintListenedPortsByServices()
	if err != nil {
		return fmt.Errorf("PrintListenedPortsByServices error: %w", err)
	}
	return nil
}

func StopAndCheckServices() error {
	if err := LoadStartConfig(); err != nil {
		return err
	}
	PrintErr(KillExistingServices())
	err := ensureAllServicesStopped()
	if err != nil {
		return err
	}
	PrintGreen("All services have been stopped")
	return nil
}

func ensureAllServicesStopped() error {
	const maxAttempts = 15
	var err error
	for i := range maxAttempts {
		err = CheckServicesStopped()
		if err == nil {
			return nil
		}
		PrintYellow("Some services have not been stopped, details are as follows: " + err.Error())
		PrintYellow(fmt.Sprintf("Continue to wait for %v before checking again", checkDelay))
		if i < maxAttempts-1 {
			time.Sleep(checkDelay)
		}
	}
	return fmt.Errorf("already waited for %d seconds, some services have still not stopped", maxAttempts)
}

func StartToolsAndServices(tools []string, services []string, pathOpts *PathOptions) error {
	if pathOpts != nil {
		if err := UpdateGlobalPaths(pathOpts); err != nil {
			return fmt.Errorf("failed to update paths: %w", err)
		}
	}

	if err := LoadStartConfig(); err != nil {
		return err
	}
	if err := rlimit.SetMaxOpenFiles(startConfig.MaxFileDescriptors); err != nil {
		return err
	}

	var toolsToStart, servicesToStart []string
	if len(tools) == 0 && len(services) == 0 {
		toolsToStart = startConfig.Tools
		servicesToStart = datautil.Keys(startConfig.Services)
	} else {
		for _, tool := range tools {
			if util.IsExecutableFile(Paths.GetBinToolFullPath(util.BinaryWithRuntimeExtension(tool))) {
				toolsToStart = append(toolsToStart, tool)
			} else {
				PrintErr(fmt.Errorf("tool %s is not executable", tool))
			}
		}
		for _, service := range services {
			if util.IsExecutableFile(Paths.GetBinServiceFullPath(util.BinaryWithRuntimeExtension(service))) {
				servicesToStart = append(servicesToStart, service)
			} else {
				PrintErr(fmt.Errorf("service %s is not executable", service))
			}
		}
	}

	if len(toolsToStart) == 0 && len(servicesToStart) == 0 {
		PrintYellow("No valid services or tools found to start. Please build first.")
		return nil
	}

	PrintBlue(fmt.Sprintf("Services to start: %v", servicesToStart))
	PrintBlue(fmt.Sprintf("Tools to start: %v", toolsToStart))

	if len(toolsToStart) > 0 {
		PrintBlue("Starting tools primarily involves component verification and other preparatory tasks.")
		if err := StartTools(toolsToStart); err != nil {
			return fmt.Errorf("some tools failed to start, details are as follows, abort start: %w", err)
		}
		PrintGreen("All tools executed successfully")
	}

	PrintErr(KillExistingServices())
	err := ensureAllServicesStopped()
	if err != nil {
		return fmt.Errorf("some services running, details are as follows, abort start %w", err)
	}

	if len(servicesToStart) > 0 {
		err = StartServices(servicesToStart)
		if err != nil {
			return fmt.Errorf("failed to start all services %w", err)
		}
		return CheckAndReportServicesStatus()
	}

	return nil
}

func Build(services []string, tools []string, pathOpts *PathOptions, buildOpt *BuildOptions) error {
	resolvedBuildOpt := ResolveBuildOptions(buildOpt, &BuildOptions{
		CgoEnabled: util.GetEnvWithNoErr[string]("CGO_ENABLED"),
		Release:    util.GetEnvWithNoErr[bool]("RELEASE"),
		Compress:   util.GetEnvWithNoErr[bool]("COMPRESS"),
		Platforms:  util.GetEnvWithNoErr[[]string]("PLATFORMS"),
	})

	if pathOpts != nil {
		if err := UpdateGlobalPaths(pathOpts); err != nil {
			return fmt.Errorf("failed to update paths: %w", err)
		}
	}

	if _, err := os.Stat(StartConfigFile); err == nil {
		if err := LoadStartConfig(); err != nil {
			return err
		}
	}

	serviceTargets, toolTargets := resolveCompileTargets(services, tools)
	if cgoEnabled := resolvedBuildOpt.GetCgoEnabled(); cgoEnabled != "" {
		PrintBlue(fmt.Sprintf("CGO_ENABLED %s", cgoEnabled))
	}
	platforms := resolvedBuildOpt.GetPlatforms()
	if len(platforms) == 0 {
		platform, err := DetectPlatform()
		if err != nil {
			return err
		}
		platforms = []string{platform}
	}
	for _, platform := range platforms {
		if err := CompileForPlatform(resolvedBuildOpt, platform, serviceTargets, toolTargets); err != nil {
			return err
		}
	}
	if err := createStartConfigYML(serviceTargets, toolTargets); err != nil {
		return err
	}
	PrintGreen("All specified binaries under cmd and tools were successfully compiled.")
	return nil
}
