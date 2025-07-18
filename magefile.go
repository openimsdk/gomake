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

	mageutil.Build(bin, nil)
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

	mageutil.Build(bin, config)
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

	mageutil.StartToolsAndServices(bin, nil)
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
		ConfigDir: &customConfigDir, // default is "config"
	}

	mageutil.StartToolsAndServices(bin, config)
}

func Stop() {
	mageutil.StopAndCheckBinaries()
}

func Check() {
	mageutil.CheckAndReportBinariesStatus()
}

func Protocol() {
	mageutil.Protocol()
}
