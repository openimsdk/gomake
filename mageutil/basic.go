package mageutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/magefile/mage/sh"
)

// CheckAndReportBinariesStatus checks the running status of all binary files and reports it.
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

// StopAndCheckBinaries stops all binary processes and checks if they have all stopped.
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
			time.Sleep(1 * time.Second) // Sleep for 1 second before retrying
		}
	}
	return fmt.Errorf("already waited for %d seconds, some services have still not stopped", maxAttempts)
}

// StartToolsAndServices starts the process for tools and services.
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
			if isExecutableBinary(binary) {
				if runtime.GOOS == "windows" {
					binary += ".exe"
				}
				cmdBinaries = append(cmdBinaries, binary)
			}
			if isExecutableToolBinary(binary) {
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

// CompileForPlatform Main compile function
func CompileForPlatform(cgoEnabled string, platform string, compileBinaries []string) {
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

	// PrintBlue(fmt.Sprintf("Using cmd prefix: '%s'", cmdPrefix))
	// PrintBlue(fmt.Sprintf("Using tools prefix: '%s'", toolsPrefix))

	for _, binary := range compileBinaries {
		// PrintBlue(fmt.Sprintf("Processing binary: %s", binary))

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
			// PrintBlue(fmt.Sprintf("Added to cmd binaries: %s", cmdBinary))
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
		// PrintBlue(fmt.Sprintf("Source directory: %s", filepath.Join(Paths.Root, Paths.SrcDir)))
		// PrintBlue(fmt.Sprintf("Output directory: %s", Paths.OutputBinPath))
		cmdCompiledDirs = compileDir(cgoEnabled, filepath.Join(Paths.Root, Paths.SrcDir), Paths.OutputBinPath, platform, cmdBinaries)
	}

	if len(toolsBinaries) > 0 {
		PrintBlue(fmt.Sprintf("Compiling tools binaries for %s...", platform))
		// PrintBlue(fmt.Sprintf("Source directory: %s", filepath.Join(Paths.Root, Paths.ToolsDir)))
		// PrintBlue(fmt.Sprintf("Output directory: %s", Paths.OutputBinToolPath))
		toolsCompiledDirs = compileDir(cgoEnabled, filepath.Join(Paths.Root, Paths.ToolsDir), Paths.OutputBinToolPath, platform, toolsBinaries)
	}

	createStartConfigYML(cmdCompiledDirs, toolsCompiledDirs)
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

func getMainFile(binaryPath string) (string, error) {
	var retPath string
	err := filepath.Walk(binaryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) != "main.go" {
			return nil
		}
		retPath = path
		return nil
	})
	if err != nil {
		return "", err
	}
	return retPath, nil
}

