package mageutil

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/sh"
)

func ensureToolsInstalled() error {
	tools := map[string]string{
		"protoc-gen-go": "github.com/golang/protobuf/protoc-gen-go@latest",
	}

	// Setting GOBIN based on OS, Windows needs a different default path
	var targetDir string
	if runtime.GOOS == "windows" {
		targetDir = filepath.Join(os.Getenv("USERPROFILE"), "go", "bin")
	} else {
		targetDir = "/usr/local/bin"
	}

	os.Setenv("GOBIN", targetDir)

	for tool, path := range tools {
		if _, err := exec.LookPath(filepath.Join(targetDir, tool)); err != nil {
			PrintBlue(fmt.Sprintf("Installing %s to %s...", tool, targetDir))
			if err := sh.Run("go", "install", path); err != nil {
				return fmt.Errorf("failed to install %s: %s", tool, err)
			}
		} else {
			PrintGreen(fmt.Sprintf("%s is already installed in %s.", tool, targetDir))
		}
	}

	if _, err := exec.LookPath(filepath.Join(targetDir, "protoc")); err == nil {
		PrintGreen("protoc is already installed.")
		return nil
	}

	PrintBlue("Installing protoc...")
	return installProtoc(targetDir)
}

func installProtoc(installDir string) error {
	version := "26.1"
	baseURL := "https://github.com/protocolbuffers/protobuf/releases/download/v" + version
	archMap := map[string]string{
		"amd64": "x86_64",
		"386":   "x86",
		"arm64": "aarch64",
	}
	protocFile := "protoc-%s-%s.zip"

	osArch := runtime.GOOS + "-" + getProtocArch(archMap, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		osArch = "win64" // assuming 64-bit, for 32-bit use "win32"
	}
	fileName := fmt.Sprintf(protocFile, version, osArch)
	url := baseURL + "/" + fileName

	PrintBlue(fmt.Sprintf("URL: %s", url))

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "protoc-*.zip")
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return err
	}
	PrintBlue(fmt.Sprintf("tmp %s install %s", tmpFile.Name(), installDir))
	return unzip(tmpFile.Name(), installDir)
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func getProtocArch(archMap map[string]string, goArch string) string {
	if arch, ok := archMap[goArch]; ok {
		return arch
	}
	return goArch
}

func Protocol() error {
	if err := ensureToolsInstalled(); err != nil {
		return err
	}

	moduleName, err := getModuleNameFromGoMod()
	if err != nil {
		return fmt.Errorf("error fetching module name from go.mod: %w", err)
	}

	protoPath := "./pkg/protocol"
	dirs, err := os.ReadDir(protoPath)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			if err := compileProtoFiles(protoPath, dir.Name(), moduleName); err != nil {
				return err
			}
		}
	}
	return nil
}

func compileProtoFiles(basePath, dirName, moduleName string) error {
	protoFile := filepath.Join(basePath, dirName, dirName+".proto")
	outputDir := filepath.Join(basePath, dirName)
	module := moduleName + "/pkg/protocol/" + dirName

	// Include the base path and potentially other necessary paths
	includePaths := []string{
		basePath, // Current proto files directory
	}

	// Build the args for the protoc command
	args := []string{
		"--proto_path=" + strings.Join(includePaths, ":"), // Setting multiple proto paths
		"--go_out=plugins=grpc:" + outputDir,              // Output directory
		"--go_opt=module=" + module,                       // Module name
		protoFile,                                         // Proto file to compile
	}

	// Print which file is being compiled for clarity
	PrintBlue(fmt.Sprintf("Compiling %s...", protoFile))

	// Execute the protoc command
	if err := sh.Run("protoc", args...); err != nil {
		return fmt.Errorf("failed to compile %s: %s", protoFile, err)
	}

	// Attempt to fix 'omitempty' in all generated .pb.go files within the directory
	return fixOmitemptyInDirectory(outputDir)
}

func fixOmitemptyInDirectory(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.pb.go"))
	if err != nil {
		return fmt.Errorf("failed to list .pb.go files in %s: %s", dir, err)
	}
	PrintBlue(fmt.Sprintf("Fixing omitempty in dir %s...", dir))
	for _, file := range files {
		PrintBlue(fmt.Sprintf("Fixing omitempty in %s...", file))
		if err := RemoveOmitemptyFromFile(file); err != nil {
			return fmt.Errorf("failed to replace omitempty in %s: %s", file, err)
		}
	}
	return nil
}

func RemoveOmitemptyFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %s", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.ReplaceAll(line, ",omitempty", "")
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}

	return writeLines(lines, filePath)
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %s", err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("error writing to file: %s", err)
		}
	}
	return w.Flush()
}

// getModuleNameFromGoMod extracts the module name from go.mod file.
func getModuleNameFromGoMod() (string, error) {
	file, err := os.Open("go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			// Assuming line looks like "module github.com/user/repo"
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading go.mod: %v", err)
	}

	return "", fmt.Errorf("module directive not found in go.mod")
}
