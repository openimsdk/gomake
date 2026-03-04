//go:build mage
// +build mage

package main

import (
	"flag"
	"os"

	"github.com/openimsdk/gomake/mageutil"
)

var Default = Build

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

// Build support specifical binary build.
//
// Example: `mage build openim-api openim-rpc-user seq`
func Build() {
	flag.Parse()
	bin := flag.Args()
	if len(bin) != 0 {
		bin = bin[1:]
	}

	mageutil.WithSpinner("Building binaries...", func() {
		mageutil.Build(bin, nil, nil)
	})
}

func BuildWithCustomConfig() {
	flag.Parse()
	bin := flag.Args()
	if len(bin) != 0 {
		bin = bin[1:]
	}

	config := &mageutil.PathOptions{
		RootDir:   &customRootDir,   // default is "."(current directory)
		OutputDir: &customOutputDir, // default is "_output"
		SrcDir:    &customSrcDir,    // default is "cmd"
		ToolsDir:  &customToolsDir,  // default is "tools"
	}

	mageutil.WithSpinner("Building binaries with custom config...", func() {
		mageutil.Build(bin, config, nil)
	})
}

func Start() {
	mageutil.InitForSSC()
	err := setMaxOpenFiles()
	if err != nil {
		mageutil.PrintRed("setMaxOpenFiles failed " + err.Error())
		os.Exit(1)
	}

	flag.Parse()
	bin := flag.Args()
	if len(bin) != 0 {
		bin = bin[1:]
	}

	mageutil.WithSpinner("Starting tools and services...", func() {
		mageutil.StartToolsAndServices(bin, nil)
	})
}

func StartWithCustomConfig() {
	mageutil.InitForSSC()
	err := setMaxOpenFiles()
	if err != nil {
		mageutil.PrintRed("setMaxOpenFiles failed " + err.Error())
		os.Exit(1)
	}

	flag.Parse()
	bin := flag.Args()
	if len(bin) != 0 {
		bin = bin[1:]
	}

	config := &mageutil.PathOptions{
		RootDir:   &customRootDir,   // default is "."(current directory)
		OutputDir: &customOutputDir, // default is "_output"
		ConfigDir: &customConfigDir, // default is "config"
	}

	mageutil.WithSpinner("Starting tools and services with custom config...", func() {
		mageutil.StartToolsAndServices(bin, config)
	})
}

func Stop() {
	mageutil.WithSpinner("Checking service status...", mageutil.StopAndCheckBinaries)
}

func Check() {
	mageutil.WithSpinner("Checking service status...", mageutil.CheckAndReportBinariesStatus)
}

func Protocol() {
	mageutil.WithSpinnerE("Generating protocol artifacts...", mageutil.Protocol)
}

func Export() {
	exportOpt := &mageutil.ExportOptions{
		ProjectName: &customExportProjectName,
		BuildOpt:    customExportBuildOpt,
	}
	err := mageutil.WithSpinnerE("Exporting launcher archive...", func() error { return mageutil.ExportMageLauncherArchived(nil, exportOpt) })
	if err != nil {
		mageutil.PrintRed("export failed " + err.Error())
		os.Exit(1)
	}
}
