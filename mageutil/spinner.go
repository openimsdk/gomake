package mageutil

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openimsdk/gomake/internal/util"
	"github.com/pterm/pterm"
)

const spinnerDelay = 120 * time.Millisecond

var activeSpinner atomic.Pointer[Spinner]

type Spinner struct {
	printer  *pterm.SpinnerPrinter
	enabled  bool
	stopOnce sync.Once
}

func NewSpinner(message string) *Spinner {
	msg := strings.TrimSpace(message)
	s := &Spinner{
		enabled: util.StderrIsTerminal(),
	}

	if !s.enabled {
		return s
	}

	if inactive := activeSpinner.Swap(nil); inactive != nil {
		inactive.Stop()
	}

	printer, err := pterm.DefaultSpinner.
		WithWriter(os.Stderr).
		WithDelay(spinnerDelay).
		WithRemoveWhenDone(true).
		WithShowTimer(false).
		WithStyle(pterm.NewStyle(pterm.FgMagenta)).
		WithMessageStyle(pterm.NewStyle(pterm.FgMagenta)).
		WithSequence("|", "/", "-", "\\").
		Start(msg)
	if err != nil {
		s.enabled = false
		return s
	}

	s.printer = printer
	activeSpinner.Store(s)
	return s
}

func WithSpinner(message string, fn func()) {
	spinner := NewSpinner(message)
	defer spinner.Stop()
	fn()
}

func WithSpinnerR[R any](message string, fn func() R) R {
	spinner := NewSpinner(message)
	defer spinner.Stop()
	return fn()
}

func (s *Spinner) Stop() {
	s.stopOnce.Do(func() {
		if !s.enabled || s.printer == nil {
			return
		}

		_ = s.printer.Stop()
		time.Sleep(s.printer.Delay)
		clearSpinnerLine()
		activeSpinner.CompareAndSwap(s, nil)
	})
}

func clearSpinnerLine() {
	if pterm.RawOutput || !pterm.Output {
		return
	}
	pterm.Fprinto(os.Stderr, strings.Repeat(" ", pterm.GetTerminalWidth()))
	pterm.Fprinto(os.Stderr)
}

func (s *Spinner) Refresh() {
	if !s.enabled || s.printer == nil || !s.printer.IsActive {
		return
	}
	s.printer.UpdateText(s.printer.Text)
}

func StopSpinner() {
	if sp := activeSpinner.Swap(nil); sp != nil {
		sp.Stop()
	}
}

func RefreshSpinner() {
	if sp := activeSpinner.Load(); sp != nil {
		sp.Refresh()
	}
}
