package mageutil

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/openimsdk/gomake/internal/util"
	"github.com/openimsdk/tools/utils/datautil"
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
	Build(nil, nil, exportOpt.GetBuildOpt())

	tmpDir := Paths.OutputTmp
	exportDir := Paths.OutputExport
	PrintBlue(fmt.Sprintf("Using tmp directory: %s", tmpDir))
	PrintBlue(fmt.Sprintf("Using export directory: %s", exportDir))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create tmp directory %s: %v", tmpDir, err)
	}
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory %s: %v", exportDir, err)
	}

	platforms := os.Getenv("PLATFORMS")
	if platforms == "" {
		platforms = DetectPlatform()
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
		cmd := exec.Command("mage", "-compile", mageBinaryPath, "-goos", targetOS, "-goarch", targetArch, "-ldflags", "-s -w")
		cmd.Dir = Paths.Root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to compile mage for %s: %v", platform, err)
		}
		PrintGreen(fmt.Sprintf("Mage binary compiled: %s", mageBinaryPath))

		mappingPaths, err := EnsureRootRelPaths(
			filepath.Join(Paths.OutputBinPath, targetOS, targetArch),
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
		for k, v := range overrideMappingPaths {
			mappingPaths[k] = v
		}

		archiveName := exportArchiveBaseName(platform, exportOpt)
		err = archive(filepath.Join(exportDir, archiveName), mappingPaths)
		if err != nil {
			return err
		}
	}
	return nil
}

func exportArchiveBaseName(platform string, exportOpt *ExportOptions) string {
	projectName := exportOpt.GetProjectName()
	if projectName == "" {
		return fmt.Sprintf("exported_%s", platform)
	}
	return fmt.Sprintf("exported_%s_%s", projectName, platform)
}

func archive(archivePath string, mappingPaths map[string]string) error {
	archivePath = fmt.Sprintf("%s.tar.gz", archivePath)
	PrintBlue(fmt.Sprintf("Creating archive: %s", archivePath))
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file %s: %v", archivePath, err)
	}
	defer archiveFile.Close()
	gzipWriter, err := gzip.NewWriterLevel(archiveFile, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %v", err)
	}
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for in, out := range mappingPaths {
		err := util.CheckExist(in)
		if err != nil {
			return err
		}

		PrintBlue(fmt.Sprintf("Adding %s to archive", in))
		if err := util.AddToTar(tarWriter, in, out); err != nil {
			return fmt.Errorf("failed to add %s to archive: %v", in, err)
		}
	}

	PrintGreen(fmt.Sprintf("Archive created successfully: %s", archivePath))
	return nil
}

func EnsureRootRelPaths(paths ...string) (map[string]string, error) {
	relPathMap := make(map[string]string)
	for _, path := range paths {
		relPath, err := filepath.Rel(Paths.Root, path)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}
		relPathMap[path] = relPath
	}

	return relPathMap, nil
}

func GetAllRootFilesExcludeIgnore() ([]string, error) {
	root := Paths.Root
	if root == "" {
		return nil, fmt.Errorf("root path is empty")
	}

	cmd := exec.Command("git", "ls-files", "-co", "--exclude-standard", "-z")
	cmd.Dir = root

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to list root files via git ls-files: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("failed to list root files via git ls-files: %v", err)
	}

	ret := make([]string, 0)
	for _, relPath := range strings.Split(string(output), "\x00") {
		if relPath == "" {
			continue
		}

		cleanRelPath := filepath.Clean(filepath.FromSlash(relPath))
		if cleanRelPath == "." {
			continue
		}

		absPath := filepath.Join(root, cleanRelPath)
		info, statErr := os.Stat(absPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return nil, fmt.Errorf("failed to stat file %s listed by git: %v", absPath, statErr)
		}
		if info.IsDir() {
			continue
		}

		ret = append(ret, filepath.ToSlash(absPath))
	}

	if len(ret) == 0 {
		return nil, fmt.Errorf("no files found under root %s after applying gitignore rules", root)
	}

	return ret, nil
}

func GetDefaultExportMappingPaths() (map[string]string, error) {
	allFiles, err := GetAllRootFilesExcludeIgnore()
	if err != nil {
		return nil, err
	}
	excludeSuffix := []string{
		".go",
		".proto",
	}
	allFilteredFiles := datautil.Filter(allFiles, func(e string) (string, bool) {
		ok := true
		for _, suffix := range excludeSuffix {
			if strings.HasSuffix(e, suffix) {
				ok = false
				break
			}
		}
		return e, ok
	})
	mappingPaths, err := EnsureRootRelPaths(allFilteredFiles...)
	if err != nil {
		return nil, err
	}
	return mappingPaths, nil
}