func compileDir(cgoEnabled string, sourceDir, outputBase, platform string, compileBinaries []string) []string {
	// PrintBlue("=== compileDir called ===")
	// PrintBlue(fmt.Sprintf("sourceDir: %s", sourceDir))
	// PrintBlue(fmt.Sprintf("outputBase: %s", outputBase))
	// PrintBlue(fmt.Sprintf("platform: %s", platform))
	// PrintBlue(fmt.Sprintf("compileBinaries: %v", compileBinaries))

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
		"GOOS":        targetOS,
		"GOARCH":      targetArch,
		"CGO_ENABLED": cgoEnabled,
	}
	if cgoEnabled == "" {
		delete(env, "CGO_ENABLED")
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
				path, err := getMainFile(binaryPath)
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

				// Find Go module directory
				goModDir := findGoModDir(dir)
				if goModDir == "" {
					goModDir = "."
				}

				// checkout to the Go module directory
				if err := os.Chdir(goModDir); err != nil {
					PrintRed(fmt.Sprintf("Failed to change directory to %s: %v", goModDir, err))
					os.Chdir(originalDir)
					continue
				}

				outputPath := filepath.Join(outputDir, outputFileName)

				// get relative path from the build directory to the Go module directory
				relPath, err := filepath.Rel(goModDir, path)
				if err != nil {
					PrintRed(fmt.Sprintf("Failed to get relative path: %v", err))
					os.Exit(1)
				}

				// Use the relative path as the build target
				buildTarget := relPath

				PrintBlue(fmt.Sprintf("Compiling dir: %s for platform: %s binary: %s ...", dirName, platform, outputFileName))

				// PrintBlue(fmt.Sprintf("DEBUG: buildTarget = '%s'", buildTarget))
				// PrintBlue(fmt.Sprintf("DEBUG: goModDir = '%s'", goModDir))
				// PrintBlue(fmt.Sprintf("DEBUG: path = '%s'", path))

				buildArgs := []string{"build", "-o", outputPath}
				if strings.ToLower(os.Getenv("RELEASE")) == "true" {
					PrintBlue("Building in release mode with optimizations...")
					buildArgs = append(buildArgs, "-trimpath", "-ldflags", "-s -w")
				}
				buildArgs = append(buildArgs, buildTarget)

				err = sh.RunWith(env, "go", buildArgs...)

				os.Chdir(originalDir)

				if err != nil {
					PrintRed("Compilation aborted. " + fmt.Sprintf("failed to compile %s for %s: %v", dirName, platform, err))
					os.Exit(1)
				}

				PrintGreen(fmt.Sprintf("Successfully compiled. dir: %s for platform: %s binary: %s", dirName, platform, outputFileName))

				if strings.ToLower(os.Getenv("COMPRESS")) == "true" {
					PrintBlue(fmt.Sprintf("Compressing %s with UPX...", outputFileName))
					if err := sh.RunWith(nil, "upx", "--lzma", outputPath); err != nil {
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

func Build(binaries []string, pathOpts *PathOptions) {
	if _, err := os.Stat(StartConfigFile); err == nil {
		InitForSSC()
		KillExistBinaries()
	}

	if pathOpts != nil {
		if err := UpdateGlobalPaths(pathOpts); err != nil {
			PrintRed("Failed to update paths: " + err.Error())
			os.Exit(1)
		}
	}

	platforms := os.Getenv("PLATFORMS")
	if platforms == "" {
		platforms = DetectPlatform()
	}
	compileBinaries := getBinaries(binaries)
	cgoEnabled := os.Getenv("CGO_ENABLED")
	if cgoEnabled != "" {
		PrintBlue(fmt.Sprintf("CGO_ENABLED %s", cgoEnabled))
	}
	for _, platform := range strings.Split(platforms, " ") {
		CompileForPlatform(cgoEnabled, platform, compileBinaries)
	}
	PrintGreen("All specified binaries under cmd and tools were successfully compiled.")
}

func getBinaries(binaries []string) []string {
	if len(binaries) > 0 {
		var resolved []string
		for _, binary := range binaries {
			if path, found := isCmdBinary(binary); found {
				resolved = append(resolved, path)
			} else if path, found := isToolBinary(binary); found {
				resolved = append(resolved, path)
			} else {
				PrintYellow(fmt.Sprintf("Binary %s not found in cmd (%s) or tools (%s) directories. Skipping...", binary, Paths.SrcDir, Paths.ToolsDir))
			}
		}
		fmt.Println("Resolved binaries:", resolved)
		return resolved
	}

	var allBinaries []string
	baseDirPatterns := []string{
		filepath.Join(Paths.Root, Paths.SrcDir),
		filepath.Join(Paths.Root, Paths.ToolsDir),
	}

	// PrintBlue(fmt.Sprintf("Scanning directories: %v", baseDirPatterns))

	for _, baseDir := range baseDirPatterns {
		binaries, err := getSubDirectoriesBFS(baseDir)
		if err != nil {
			PrintYellow(fmt.Sprintf("Failed to glob pattern %s: %v", baseDir, err))
			continue
		}

		var prefix string
		if baseDir == filepath.Join(Paths.Root, Paths.SrcDir) {
			prefix = Paths.SrcDir
		} else if baseDir == filepath.Join(Paths.Root, Paths.ToolsDir) {
			prefix = Paths.ToolsDir
		}

		if prefix == "." {
			prefix = ""
		}

		for _, bin := range binaries {
			var fullPath string
			if prefix == "" {
				fullPath = bin
			} else {
				// e.g., "cmd/openim-rpc/openim-rpc-user" or "tools/seq"
				fullPath = filepath.Join(prefix, bin)
			}
			allBinaries = append(allBinaries, fullPath)
		}

		// PrintBlue(fmt.Sprintf("Found binaries in %s: %v", baseDir, binaries))
	}

	return allBinaries
}

func getSubDirectoriesBFS(baseDir string) ([]string, error) {
	var subDirs []string
	var queue []string
	var mu sync.Mutex

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return subDirs, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.EqualFold(name, "internal") {
				// PrintYellow(fmt.Sprintf("Skipping excluded directory: %s", name))
				continue
			}
			subDirPath := filepath.Join(baseDir, name)
			queue = append(queue, subDirPath)
		}
	}

	for len(queue) > 0 {
		currentDir := queue[0]
		queue = queue[1:]

		if containsMainGo(currentDir) {
			relPath, err := filepath.Rel(baseDir, currentDir)
			if err != nil {
				// PrintYellow(fmt.Sprintf("Failed to get relative path for %s: %v", currentDir, err))
				continue
			}
			mu.Lock()
			subDirs = append(subDirs, relPath)
			mu.Unlock()
			// PrintBlue(fmt.Sprintf("Added binary directory: %s", relPath))
			continue
		}

		entries, err := os.ReadDir(currentDir)
		if err != nil {
			PrintYellow(fmt.Sprintf("Failed to read directory %s: %v", currentDir, err))
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				name := entry.Name()
				if strings.HasPrefix(name, ".") || strings.EqualFold(name, "internal") {
					PrintYellow(fmt.Sprintf("Skipping excluded directory: %s", name))
					continue
				}
				subDirPath := filepath.Join(currentDir, name)
				queue = append(queue, subDirPath)
			}
		}
	}

	return subDirs, nil
}

func findBinaryPath(baseDir, binaryName string) (string, bool) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		PrintYellow(fmt.Sprintf("Failed to read directory %s: %v", baseDir, err))
		return "", false
	}

	for _, entry := range entries {
		if entry.IsDir() {
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
	}
	return "", false
}

func containsMainGo(dir string) bool {
	mainGoPath := filepath.Join(dir, "main.go")
	info, err := os.Stat(mainGoPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func isCmdBinary(binary string) (string, bool) {
	path, found := findBinaryPath(filepath.Join(Paths.Root, Paths.SrcDir), binary)
	if found {
		if Paths.SrcDir == "." {
			return path, true
		} else {
			return filepath.Join(Paths.SrcDir, path), true
		}
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

func isExecutableBinary(binary string) bool {
	fullPath := GetBinFullPath(binary)
	return isExecutableFile(fullPath)
}

func isExecutableToolBinary(binary string) bool {
	fullPath := GetBinToolsFullPath(binary)
	return isExecutableFile(fullPath)
}

func findGoModDir(startDir string) string {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			PrintBlue(fmt.Sprintf("Found go.mod at: %s", dir))
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
