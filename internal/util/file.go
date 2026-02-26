package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CheckExist(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return fmt.Errorf("file or directory does not exist: %s", path)
	}
	return fmt.Errorf("failed to stat %s: %v", path, err)
}

func NormalizeExePath(path string) string {
	const deletedSuffix = " (deleted)"
	if strings.HasSuffix(path, deletedSuffix) {
		return strings.TrimSuffix(path, deletedSuffix)
	}
	return path
}

func ContainsMainGo(dir string) bool {
	mainGoPath := filepath.Join(dir, "main.go")
	info, err := os.Stat(mainGoPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func FindMainGoFile(binaryPath string) (string, error) {
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

func FindGoModDir(startDir string) string {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
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

func IsExcludedBinaryDir(name string) bool {
	return strings.HasPrefix(name, ".") || strings.EqualFold(name, "internal")
}
