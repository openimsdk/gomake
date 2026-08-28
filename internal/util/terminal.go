package util

import "os"

func StdoutIsTerminal() bool {
	return fileIsTerminal(os.Stdout)
}

func StderrIsTerminal() bool {
	return fileIsTerminal(os.Stderr)
}

func fileIsTerminal(file *os.File) bool {
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
