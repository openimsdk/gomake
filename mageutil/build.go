package mageutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/openimsdk/gomake/internal/util"
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

func CompileForPlatform(buildOpt *BuildOptions, platform string, compileBinaries []string) {
	var cmdBinaries, toolsBinaries []string

	toolsPrefix := Paths.ToolsDir
	cmdPrefix := Paths.SrcDir

	if Paths.SrcDir == "." {
		cmdPrefix = ""
	}

	if toolsPrefix != "" {
		toolsPrefix += string(filepath.Separator)
	}
	if cmdPrefix != "" {
		cmdPrefix += string(filepath.Separator)
	}

	for _, binary := range compileBinaries {
		if toolsPrefix != "" && strings.HasPrefix(binary, toolsPrefix) {
			toolsBinary := strings.TrimPrefix(binary, toolsPrefix)
			toolsBinaries = append(toolsBinaries, toolsBinary)
		} else if cmdPrefix == "" || strings.HasPrefix(binary, cmdPrefix) {
			var cmdBinary string
			if cmdPrefix == "" {
				cmdBinary = binary
			} else {
				cmdBinary = strings.TrimPrefix(binary, cmdPrefix)
			}
			cmdBinaries = append(cmdBinaries, cmdBinary)
		} else {
			PrintYellow(fmt.Sprintf("Binary %s does not have a valid prefix. Skipping...", binary))
		}
	}

	PrintBlue(fmt.Sprintf("Cmd binaries: %v", cmdBinaries))
	PrintBlue(fmt.Sprintf("Tools binaries: %v", toolsBinaries))

	var cmdCompiledDirs []string
	var toolsCompiledDirs []string

	if len(cmdBinaries) > 0 {
		PrintBlue(fmt.Sprintf("Compiling cmd binaries for %s...", platform))
		cmdCompiledDirs = compileDir(buildOpt, filepath.Join(Paths.Root, Paths.SrcDir), Paths.OutputBinPath, platform, cmdBinaries)
	}

	if len(toolsBinaries) > 0 {
		PrintBlue(fmt.Sprintf("Compiling tools binaries for %s...", platform))
		toolsCompiledDirs = compileDir(buildOpt, filepath.Join(Paths.Root, Paths.ToolsDir), Paths.OutputBinToolPath, platform, toolsBinaries)
	}

	createStartConfigYML(cmdCompiledDirs, toolsCompiledDirs)
}

