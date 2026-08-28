//go:build mage

package main

import (
	"fmt"

	"github.com/openimsdk/gomake/mageutil"
)

var Default = BuildAll

var Aliases = map[string]any{
	"buildcc": BuildWithCustomConfig,
	"startcc": StartWithCustomConfig,
}

var (
	customRootDir = "."
	// customSrcDir  = "work_cmd"
	customSrcDir    = "cmd"
	customOutputDir = "_output"
	customConfigDir = "config"
	customToolsDir  = "tools"

	customExportProjectName = "gomake"
	customExportBuildOpt    *mageutil.BuildOptions
)

func BuildAll() error { return Build(nil, nil) }

// Build supports building specific services and tools.
//
// Example: `mage build -services=openim-api,openim-rpc-user -tools=seq`
func Build(services *string, tools *string) (err error) {
	defer mageutil.PrintErrPtr(&err)

	return mageutil.WithSpinnerR("Building binaries...", func() error {
		return mageutil.Build(mageutil.ParseArgList(services), mageutil.ParseArgList(tools), nil, nil)
	})
}

func BuildWithCustomConfig(services *string, tools *string) (err error) {
	defer mageutil.PrintErrPtr(&err)

	config := &mageutil.PathOptions{
		RootDir:   &customRootDir,   // default is "."(current directory)
		OutputDir: &customOutputDir, // default is "_output"
		SrcDir:    &customSrcDir,    // default is "cmd"
		ToolsDir:  &customToolsDir,  // default is "tools"
	}

	return mageutil.WithSpinnerR("Building binaries with custom config...", func() error {
		return mageutil.Build(mageutil.ParseArgList(services), mageutil.ParseArgList(tools), config, nil)
	})
}

func Start(tools *string, services *string) (err error) {
	defer mageutil.PrintErrPtr(&err)

	return mageutil.WithSpinnerR("Starting tools and services...", func() error {
		return mageutil.StartToolsAndServices(mageutil.ParseArgList(tools), mageutil.ParseArgList(services), nil)
	})
}

func StartWithCustomConfig(tools *string, services *string) (err error) {
	defer mageutil.PrintErrPtr(&err)

	config := &mageutil.PathOptions{
		RootDir:   &customRootDir,   // default is "."(current directory)
		OutputDir: &customOutputDir, // default is "_output"
		ConfigDir: &customConfigDir, // default is "config"
	}

	return mageutil.WithSpinnerR("Starting tools and services with custom config...", func() error {
		return mageutil.StartToolsAndServices(mageutil.ParseArgList(tools), mageutil.ParseArgList(services), config)
	})
}

func Stop() (err error) {
	defer mageutil.PrintErrPtr(&err)
	return mageutil.WithSpinnerR("Checking service status...", mageutil.StopAndCheckServices)
}

func Check() (err error) {
	defer mageutil.PrintErrPtr(&err)
	return mageutil.WithSpinnerR("Checking service status...", mageutil.CheckAndReportServicesStatus)
}

func Protocol() (err error) {
	defer mageutil.PrintErrPtr(&err)
	return mageutil.WithSpinnerR("Generating protocol artifacts...", mageutil.Protocol)
}

func Export() (err error) {
	defer mageutil.PrintErrPtr(&err)

	exportOpt := &mageutil.ExportOptions{
		ProjectName: &customExportProjectName,
		BuildOpt:    customExportBuildOpt,
	}
	err = mageutil.WithSpinnerR("Exporting launcher archive...", func() error {
		return mageutil.ExportMageLauncherArchived(nil, exportOpt)
	})
	if err != nil {
		return fmt.Errorf("export failed %w", err)
	}
	return nil
}
