package mageutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/openimsdk/gomake/internal/priority"
	"github.com/openimsdk/gomake/internal/util"

	"gopkg.in/yaml.v3"
)

type BuildOptions struct {
	CgoEnabled *string
	Release    *bool
	Compress   *bool
	Platforms  *[]string
}

func (opt *BuildOptions) GetCgoEnabled() string {
	return util.NilAsZero(util.NilAsZero(opt).CgoEnabled)
}

func (opt *BuildOptions) GetRelease() bool {
	return util.NilAsZero(util.NilAsZero(opt).Release)
}

func (opt *BuildOptions) GetCompress() bool {
	return util.NilAsZero(util.NilAsZero(opt).Compress)
}

func (opt *BuildOptions) GetPlatforms() []string {
	return util.NilAsZero(util.NilAsZero(opt).Platforms)
}

func CompileForPlatform(buildOpt *BuildOptions, platform string, serviceTargets, toolTargets []string) error {
	PrintBlue(fmt.Sprintf("Cmd binaries: %v", serviceTargets))
	PrintBlue(fmt.Sprintf("Tools binaries: %v", toolTargets))

	if len(serviceTargets) > 0 {
		PrintBlue(fmt.Sprintf("Compiling cmd binaries for %s...", platform))
		_, err := compileDir(buildOpt, Paths.Cmd, Paths.OutputBinServicePath, platform, serviceTargets)
		if err != nil {
			return err
		}
	}

	if len(toolTargets) > 0 {
		PrintBlue(fmt.Sprintf("Compiling tools binaries for %s...", platform))
		_, err := compileDir(buildOpt, Paths.Tools, Paths.OutputBinToolPath, platform, toolTargets)
		if err != nil {
			return err
		}
	}

	return nil
}

