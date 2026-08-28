package util

import "runtime"

func BinaryWithRuntimeExtension(stem string) string {
	switch runtime.GOOS {
	case "windows":
		return stem + ".exe"
	default:
		return stem
	}
}
