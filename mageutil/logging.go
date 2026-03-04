package mageutil

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
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

type PrintOptions struct {
	Writer    io.Writer
	Color     string
	Message   string
	WithTime  bool
	TwoLine   bool
	NoNewLine bool
	TimeFmt   string
}

func Print(opt PrintOptions) (int, error) {
	w := opt.Writer
	if w == nil {
		w = os.Stdout
	}
	tf := opt.TimeFmt
	if tf == "" {
		tf = defaultTimeFmt
	}

	var (
		n   int
		err error
	)

	WithActiveSpinnerPaused(func() {
		var b strings.Builder

		if opt.WithTime {
			ts := time.Now().Format(tf)
			if opt.TwoLine {
				b.WriteString(ts)
				b.WriteByte('\n')
			} else {
				b.WriteString(ts)
				b.WriteByte(' ')
			}
		}

		if opt.Color != "" {
			b.WriteString(opt.Color)
		}
		b.WriteString(opt.Message)
		if opt.Color != "" {
			b.WriteString(ColorReset)
		}

		if !opt.NoNewLine {
			b.WriteByte('\n')
		}

		nLocal, errLocal := io.WriteString(w, b.String())
		n, err = nLocal, errLocal
	})

	return n, err
}

func PrintBlue(message string) error {
	_, err := Print(PrintOptions{Color: ColorBlue, Message: message, WithTime: true})
	return err
}
func PrintGreen(message string) error {
	_, err := Print(PrintOptions{Color: ColorGreen, Message: message, WithTime: true})
	return err
}
func PrintRed(message string) error {
	_, err := Print(PrintOptions{Color: ColorRed, Message: message, WithTime: true})
	return err
}
func PrintYellow(message string) error {
	_, err := Print(PrintOptions{Color: ColorYellow, Message: message, WithTime: true})
	return err
}

func PrintRedNoTimeStamp(message string) error {
	_, err := Print(PrintOptions{Color: ColorRed, Message: message, WithTime: false})
	return err
}

func PrintBlueTwoLine(message string) error {
	_, err := Print(PrintOptions{Color: ColorBlue, Message: message, WithTime: true, TwoLine: true})
	return err
}

func PrintGreenTwoLine(message string) error {
	_, err := Print(PrintOptions{Color: ColorGreen, Message: message, WithTime: true, TwoLine: true})
	return err
}

func PrintGreenNoTimeStamp(message string) error {
	_, err := Print(PrintOptions{Color: ColorGreen, Message: message, WithTime: false})
	return err
}

func PrintRedToStdErr(a ...any) (int, error) {
	return Print(PrintOptions{
		Writer:    os.Stderr,
		Color:     ColorRed,
		Message:   fmt.Sprint(a...),
		WithTime:  false,
		NoNewLine: true,
	})
}

func PrintGreenToStdOut(a ...any) (int, error) {
	return Print(PrintOptions{
		Writer:    os.Stdout,
		Color:     ColorGreen,
		Message:   fmt.Sprint(a...),
		WithTime:  false,
		NoNewLine: true,
	})
}