func compileDir(buildOpt *BuildOptions, sourceDir, outputBase, platform string, compileBinaries []string) []string {
	releaseEnabled := buildOpt.GetRelease()
	compressEnabled := buildOpt.GetCompress()
	cgoEnabled := buildOpt.GetCgoEnabled()

	PrintBlue(fmt.Sprintf("Build flags: RELEASE=%t, COMPRESS=%t", releaseEnabled, compressEnabled))

	if info, err := os.Stat(sourceDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		fmt.Printf("Failed read directory %s: %v\n", sourceDir, err)
		os.Exit(1)
	} else if !info.IsDir() {
		fmt.Printf("Failed %s is not dir\n", sourceDir)
		os.Exit(1)
	}

	targetOS, targetArch := strings.Split(platform, "_")[0], strings.Split(platform, "_")[1]
	outputDir := filepath.Join(outputBase, targetOS, targetArch)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Failed to create directory %s: %v\n", outputDir, err)
		os.Exit(1)
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
	if len(compileBinaries) < cpuNum {
		cpuNum = len(compileBinaries)
	}
	PrintGreen(fmt.Sprintf("The number of concurrent compilations is %d", cpuNum))

	task := make(chan int, cpuNum)
	go func() {
		for i := range compileBinaries {
			task <- i
		}
		close(task)
	}()

	res := make(chan string, 1)
	running := int64(cpuNum)

	env := map[string]string{
		"GOOS":   targetOS,
		"GOARCH": targetArch,
	}
	if cgoEnabled != "" {
		env["CGO_ENABLED"] = cgoEnabled
	}

	baseDirAbs, err := filepath.Abs(Paths.Root)
	if err != nil {
		PrintRed(fmt.Sprintf("Failed to get absolute path for root: %v", err))
		os.Exit(1)
	}

	for i := 0; i < cpuNum; i++ {
		go func() {
			defer func() {
				if atomic.AddInt64(&running, -1) == 0 {
					close(res)
				}
			}()

			for index := range task {
				originalDir := baseDirAbs

				binaryPath := filepath.Join(sourceDir, compileBinaries[index])
				path, err := util.FindMainGoFile(binaryPath)
				if err != nil {
					PrintYellow(fmt.Sprintf("Failed to walk through binary path %s: %v", binaryPath, err))
					os.Exit(1)
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

				if err := os.Chdir(goModDir); err != nil {
					PrintRed(fmt.Sprintf("Failed to change directory to %s: %v", goModDir, err))
					os.Chdir(originalDir)
					continue
				}

				outputPath := filepath.Join(outputDir, outputFileName)

				relPath, err := filepath.Rel(goModDir, path)
				if err != nil {
					PrintRed(fmt.Sprintf("Failed to get relative path: %v", err))
					os.Exit(1)
				}

				buildTarget := relPath

				PrintBlue(fmt.Sprintf("Compiling dir: %s for platform: %s binary: %s ...", dirName, platform, outputFileName))

				buildArgs := []string{"build", "-o", outputPath}
				if releaseEnabled {
					PrintBlue("Building in release mode with optimizations...")
					buildArgs = append(buildArgs, "-trimpath", "-ldflags", "-s -w")
				}
				buildArgs = append(buildArgs, buildTarget)

				err = RunWithPriority(PriorityLow, env, "go", buildArgs...)

				os.Chdir(originalDir)

				if err != nil {
					PrintRed("Compilation aborted. " + fmt.Sprintf("failed to compile %s for %s: %v", dirName, platform, err))
					os.Exit(1)
				}

				PrintGreen(fmt.Sprintf("Successfully compiled. dir: %s for platform: %s binary: %s", dirName, platform, outputFileName))

				if compressEnabled {
					PrintBlue(fmt.Sprintf("Compressing %s with UPX...", outputFileName))
					if err := RunWithPriority(PriorityLow, nil, "upx", "--lzma", outputPath); err != nil {
						PrintYellow(fmt.Sprintf("UPX compression failed for %s (non-fatal): %v", outputFileName, err))
					} else {
						PrintGreen(fmt.Sprintf("Successfully compressed with UPX: %s", outputFileName))
					}
				}

				res <- dirName
			}
		}()
	}

	compiledDirs := make([]string, 0, len(compileBinaries))
	for str := range res {
		compiledDirs = append(compiledDirs, str)
	}
	return compiledDirs
}

func createStartConfigYML(cmdDirs, toolsDirs []string) {
	configPath := filepath.Join(Paths.Root, StartConfigFile)

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		PrintBlue("start-config.yml already exists, skipping creation.")
		return
	}

	var content strings.Builder
	content.WriteString("serviceBinaries:\n")
	for _, dir := range cmdDirs {
		content.WriteString(fmt.Sprintf("  %s: 1\n", dir))
	}
	content.WriteString("toolBinaries:\n")
	for _, dir := range toolsDirs {
		content.WriteString(fmt.Sprintf("  - %s\n", dir))
	}
	content.WriteString("maxFileDescriptors: 10000\n")

	err := os.WriteFile(configPath, []byte(content.String()), 0644)
	if err != nil {
		PrintRed("Failed to create start-config.yml: " + err.Error())
		return
	}
	PrintGreen("start-config.yml created successfully.")
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

func getBinaries(binaries []string) []string {
	if len(binaries) > 0 {
		return resolveRequestedBinaries(binaries)
	}

	type binarySource struct {
		baseDir string
		prefix  string
	}

	sources := []binarySource{
		{baseDir: filepath.Join(Paths.Root, Paths.SrcDir), prefix: normalizedSourcePrefix(Paths.SrcDir)},
		{baseDir: filepath.Join(Paths.Root, Paths.ToolsDir), prefix: normalizedSourcePrefix(Paths.ToolsDir)},
	}

	var allBinaries []string
	for _, source := range sources {
		dirs, err := getSubDirectoriesBFS(source.baseDir)
		if err != nil {
			PrintYellow(fmt.Sprintf("Failed to glob pattern %s: %v", source.baseDir, err))
			continue
		}

		for _, dir := range dirs {
			allBinaries = append(allBinaries, withSourcePrefix(source.prefix, dir))
		}
	}

	return allBinaries
}

func getSubDirectoriesBFS(baseDir string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	queue := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || util.IsExcludedBinaryDir(entry.Name()) {
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
			if util.IsExcludedBinaryDir(name) {
				PrintYellow(fmt.Sprintf("Skipping excluded directory: %s", name))
				continue
			}
			queue = append(queue, filepath.Join(currentDir, name))
		}
	}

	return subDirs, nil
}

func resolveRequestedBinaries(binaries []string) []string {
	var resolved []string
	for _, binary := range binaries {
		if path, found := isCmdBinary(binary); found {
			resolved = append(resolved, path)
			continue
		}
		if path, found := isToolBinary(binary); found {
			resolved = append(resolved, path)
			continue
		}
		PrintYellow(fmt.Sprintf("Binary %s not found in cmd (%s) or tools (%s) directories. Skipping...", binary, Paths.SrcDir, Paths.ToolsDir))
	}
	fmt.Println("Resolved binaries:", resolved)
	return resolved
}

func normalizedSourcePrefix(prefix string) string {
	if prefix == "." {
		return ""
	}
	return prefix
}

func withSourcePrefix(prefix, relPath string) string {
	if prefix == "" {
		return relPath
	}
	return filepath.Join(prefix, relPath)
}

func findBinaryPath(baseDir, binaryName string) (string, bool) {
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
		if entry.Name() == binaryName {
			relativePath, err := filepath.Rel(baseDir, subDirPath)
			if err != nil {
				PrintYellow(fmt.Sprintf("Failed to get relative path for %s: %v", subDirPath, err))
				continue
			}
			return relativePath, true
		}
		if path, found := findBinaryPath(subDirPath, binaryName); found {
			return filepath.Join(entry.Name(), path), true
		}
	}
	return "", false
}

func isCmdBinary(binary string) (string, bool) {
	path, found := findBinaryPath(filepath.Join(Paths.Root, Paths.SrcDir), binary)
	if found {
		if Paths.SrcDir == "." {
			return path, true
		}

		return filepath.Join(Paths.SrcDir, path), true
	}
	return "", false
}

func isToolBinary(binary string) (string, bool) {
	path, found := findBinaryPath(filepath.Join(Paths.Root, Paths.ToolsDir), binary)
	if found {
		return filepath.Join(Paths.ToolsDir, path), true
	}
	return "", false
}
