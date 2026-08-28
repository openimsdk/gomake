//go:build windows

package rlimit

func SetMaxOpenFiles(num int) error {
	return nil
}