func compileDir(buildOpt *BuildOptions, sourceDir, outputBase, platform string, compileTargets []string) ([]string, error) {
	releaseEnabled := buildOpt.GetRelease()
	compressEnabled := buildOpt.GetCompress()
	cgoEnabled := buildOpt.GetCgoEnabled()

	PrintBlue(fmt.Sprintf("Build flags: RELEASE=%t, COMPRESS=%t", releaseEnabled, compressEnabled))

	if info, err := os.Stat(sourceDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		PrintErr(fmt.Errorf("failed read directory %s: %w", sourceDir, err))
		return nil, err
	} else if !info.IsDir() {
		err := fmt.Errorf("%s is not dir", sourceDir)
		PrintErr(fmt.Errorf("failed %w", err))
		return nil, err
	}

	platformParts := strings.SplitN(platform, "_", 2)
	if len(platformParts) != 2 {
		return nil, fmt.Errorf("invalid platform format: %s", platform)
	}
	targetOS, targetArch := platformParts[0], platformParts[1]
	outputDir := filepath.Join(outputBase, targetOS, targetArch)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		PrintErr(fmt.Errorf("failed to create directory %s: %w", outputDir, err))
		return nil, err
	}

	cpuNum := runtime.GOMAXPROCS(0)
	if cpuNum <= 0 {
		cpuNum = runtime.NumCPU()
	} else if cpuNum > runtime.NumCPU() {
		cpuNum = runtime.NumCPU()
	}
	const compilationUsage = 16
	cpuNum = cpuNum / compilationUsage
	if cpuNum%compilationUsage != 0 {
		cpuNum++
	}
	if cpuNum < 1 {
		cpuNum = 1
	}
	if len(compileTargets) < cpuNum {
		cpuNum = len(compileTargets)
	}
	PrintGreen(fmt.Sprintf("The number of concurrent compilations is %d", cpuNum))

	task := make(chan int, len(compileTargets))
	for i := range compileTargets {
		task <- i
	}
	close(task)

	res := make(chan string, len(compileTargets))
	errCh := make(chan error, cpuNum)
	var wg sync.WaitGroup

	env := map[string]string{
		"GOOS":   targetOS,
		"GOARCH": targetArch,
	}
	if cgoEnabled != "" {
		env["CGO_ENABLED"] = cgoEnabled
	}

	for i := 0; i < cpuNum; i++ {
		wg.Go(func() {
			for index := range task {
				targetPath := filepath.Join(sourceDir, compileTargets[index])
				path, err := util.FindMainGoFile(targetPath)
				if err != nil {
					PrintYellow(fmt.Sprintf("Failed to walk through build target %s: %v", targetPath, err))
					errCh <- err
					return
				}
				if path == "" {
					continue
				}

				dir := filepath.Dir(path)
				dirName := filepath.Base(dir)
				outputFileName := dirName
				if targetOS == "windows" {
					outputFileName += ".exe"
				}

				goModDir := util.FindGoModDir(dir)
				if goModDir == "" {
					goModDir = "."
				} else {
					PrintBlue(fmt.Sprintf("Found go.mod at: %s", goModDir))
				}

				outputPath := filepath.Join(outputDir, outputFileName)

				relPath, err := filepath.Rel(goModDir, path)
				if err != nil {
					PrintErr(fmt.Errorf("failed to get relative path: %w", err))
					errCh <- err
					return
				}

				buildTarget := relPath

				PrintBlue(fmt.Sprintf("Compiling dir: %s for platform: %s binary: %s ...", dirName, platform, outputFileName))

				buildArgs := []string{"build", "-o", outputPath}
				if releaseEnabled {
					PrintBlue("Building in release mode with optimizations...")
					buildArgs = append(buildArgs, "-trimpath", "-ldflags", "-s -w")
				}
				buildArgs = append(buildArgs, buildTarget)

				err = NewCmd("go").
					WithArgs(buildArgs...).
					WithDir(goModDir).
					WithEnv(env).
					WithPriority(priority.Low).
					WithStdout(GetStdoutInnerLogWriter()).
					WithStderr(GetStderrInnerLogWriter()).
					Run()

				if err != nil {
					err = fmt.Errorf("failed to compile %s for %s: %w", dirName, platform, err)
					PrintErr(fmt.Errorf("compilation aborted: %w", err))
					errCh <- err
					return
				}

				PrintGreen(fmt.Sprintf("Successfully compiled. dir: %s for platform: %s binary: %s", dirName, platform, outputFileName))

				if compressEnabled {
					PrintBlue(fmt.Sprintf("Compressing %s with UPX...", outputFileName))
					cmd := NewCmd("upx").
						WithArgs("--lzma", outputPath).
						WithPriority(priority.Low).
						WithStdout(GetStdoutInnerLogWriter()).
						WithStderr(GetStderrInnerLogWriter())
					if err := cmd.Run(); err != nil {
						PrintYellow(fmt.Sprintf("UPX compression failed for %s (non-fatal): %v", outputFileName, err))
					} else {
						PrintGreen(fmt.Sprintf("Successfully compressed with UPX: %s", outputFileName))
					}
				}

				res <- dirName
			}
		})
	}
	go func() {
		wg.Wait()
		close(res)
		close(errCh)
	}()

	compiledDirs := make([]string, 0, len(compileTargets))
	for str := range res {
		compiledDirs = append(compiledDirs, str)
	}
	for err := range errCh {
		if err != nil {
			return compiledDirs, err
		}
	}
	return compiledDirs, nil
}

func createStartConfigYML(serviceTargets, toolTargets []string) error {
	configPath := filepath.Join(Paths.Root, StartConfigFile)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		PrintBlue("start-config.yml already exists, skipping creation.")
		return nil
	}

	services := make(map[string]int, len(serviceTargets))
	for _, target := range serviceTargets {
		services[filepath.Base(target)] = 1
	}
	tools := make([]string, 0, len(toolTargets))
	for _, target := range toolTargets {
		tools = append(tools, filepath.Base(target))
	}

	content, err := yaml.Marshal(StartConfig{
		Services:           services,
		Tools:              tools,
		MaxFileDescriptors: 10000,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal start config: %w", err)
	}

	err = os.WriteFile(configPath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to create start-config.yml: %w", err)
	}
	PrintGreen("start-config.yml created successfully.")
	return nil
}

