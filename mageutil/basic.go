package mageutil

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/openimsdk/gomake/internal/util"
)

func CheckAndReportBinariesStatus() {
	InitForSSC()
	err := CheckBinariesRunning()
	if err != nil {
		PrintRed("Some programs are not running properly:")
		PrintRedNoTimeStamp(err.Error())
		os.Exit(1)
	}
	PrintGreen("All services are running normally.")
	PrintBlue("Display details of the ports listened to by the service:")
	time.Sleep(1 * time.Second)
	err = PrintListenedPortsByBinaries()
	if err != nil {
		PrintRed("PrintListenedPortsByBinaries error")
		PrintRedNoTimeStamp(err.Error())
		os.Exit(1)
	}
}

func StopAndCheckBinaries() {
	InitForSSC()
	KillExistBinaries()
	err := attemptCheckBinaries()
	if err != nil {
		PrintRed(err.Error())
		return
	}
	PrintGreen("All services have been stopped")
}

func attemptCheckBinaries() error {
	const maxAttempts = 15
	var err error
	for i := 0; i < maxAttempts; i++ {
		err = CheckBinariesStop()
		if err == nil {
			return nil
		}
		PrintYellow("Some services have not been stopped, details are as follows: " + err.Error())
		PrintYellow("Continue to wait for 1 second before checking again")
		if i < maxAttempts-1 {
			time.Sleep(1 * time.Second)
		}
	}
	return fmt.Errorf("already waited for %d seconds, some services have still not stopped", maxAttempts)
}

func StartToolsAndServices(binaries []string, pathOpts *PathOptions) {
	if pathOpts != nil {
		if err := UpdateGlobalPaths(pathOpts); err != nil {
			PrintRed("Failed to update paths: " + err.Error())
			os.Exit(1)
		}
	}

	if len(binaries) > 0 {
		PrintBlue(fmt.Sprintf("Starting specified binaries: %v", binaries))

		var cmdBinaries, toolsBinaries []string

		for _, binary := range binaries {
			if isExecutableFile(GetBinFullPath(binary)) {
				if runtime.GOOS == "windows" {
					binary += ".exe"
				}
				cmdBinaries = append(cmdBinaries, binary)
			}
			if isExecutableFile(GetBinToolsFullPath(binary)) {
				if runtime.GOOS == "windows" {
					binary += ".exe"
				}
				toolsBinaries = append(toolsBinaries, binary)
			}
		}

		if len(cmdBinaries) == 0 && len(toolsBinaries) == 0 {
			PrintYellow("No valid executable binaries found to start. Please build first.")
			return
		}

		PrintBlue(fmt.Sprintf("Cmd binaries to start: %v", cmdBinaries))
		PrintBlue(fmt.Sprintf("Tools binaries to start: %v", toolsBinaries))

		if len(toolsBinaries) > 0 {
			PrintBlue("Starting specified tools...")
			if err := StartTools(toolsBinaries...); err != nil {
				PrintRed("Some specified tools failed to start:")
				PrintRedNoTimeStamp(err.Error())
				return
			}
			PrintGreen("Specified tools executed successfully")
		}

		if len(cmdBinaries) > 0 {
			KillExistBinaries()
			err := attemptCheckBinaries()
			if err != nil {
				PrintRed("Some services running, details are as follows, abort start " + err.Error())
				return
			}
			err = StartBinaries(cmdBinaries...)
			if err != nil {
				PrintRed("Failed to start specified binaries:")
				PrintRedNoTimeStamp(err.Error())
				return
			}
			CheckAndReportBinariesStatus()
		}
		return
	}

	PrintBlue("Starting tools primarily involves component verification and other preparatory tasks.")
	if err := StartTools(); err != nil {
		PrintRed("Some tools failed to start, details are as follows, abort start")
		PrintRedNoTimeStamp(err.Error())
		return
	}
	PrintGreen("All tools executed successfully")

	KillExistBinaries()
	err := attemptCheckBinaries()
	if err != nil {
		PrintRed("Some services running, details are as follows, abort start " + err.Error())
		return
	}
	err = StartBinaries()
	if err != nil {
		PrintRed("Failed to start all binaries")
		PrintRedNoTimeStamp(err.Error())
		return
	}
	CheckAndReportBinariesStatus()
}

func isExecutableFile(filePath string) bool {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(filePath), ".exe") {
		filePath += ".exe"
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	if !info.Mode().IsRegular() {
		return false
	}

	if runtime.GOOS == "windows" {
		return true
	}

	return info.Mode()&0111 != 0
}

func Build(binaries []string, pathOpts *PathOptions, buildOpt *BuildOptions) {
	resolvedBuildOpt := ResolveBuildOptions(buildOpt, &BuildOptions{
		CgoEnabled: util.ResolveEnvOption[string]("CGO_ENABLED"),
		Release:    util.ResolveEnvOption[bool]("RELEASE"),
		Compress:   util.ResolveEnvOption[bool]("COMPRESS"),
		Platforms:  util.ResolveEnvOption[[]string]("PLATFORMS"),
	})

	if _, err := os.Stat(StartConfigFile); err == nil {
		InitForSSC()
	}

	if pathOpts != nil {
		if err := UpdateGlobalPaths(pathOpts); err != nil {
			PrintRed("Failed to update paths: " + err.Error())
			os.Exit(1)
		}
	}

	compileBinaries := getBinaries(binaries)
	if cgoEnabled := resolvedBuildOpt.GetCgoEnabled(); cgoEnabled != "" {
		PrintBlue(fmt.Sprintf("CGO_ENABLED %s", cgoEnabled))
	}
	platforms := resolvedBuildOpt.GetPlatforms()
	if len(platforms) == 0 {
		platforms = []string{DetectPlatform()}
	}
	for _, platform := range platforms {
		CompileForPlatform(resolvedBuildOpt, platform, compileBinaries)
	}
	PrintGreen("All specified binaries under cmd and tools were successfully compiled.")
}
