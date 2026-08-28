package mageutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/openimsdk/gomake/internal/util"
	"github.com/openimsdk/tools/utils/datautil"
)

// Path constants
const (
	MountConfigFilePath = "CONFIG_PATH"
	DeploymentType      = "DEPLOYMENT_TYPE"
	KUBERNETES          = "kubernetes"

	// Directory command constants
	ConfigDir    = "config"
	OutputDir    = "_output"
	SrcDir       = "cmd"
	ToolsDir     = "tools"
	TmpDir       = "tmp"
	ExportDir    = "export"
	ArchiveDir   = "archive"
	LogDir       = "logs"
	BinDir       = "bin"
	PlatformsDir = "platforms"
)

// PathConfig represents the path configuration structure
type PathConfig struct {
	Root                 string
	Config               string
	K8sConfig            string
	Output               string
	OutputTools          string
	OutputTmp            string
	OutputExport         string
	OutputArchive        string
	OutputLog            string
	OutputBin            string
	OutputBinServicePath string
	OutputBinToolPath    string
	OutputHostBinService string
	OutputHostBinTool    string

	Cmd   string // Source cmd directory
	Tools string // Source tools directory
}

type PathOptions struct {
	RootDir   *string // Custom root directory, default is current working directory(./)
	OutputDir *string // Custom output directory command, default is "_output"
	ConfigDir *string // Custom config directory command, default is "config"

	SrcDir   *string // Custom cmd source directory command, default is "cmd"
	ToolsDir *string // Custom tools source directory command, default is "tools"
}

var Paths *PathConfig

func init() {
	var err error
	Paths, err = NewPathConfig(nil)
	if err != nil {
		panic("Failed to initialize paths: " + err.Error())
	}
}

// NewPathConfig creates a new path configuration with optional settings
func NewPathConfig(opts *PathOptions) (*PathConfig, error) {
	// Determine root directory
	var rootDir string
	if opts != nil && opts.RootDir != nil {
		rootDir = *opts.RootDir
	} else {
		currentDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting current directory: %w", err)
		}
		rootDir = currentDir
	}
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error resolving root directory: %w", err)
	}
	rootDir, err = filepath.EvalSymlinks(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error resolving root directory symlinks: %w", err)
	}

	// Determine source directories
	srcDir := SrcDir
	if opts != nil && opts.SrcDir != nil {
		srcDir = *opts.SrcDir
	}

	toolsDir := ToolsDir
	if opts != nil && opts.ToolsDir != nil {
		toolsDir = *opts.ToolsDir
	}

	// Determine other directories
	configDir := ConfigDir
	if opts != nil && opts.ConfigDir != nil {
		configDir = *opts.ConfigDir
	}

	outputDir := OutputDir
	if opts != nil && opts.OutputDir != nil {
		outputDir = *opts.OutputDir
	}

	config := &PathConfig{
		Root: rootDir,
	}

	// Set base paths
	config.Cmd = config.joinPath(config.Root, srcDir)
	config.Tools = config.joinPath(config.Root, toolsDir)
	config.Config = config.joinPath(config.Root, configDir)
	config.Output = config.joinPath(config.Root, outputDir)

	// Set output subdirectories
	config.OutputTools = config.joinPath(config.Output, ToolsDir)
	config.OutputTmp = config.joinPath(config.Output, TmpDir)
	config.OutputExport = config.joinPath(config.Output, ExportDir)
	config.OutputArchive = config.joinPath(config.Output, ArchiveDir)
	config.OutputLog = config.joinPath(config.Output, LogDir)
	config.OutputBin = config.joinPath(config.Output, BinDir)

	// Set binary file paths
	config.OutputBinServicePath = config.joinPath(config.Output, BinDir, PlatformsDir)
	config.OutputBinToolPath = config.joinPath(config.Output, BinDir, ToolsDir)

	// Set host-specific paths
	osArch := OsArch()
	config.OutputHostBinService = config.joinPath(config.OutputBinServicePath, osArch)
	config.OutputHostBinTool = config.joinPath(config.OutputBinToolPath, osArch)

	// Handle Kubernetes configuration
	if os.Getenv(DeploymentType) == KUBERNETES {
		config.K8sConfig = config.joinPath("/", configDir)
	}

	// Create all necessary directories
	if err := config.createDirectories(); err != nil {
		return nil, err
	}

	return config, nil
}

