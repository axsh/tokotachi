package log

import (
	"fmt"
	"io"
)

// Level represents log severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger provides leveled logging output.
type Logger struct {
	out     io.Writer
	verbose bool
}

// New creates a Logger. If verbose is true, DEBUG messages are shown.
func New(out io.Writer, verbose bool) *Logger {
	return &Logger{out: out, verbose: verbose}
}

func (l *Logger) log(level Level, prefix, format string, args ...any) {
	if level == LevelDebug && !l.verbose {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s %s\n", prefix, msg)
}

// Info logs an informational message.
func (l *Logger) Info(format string, args ...any) { l.log(LevelInfo, "[INFO]", format, args...) }

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...any) { l.log(LevelWarn, "[WARN]", format, args...) }

// Error logs an error message.
func (l *Logger) Error(format string, args ...any) { l.log(LevelError, "[ERROR]", format, args...) }

// Debug logs a debug message (only visible when verbose is true).
func (l *Logger) Debug(format string, args ...any) { l.log(LevelDebug, "[DEBUG]", format, args...) }
