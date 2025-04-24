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
func StartToolsAndServices() {
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
	PrintBlue("Starting services involves multiple RPCs and APIs and may take some time. Please be patient")
	err = StartBinaries()
	if err != nil {
		PrintRed("Failed to start all binaries")
		PrintRedNoTimeStamp(err.Error())
		return
	}
	CheckAndReportBinariesStatus()
}

// CompileForPlatform Main compile function
func CompileForPlatform(platform string, compileBinaries []string) {
	var cmdBinaries, toolsBinaries []string

	toolsPrefix := filepath.Join("tools", "")
	cmdPrefix := filepath.Join("cmd", "")

	for _, binary := range compileBinaries {
		if strings.HasPrefix(binary, toolsPrefix) {
			toolsBinaries = append(toolsBinaries, strings.TrimPrefix(binary, toolsPrefix))
		} else if strings.HasPrefix(binary, cmdPrefix) {
			cmdBinaries = append(cmdBinaries, strings.TrimPrefix(binary, cmdPrefix))
		} else {
			PrintYellow(fmt.Sprintf("Binary %s does not have a valid prefix. Skipping...", binary))
		}
	}

	var cmdCompiledDirs []string
	var toolsCompiledDirs []string

	if len(cmdBinaries) > 0 {
		PrintBlue(fmt.Sprintf("Compiling cmd binaries for %s...", platform))
		cmdCompiledDirs = compileDir(filepath.Join(rootDirPath, "cmd"), OpenIMOutputBinPath, platform, cmdBinaries)
	}

	if len(toolsBinaries) > 0 {
		PrintBlue(fmt.Sprintf("Compiling tools binaries for %s...", platform))
		toolsCompiledDirs = compileDir(filepath.Join(rootDirPath, "tools"), OpenIMOutputBinToolPath, platform, toolsBinaries)
	}

	createStartConfigYML(cmdCompiledDirs, toolsCompiledDirs)
}

func createStartConfigYML(cmdDirs, toolsDirs []string) {
	configPath := filepath.Join(rootDirPath, "start-config.yml")

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

func compileDir(sourceDir, outputBase, platform string, compileBinaries []string) []string {
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

	for i := 0; i < cpuNum; i++ {
		go func() {
			defer func() {
				if atomic.AddInt64(&running, -1) == 0 {
					close(res)
				}
			}()
			for index := range task {
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
				PrintBlue(fmt.Sprintf("Compiling dir: %s for platform: %s binary: %s ...", dirName, platform, outputFileName))
				err = sh.RunWith(map[string]string{
					"GOOS":   targetOS,
					"GOARCH": targetArch,
				}, "go", "build", "-o", filepath.Join(outputDir, outputFileName), path)
				if err != nil {
					PrintRed("Compilation aborted. " + fmt.Sprintf("failed to compile %s for %s: %v", dirName, platform, err))
					os.Exit(1)
				}
				PrintGreen(fmt.Sprintf("Successfully compiled. dir: %s for platform: %s binary: %s", dirName, platform, outputFileName))
				res <- dirName
			}
		}()
	}

	compiledDirs := make([]string, 0, len(compileBinaries))
	for str := range res {
		compiledDirs = append(compiledDirs, str)
	}
	return compiledDirs

	//var compiledDirs []string
	//var mu sync.Mutex
	//var wg sync.WaitGroup
	//errors := make(chan error, 1)
	//sem := make(chan struct{}, 4)
	//
	//for _, binary := range compileBinaries {
	//	binaryPath := filepath.Join(sourceDir, binary)
	//	// PrintBlue(fmt.Sprintf("Walking through binary path: %s", binaryPath))
	//
	//	err := filepath.Walk(binaryPath, func(path string, info os.FileInfo, err error) error {
	//		if err != nil {
	//			return err
	//		}
	//		if info.IsDir() {
	//			return nil
	//		}
	//		if filepath.Base(path) != "main.go" {
	//			return nil
	//		}
	//
	//		dir := filepath.Dir(path)
	//		dirName := filepath.Base(dir)
	//
	//		wg.Add(1)
	//		go func(dir, dirName string) {
	//			sem <- struct{}{}
	//			defer wg.Done()
	//			defer func() { <-sem }()
	//
	//			outputFileName := dirName
	//			if targetOS == "windows" {
	//				outputFileName += ".exe"
	//			}
	//
	//			PrintBlue(fmt.Sprintf("Compiling dir: %s for platform: %s binary: %s ...", dirName, platform, outputFileName))
	//			err := sh.RunWith(map[string]string{
	//				"GOOS":   targetOS,
	//				"GOARCH": targetArch,
	//			}, "go", "build", "-o", filepath.Join(outputDir, outputFileName), path)
	//			if err != nil {
	//				errors <- fmt.Errorf("failed to compile %s for %s: %v", dirName, platform, err)
	//				PrintRed("Compilation aborted. " + fmt.Sprintf("failed to compile %s for %s: %v", dirName, platform, err))
	//				os.Exit(1)
	//				return
	//			}
	//			PrintGreen(fmt.Sprintf("Successfully compiled. dir: %s for platform: %s binary: %s", dirName, platform, outputFileName))
	//			mu.Lock()
	//			compiledDirs = append(compiledDirs, dirName)
	//			mu.Unlock()
	//		}(dir, dirName)
	//
	//		return nil
	//	})
	//
	//	if err != nil {
	//		PrintYellow(fmt.Sprintf("Failed to walk through binary path %s: %v", binaryPath, err))
	//		os.Exit(1)
	//	}
	//}
	//wg.Wait()
	//close(errors)
	//
	//// Check for errors
	//if err, ok := <-errors; ok {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//
	//return compiledDirs
}

func Build(binaries []string) {
	if _, err := os.Stat("start-config.yml"); err == nil {
		InitForSSC()
		KillExistBinaries()
	}

	platforms := os.Getenv("PLATFORMS")
	if platforms == "" {
		platforms = DetectPlatform()
	}
	compileBinaries := getBinaries(binaries)

	for _, platform := range strings.Split(platforms, " ") {
		CompileForPlatform(platform, compileBinaries)
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
				PrintYellow(fmt.Sprintf("Binary %s not found in cmd or tools directories. Skipping...", binary))
			}
		}
		fmt.Println("Resolved binaries:", resolved)
		return resolved
	}

	var allBinaries []string
	baseDirPatterns := []string{
		filepath.Join(rootDirPath, "cmd"),
		filepath.Join(rootDirPath, "tools"),
	}

	for _, baseDir := range baseDirPatterns {
		binaries, err := getSubDirectoriesBFS(baseDir)
		if err != nil {
			PrintYellow(fmt.Sprintf("Failed to glob pattern %s: %v", baseDir, err))
			continue
		}

		prefix := filepath.Base(baseDir)

		for _, bin := range binaries {
			fullPath := filepath.Join(prefix, bin) // e.g., "cmd/openim-rpc/openim-rpc-user" or "tools/seq"
			allBinaries = append(allBinaries, fullPath)
			// PrintBlue(fmt.Sprintf("Discovered binary: %s", fullPath))
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
	path, found := findBinaryPath(filepath.Join(rootDirPath, "cmd"), binary)
	if found {
		return filepath.Join("cmd", path), true
	}
	return "", false
}

func isToolBinary(binary string) (string, bool) {
	path, found := findBinaryPath(filepath.Join(rootDirPath, "tools"), binary)
	if found {
		return filepath.Join("tools", path), true
	}
	return "", false
}