// UpdateGlobalPaths updates the global Paths variable with new options
func UpdateGlobalPaths(opts *PathOptions) error {
	if opts == nil {
		return nil // No changes needed
	}

	PrintBlue("Updating global paths with custom configuration...")

	newPaths, err := NewPathConfig(opts)
	if err != nil {
		return fmt.Errorf("failed to create new path config: %w", err)
	}

	Paths = newPaths

	PrintBlue("======== Path Configuration ========")
	PrintBlue(fmt.Sprintf("Root: %s", Paths.Root))
	PrintBlue(fmt.Sprintf("Output: %s", Paths.Output))
	PrintBlue(fmt.Sprintf("Config: %s", Paths.Config))

	PrintBlue(fmt.Sprintf("SrcDir: %s", Paths.Cmd))
	PrintBlue(fmt.Sprintf("ToolsDir: %s", Paths.Tools))

	PrintGreen("======== Global paths updated successfully ========")
	return nil
}

// joinPath helper method: joins path and adds separator
func (p *PathConfig) joinPath(elements ...string) string {
	path := filepath.Join(elements...)
	return path + string(filepath.Separator)
}

// createDirectories creates all necessary directories
func (p *PathConfig) createDirectories() error {
	dirs := []string{
		p.Config,
		p.Output,
		p.OutputTools,
		p.OutputTmp,
		p.OutputExport,
		p.OutputArchive,
		p.OutputLog,
		p.OutputBin,
		p.OutputBinServicePath,
		p.OutputBinToolPath,
		p.OutputHostBinService,
		p.OutputHostBinTool,
	}

	for _, dir := range dirs {
		if err := p.createDirIfNotExist(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// createDirIfNotExist creates directory if it doesn't exist
func (p *PathConfig) createDirIfNotExist(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// GetBinServiceFullPath returns the full path for a binary file
func (p *PathConfig) GetBinServiceFullPath(binName string) string {
	return filepath.Join(p.OutputHostBinService, binName)
}

// GetBinToolFullPath GetToolFullPath returns the full path for a tool
func (p *PathConfig) GetBinToolFullPath(toolName string) string {
	return filepath.Join(p.OutputHostBinTool, toolName)
}

func EnsureRootRelPaths(paths ...string) (map[string]string, error) {
	root := filepath.Clean(Paths.Root)
	if root == "" {
		return nil, fmt.Errorf("root path is empty")
	}

	relPathMap := make(map[string]string)
	for _, path := range paths {
		absPath := filepath.Clean(filepath.FromSlash(path))
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(root, absPath)
		}

		relPath, err := filepath.Rel(root, absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}
		relPathMap[absPath] = filepath.ToSlash(relPath)
	}

	return relPathMap, nil
}

func GetAllRootFilesExcludeIgnore() ([]string, error) {
	root := Paths.Root
	if root == "" {
		return nil, fmt.Errorf("root path is empty")
	}

	cmdOutput, err := NewCmd("git").
		WithArgs("ls-files", "-c", "-o", "--exclude-standard", "-z").
		WithDir(root).
		Output()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to list root files via git ls-files: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("failed to list root files via git ls-files: %v", err)
	}

	relPaths := make([]string, 0)
	for _, relPath := range strings.Split(string(cmdOutput), "\x00") {
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

		relPaths = append(relPaths, filepath.ToSlash(cleanRelPath))
	}

	if len(relPaths) == 0 {
		return nil, fmt.Errorf("no files found under root %s after applying gitignore rules", root)
	}

	return relPaths, nil
}

func GetDefaultExportMappingPaths(exclude []string) (map[string]string, error) {
	allFiles, err := GetAllRootFilesExcludeIgnore()
	if err != nil {
		return nil, err
	}

	allFilteredFiles := datautil.Filter(allFiles, func(e string) (string, bool) {
		if util.MatchAnyFilepathGlob(e, exclude) {
			return "", false
		}
		return e, true
	})

	return EnsureRootRelPaths(allFilteredFiles...)
}