func ResolveBuildOptions(codeOpt *BuildOptions, envOpt *BuildOptions) *BuildOptions {
	fromCode := BuildOptions{}
	if codeOpt != nil {
		fromCode = *codeOpt
	}

	fromEnv := BuildOptions{}
	if envOpt != nil {
		fromEnv = *envOpt
	}

	return &BuildOptions{
		CgoEnabled: util.CoalescePtr(fromCode.CgoEnabled, fromEnv.CgoEnabled),
		Release:    util.CoalescePtr(fromCode.Release, fromEnv.Release),
		Compress:   util.CoalescePtr(fromCode.Compress, fromEnv.Compress),
		Platforms:  util.CoalescePtr(fromCode.Platforms, fromEnv.Platforms),
	}
}

func resolveCompileTargets(services, tools []string) ([]string, []string) {
	if len(services) > 0 || len(tools) > 0 {
		return resolveRequestedTargets(services, tools)
	}

	serviceTargets, err := discoverBuildTargets(Paths.Cmd)
	if err != nil {
		PrintYellow(fmt.Sprintf("Failed to glob pattern %s: %v", Paths.Cmd, err))
	}
	toolTargets, err := discoverBuildTargets(Paths.Tools)
	if err != nil {
		PrintYellow(fmt.Sprintf("Failed to glob pattern %s: %v", Paths.Tools, err))
	}

	return serviceTargets, toolTargets
}

func discoverBuildTargets(baseDir string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	queue := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || util.IsExcludedSourceDir(entry.Name()) {
			continue
		}
		queue = append(queue, filepath.Join(baseDir, entry.Name()))
	}

	var subDirs []string
	for i := 0; i < len(queue); i++ {
		currentDir := queue[i]

		if util.ContainsMainGo(currentDir) {
			relPath, err := filepath.Rel(baseDir, currentDir)
			if err == nil {
				subDirs = append(subDirs, relPath)
			}
			continue
		}

		children, err := os.ReadDir(currentDir)
		if err != nil {
			PrintYellow(fmt.Sprintf("Failed to read directory %s: %v", currentDir, err))
			continue
		}

		for _, child := range children {
			if !child.IsDir() {
				continue
			}
			name := child.Name()
			if util.IsExcludedSourceDir(name) {
				PrintYellow(fmt.Sprintf("Skipping excluded directory: %s", name))
				continue
			}
			queue = append(queue, filepath.Join(currentDir, name))
		}
	}

	return subDirs, nil
}

func resolveRequestedTargets(services, tools []string) ([]string, []string) {
	var serviceTargets, toolTargets []string
	for _, service := range services {
		if path, found := findTargetByName(Paths.Cmd, service); found {
			serviceTargets = append(serviceTargets, path)
		} else {
			PrintYellow(fmt.Sprintf("Service %s not found in source directory %s. Skipping...", service, Paths.Cmd))
		}
	}
	for _, tool := range tools {
		if path, found := findTargetByName(Paths.Tools, tool); found {
			toolTargets = append(toolTargets, path)
		} else {
			PrintYellow(fmt.Sprintf("Tool %s not found in source directory %s. Skipping...", tool, Paths.Tools))
		}
	}
	PrintBlue(fmt.Sprintf("Resolved service targets: %v", serviceTargets))
	PrintBlue(fmt.Sprintf("Resolved tool targets: %v", toolTargets))
	return serviceTargets, toolTargets
}

func findTargetByName(baseDir, targetName string) (string, bool) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		PrintYellow(fmt.Sprintf("Failed to read directory %s: %v", baseDir, err))
		return "", false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDirPath := filepath.Join(baseDir, entry.Name())
		if entry.Name() == targetName && util.ContainsMainGo(subDirPath) {
			relativePath, err := filepath.Rel(baseDir, subDirPath)
			if err != nil {
				PrintYellow(fmt.Sprintf("Failed to get relative path for %s: %v", subDirPath, err))
				continue
			}
			return relativePath, true
		}
		if path, found := findTargetByName(subDirPath, targetName); found {
			return filepath.Join(entry.Name(), path), true
		}
	}
	return "", false
}
