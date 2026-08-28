package mageutil

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/openimsdk/gomake/internal/util"
)

type ExportOptions struct {
	ProjectName *string
	BuildOpt    *BuildOptions
}

func (opt *ExportOptions) GetProjectName() string {
	projectName := strings.TrimSpace(util.NilAsZero(util.NilAsZero(opt).ProjectName))
	if projectName == "" {
		return ""
	}
	return strings.NewReplacer("/", "_", "\\", "_").Replace(projectName)
}

func (opt *ExportOptions) GetBuildOpt() *BuildOptions {
	return util.NilAsZero(opt).BuildOpt
}

func ExportMageLauncherArchived(overrideMappingPaths map[string]string, exportOpt *ExportOptions) error {
	PrintBlue("Preparing launcher archive export...")
	PrintBlue("Building binaries before export...")
	if err := Build(nil, nil, nil, exportOpt.GetBuildOpt()); err != nil {
		return err
	}

	tmpDir := Paths.OutputTmp
	exportDir := Paths.OutputExport
	PrintBlue(fmt.Sprintf("Using tmp directory: %s", tmpDir))
	PrintBlue(fmt.Sprintf("Using export directory: %s", exportDir))

	platforms := os.Getenv("PLATFORMS")
	if platforms == "" {
		platform, err := DetectPlatform()
		if err != nil {
			return err
		}
		platforms = platform
	}

	platformList := strings.Fields(platforms)
	if len(platformList) == 0 {
		return fmt.Errorf("no platforms specified for export")
	}

	for _, platform := range platformList {
		PrintBlue(fmt.Sprintf("Target platform: %s", platform))
		platformParts := strings.SplitN(platform, "_", 2)
		if len(platformParts) != 2 {
			return fmt.Errorf("invalid platform format: %s", platform)
		}
		targetOS, targetArch := platformParts[0], platformParts[1]

		mageBinaryPath := filepath.Join(tmpDir, fmt.Sprintf("mage_%s", platform))
		if targetOS == "windows" {
			mageBinaryPath += ".exe"
		}
		PrintBlue(fmt.Sprintf("Compiling mage binary for %s: mage -compile %s", platform, mageBinaryPath))
		cmd := NewCmd("mage").
			WithArgs("-compile", mageBinaryPath, "-goos", targetOS, "-goarch", targetArch, "-ldflags", "-s -w").
			WithDir(Paths.Root).
			WithStdout(GetStdoutInnerLogWriter()).
			WithStderr(GetStderrInnerLogWriter())
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to compile mage for %s: %v", platform, err)
		}
		PrintGreen(fmt.Sprintf("Mage binary compiled: %s", mageBinaryPath))

		mappingPaths, err := EnsureRootRelPaths(
			filepath.Join(Paths.OutputBinServicePath, targetOS, targetArch),
			filepath.Join(Paths.OutputBinToolPath, targetOS, targetArch),
			filepath.Join(Paths.Root, StartConfigFile),
		)
		if err != nil {
			return err
		}

		mageInPath := mageBinaryPath
		mageOutPath := "mage"
		if targetOS == "windows" {
			mageOutPath = "mage.exe"
		}

		mappingPaths[mageInPath] = mageOutPath
		maps.Copy(mappingPaths, overrideMappingPaths)

		archiveName := fmt.Sprintf("exported_%s", platform)
		projectName := exportOpt.GetProjectName()
		if projectName != "" {
			archiveName = fmt.Sprintf("exported_%s_%s", projectName, platform)
		}
		err = archive(filepath.Join(exportDir, archiveName), mappingPaths)
		if err != nil {
			return err
		}
	}
	return nil
}
