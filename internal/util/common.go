package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/disk"
)

func CoalescePtr[T any](values ...*T) *T {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func NilAsZero[T any](opt *T) T {
	var zero T
	if opt == nil {
		return zero
	}
	return *opt
}

func Clamp(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	value := float64(bytes) / float64(div)
	suffixes := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	if exp >= len(suffixes) {
		exp = len(suffixes) - 1
	}
	return fmt.Sprintf("%.2f%s", value, suffixes[exp])
}

type TempStorageInfo struct {
	InMemory      bool
	AvailableDisk uint64
}

func ResolveTempStorageInfo(tempRoot string) TempStorageInfo {
	info := TempStorageInfo{InMemory: true}

	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}

	tempRoot = normalizePath(tempRoot)
	if tempRoot == "" {
		return info
	}

	parts, err := disk.Partitions(true)
	if err != nil {
		return info
	}

	part, ok := findMountpoint(tempRoot, parts)
	if !ok {
		return info
	}

	fsType := strings.ToLower(part.Fstype)
	if isMemoryFSType(fsType) {
		info.InMemory = true
		return info
	}

	usage, err := disk.Usage(tempRoot)
	if err != nil {
		return info
	}

	info.InMemory = false
	info.AvailableDisk = usage.Free
	return info
}

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" && strings.HasSuffix(cleaned, ":") {
		return cleaned
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return cleaned
	}
	return abs
}

func findMountpoint(path string, parts []disk.PartitionStat) (disk.PartitionStat, bool) {
	var best disk.PartitionStat
	bestLen := -1
	for _, part := range parts {
		mount := normalizePath(part.Mountpoint)
		if mount == "" {
			continue
		}
		if pathHasPrefix(path, mount) {
			if len(mount) > bestLen {
				best = part
				bestLen = len(mount)
			}
		}
	}
	if bestLen == -1 {
		return disk.PartitionStat{}, false
	}
	return best, true
}

func pathHasPrefix(path string, mount string) bool {
	if path == "" || mount == "" {
		return false
	}
	if runtime.GOOS == "windows" {
		pathLower := strings.ToLower(path)
		mountLower := strings.ToLower(mount)
		if strings.HasSuffix(mountLower, ":") {
			return strings.HasPrefix(pathLower, mountLower)
		}
		if !strings.HasSuffix(mountLower, "\\") {
			mountLower += "\\"
		}
		return strings.HasPrefix(pathLower, mountLower)
	}
	if path == mount {
		return true
	}
	if !strings.HasSuffix(mount, string(os.PathSeparator)) {
		mount += string(os.PathSeparator)
	}
	return strings.HasPrefix(path, mount)
}

func isMemoryFSType(fsType string) bool {
	switch strings.ToLower(fsType) {
	case "tmpfs", "ramfs", "devtmpfs", "mfs":
		return true
	default:
		return false
	}
}
