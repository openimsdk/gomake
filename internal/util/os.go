package util

import "runtime"

func BinaryWithExtension(stem string) string {
	switch runtime.GOOS {
	case "windows":
		return stem + ".exe"
	default:
		return stem
	}
}
