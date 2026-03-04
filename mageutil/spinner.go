package mageutil

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openimsdk/gomake/internal/util"
)

const (
	ESCEraseLine = "\033[2K"
)

var (
	activeSpinner         atomic.Pointer[Spinner]
	spinnerFrames         = []string{"|", "/", "-", "\\"}
	spinnerRenderInterval = 120 * time.Millisecond
	globalPauseDepth      atomic.Int32
)

type Spinner struct {
	enabled  bool
	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}

	message atomic.Value

	start time.Time
}

func NewSpinner(message string) *Spinner {
	msg := strings.TrimSpace(message)

	s := &Spinner{
		enabled: util.StdoutIsTerminal(),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		start:   time.Now(),
	}
	s.message.Store(msg)

	if !s.enabled {
		close(s.doneCh)
		return s
	}

	inactive := activeSpinner.Swap(s)
	if inactive != nil {
		inactive.Stop()
	}
	go s.run()
	return s
}

func WithSpinner(message string, fn func()) {
	spinner := NewSpinner(message)
	defer spinner.Stop()
	fn()
}

func WithSpinnerE(message string, fn func() error) error {
	spinner := NewSpinner(message)
	defer spinner.Stop()
	return fn()
}

func (s *Spinner) Stop() {
	s.stopOnce.Do(func() {
		if !s.enabled {
			return
		}

		close(s.stopCh)
		<-s.doneCh
		if activeSpinner.CompareAndSwap(s, nil) {
			fmt.Printf("\r%s", ESCEraseLine)
		}
	})
}

func (s *Spinner) run() {
	defer close(s.doneCh)

	if globalPauseDepth.Load() == 0 {
		s.render()
	}

	timer := time.NewTimer(spinnerRenderInterval)
	defer timer.Stop()

	for {
		elapsed := time.Since(s.start)
		rem := elapsed % spinnerRenderInterval
		wait := spinnerRenderInterval - rem
		if wait <= 0 {
			wait = spinnerRenderInterval
		}
		timer.Reset(wait)

		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			if globalPauseDepth.Load() > 0 {
				continue
			}
			s.render()
		}
	}
}

func (s *Spinner) render() {
	elapsed := time.Since(s.start)
	step := int(elapsed / spinnerRenderInterval)
	frame := spinnerFrames[step%len(spinnerFrames)]
	msg := s.message.Load().(string)

	fmt.Printf("\r%s%s %s%s", ColorMagenta, frame, msg, ColorReset)
}

func StopSpinner() {
	if sp := activeSpinner.Swap(nil); sp != nil {
		sp.Stop()
	}
}

func PauseSpinner() {
	if sp := activeSpinner.Load(); sp == nil || !sp.enabled {
		return
	}
	globalPauseDepth.Add(1)
	fmt.Printf("\r%s", ESCEraseLine)
}

func ResumeSpinner() {
	if sp := activeSpinner.Load(); sp == nil || !sp.enabled {
		return
	}

	for {
		d := globalPauseDepth.Load()
		if d <= 0 {
			return
		}
		if globalPauseDepth.CompareAndSwap(d, d-1) {
			if d-1 == 0 {
				if sp := activeSpinner.Load(); sp != nil {
					sp.render()
				}
			}
			return
		}
	}
}

func WithActiveSpinnerPaused(fn func()) {
	PauseSpinner()
	defer ResumeSpinner()
	fn()
}
