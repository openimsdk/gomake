package mageutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pterm/pterm"

	"github.com/openimsdk/gomake/internal/util"
)

const (
	ColorBlue    = "\033[0;34m"
	ColorGreen   = "\033[0;32m"
	ColorRed     = "\033[0;31m"
	ColorYellow  = "\033[33m"
	ColorMagenta = "\033[35m"
	ColorReset   = "\033[0m"
)

const defaultTimeFmt = "[2006-01-02 15:04:05 MST]"

const defaultLogFileName = "gomake.log"

var (
	defaultStdout = os.Stdout
	defaultStderr = os.Stderr

	logFileStateMu sync.Mutex
	logWriteMu     sync.Mutex
	sharedLogFile  *os.File
	sharedLogPath  string
)

func writeConsoleMessage(writer io.Writer, message string) (int, error) {
	if spinner := activeSpinner.Load(); spinner != nil && spinner.enabled && (writer == os.Stdout || writer == os.Stderr) {
		pterm.Fprint(writer, message)
		spinner.Refresh()
		return len(message), nil
	}
	return io.WriteString(writer, message)
}

func logFilePath() (string, error) {
	if Paths == nil {
		return "", fmt.Errorf("paths are not initialized")
	}

	logDir := strings.TrimSpace(Paths.OutputLog)
	if logDir == "" {
		return "", fmt.Errorf("log directory is empty")
	}

	logDir = filepath.Clean(logDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	return filepath.Join(logDir, defaultLogFileName), nil
}

func GetSharedLogFile() (*os.File, error) {
	path, err := logFilePath()
	if err != nil {
		return nil, err
	}

	logFileStateMu.Lock()
	defer logFileStateMu.Unlock()

	if sharedLogFile != nil && sharedLogPath == path {
		return sharedLogFile, nil
	}

	if sharedLogFile != nil {
		_ = sharedLogFile.Close()
		sharedLogFile = nil
		sharedLogPath = ""
	}

	logFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", path, err)
	}

	sharedLogFile = logFile
	sharedLogPath = path
	return sharedLogFile, nil
}

func GetSharedLogFileWithoutError() *os.File {
	logFile, err := GetSharedLogFile()
	if err != nil {
		_, _ = writeConsoleMessage(defaultStdout, err.Error())
		return nil
	}
	return logFile
}

func GetStdoutInnerLogWriter() io.Writer {
	return util.MultiWriter(
		util.WriterFunc(func(p []byte) (n int, err error) { return writeConsoleMessage(defaultStdout, string(p)) }),
		GetSharedLogFileWithoutError(),
	)
}

func GetStderrInnerLogWriter() io.Writer {
	return util.MultiWriter(
		util.WriterFunc(func(p []byte) (n int, err error) { return writeConsoleMessage(defaultStderr, string(p)) }),
		GetSharedLogFileWithoutError(),
	)
}

type PrintOptions struct {
	Writer    io.Writer
	Color     string
	Message   string
	WithTime  bool
	TwoLine   bool
	NoNewLine bool
	TimeFmt   string
}

func Print(opt PrintOptions) error {
	tf := opt.TimeFmt
	if tf == "" {
		tf = defaultTimeFmt
	}

	var (
		err error
	)

	consoleMessage := formatPrintMessage(opt, tf, true)
	fileMessage := formatPrintMessage(opt, tf, false)

	logWriteMu.Lock()
	defer logWriteMu.Unlock()

	if opt.Writer != nil {
		_, err = writeConsoleMessage(opt.Writer, consoleMessage)
	}

	logFile, logErr := GetSharedLogFile()
	if logErr != nil {
		return logErr
	}

	if _, logErr = io.WriteString(logFile, fileMessage); err == nil && logErr != nil {
		err = logErr
	}

	return err
}

func formatPrintMessage(opt PrintOptions, timeFmt string, withColor bool) string {
	var b strings.Builder

	if opt.WithTime {
		ts := time.Now().Format(timeFmt)
		if opt.TwoLine {
			b.WriteString(ts)
			b.WriteByte('\n')
		} else {
			b.WriteString(ts)
			b.WriteByte(' ')
		}
	}

	if withColor && opt.Color != "" {
		b.WriteString(opt.Color)
	}
	b.WriteString(opt.Message)
	if withColor && opt.Color != "" {
		b.WriteString(ColorReset)
	}

	if !opt.NoNewLine {
		b.WriteByte('\n')
	}

	return b.String()
}

func PrintBlue(message string) {
	_ = Print(PrintOptions{Color: ColorBlue, Message: message, WithTime: true, Writer: defaultStdout})
}
func PrintGreen(message string) {
	_ = Print(PrintOptions{Color: ColorGreen, Message: message, WithTime: true, Writer: defaultStdout})
}
func PrintYellow(message string) {
	_ = Print(PrintOptions{Color: ColorYellow, Message: message, WithTime: true, Writer: defaultStdout})
}

func PrintErr(err error) {
	if err == nil {
		return
	}
	_ = Print(PrintOptions{Color: ColorRed, Message: err.Error(), WithTime: true, Writer: defaultStderr})
}
func PrintErrNoTimeStamp(err error) {
	if err == nil {
		return
	}
	_ = Print(PrintOptions{Color: ColorRed, Message: err.Error(), WithTime: false, Writer: defaultStderr})
}

func PrintErrPtr(err *error) {
	PrintErr(*err)
}
