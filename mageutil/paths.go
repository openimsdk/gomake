package mageutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// Path constants
const (
	MountConfigFilePath = "CONFIG_PATH"
	DeploymentType      = "DEPLOYMENT_TYPE"
	KUBERNETES          = "kubernetes"

	// Directory name constants
	ConfigDir    = "config"
	OutputDir    = "_output"
	SrcDir       = "cmd"
	ToolsDir     = "tools"
	TmpDir       = "tmp"
	LogsDir      = "logs"
	BinDir       = "bin"
	PlatformsDir = "platforms"
)

// PathConfig represents the path configuration structure
type PathConfig struct {
	Root               string
	Config             string
	K8sConfig          string
	Output             string
	OutputTools        string
	OutputTmp          string
	OutputLogs         string
	OutputBin          string
	OutputBinPath      string
	OutputBinToolPath  string
	OutputHostBin      string
	OutputHostBinTools string

	SrcDir   string // Source cmd directory
	ToolsDir string // Source tools directory
}

type PathOptions struct {
	RootDir   *string // Custom root directory, default is current working directory(./)
	OutputDir *string // Custom output directory name, default is "_output"
	ConfigDir *string // Custom config directory name, default is "config"

	SrcDir   *string // Custom cmd source directory name, default is "cmd"
	ToolsDir *string // Custom tools source directory name, default is "tools"
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
		// rootDir = *opts.RootDir
		rootDir, _ = filepath.Abs(*opts.RootDir)
	} else {
		currentDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting current directory: %w", err)
		}
		rootDir = currentDir
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
		Root:     rootDir,
		SrcDir:   srcDir,
		ToolsDir: toolsDir,
	}

	// Set base paths
	config.Config = config.joinPath(config.Root, configDir)
	config.Output = config.joinPath(config.Root, outputDir)

	// Set output subdirectories
	config.OutputTools = config.joinPath(config.Output, ToolsDir)
	config.OutputTmp = config.joinPath(config.Output, TmpDir)
	config.OutputLogs = config.joinPath(config.Output, LogsDir)
	config.OutputBin = config.joinPath(config.Output, BinDir)

	// Set binary file paths
	config.OutputBinPath = config.joinPath(config.Output, BinDir, PlatformsDir)
	config.OutputBinToolPath = config.joinPath(config.Output, BinDir, ToolsDir)

	// Set host-specific paths
	osArch := OsArch()
	config.OutputHostBin = config.joinPath(config.OutputBinPath, osArch)
	config.OutputHostBinTools = config.joinPath(config.OutputBinToolPath, osArch)

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

	PrintBlue(fmt.Sprintf("SrcDir: %s", Paths.SrcDir))
	PrintBlue(fmt.Sprintf("ToolsDir: %s", Paths.ToolsDir))

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
		p.OutputLogs,
		p.OutputBin,
		p.OutputBinPath,
		p.OutputBinToolPath,
		p.OutputHostBin,
		p.OutputHostBinTools,
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

// GetBinFullPath returns the full path for a binary file
func (p *PathConfig) GetBinFullPath(binName string) string {
	return filepath.Join(p.OutputHostBin, binName)
}

// GetToolFullPath returns the full path for a tool
func (p *PathConfig) GetBinToolsFullPath(toolName string) string {
	return filepath.Join(p.OutputHostBinTools, toolName)
}

// Compatibility: maintain original global functions
func GetBinFullPath(binName string) string {
	return Paths.GetBinFullPath(binName)
}

func GetBinToolsFullPath(toolName string) string {
	return Paths.GetBinToolsFullPath(toolName)
}
