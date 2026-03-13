package scaffold

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Spinner displays a loading animation in the terminal.
type Spinner struct {
	writer  io.Writer
	message string
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
}

// Braille spinner frames
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const spinnerInterval = 100 * time.Millisecond

// NewSpinner creates a new Spinner. If writer is nil, output is discarded.
func NewSpinner(w io.Writer) *Spinner {
	return &Spinner{
		writer: w,
	}
}

// Start begins the spinner animation with the given message.
func (s *Spinner) Start(message string) {
	s.mu.Lock()
	s.message = message
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.spin()
}

// UpdateMessage changes the spinner message while it's running.
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Stop stops the spinner animation.
func (s *Spinner) Stop() {
	s.mu.Lock()
	stopCh := s.stop
	doneCh := s.done
	s.mu.Unlock()

	if stopCh == nil {
		return
	}

	close(stopCh)
	<-doneCh

	// Clear the spinner line
	if s.writer != nil {
		fmt.Fprintf(s.writer, "\r%s\r", "                                        ")
	}
}

// spin is the internal goroutine that renders the spinner.
func (s *Spinner) spin() {
	defer close(s.done)

	i := 0
	ticker := time.NewTicker(spinnerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			if s.writer != nil {
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()
				frame := spinnerFrames[i%len(spinnerFrames)]
				fmt.Fprintf(s.writer, "\r%s %s", frame, msg)
			}
			i++
		}
	}
}
