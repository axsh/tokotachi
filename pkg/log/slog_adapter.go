package log

import (
	"fmt"
	"log/slog"
)

// SlogAdapter adapts *slog.Logger to the Logger interface.
// This allows external projects using log/slog to inject their
// existing slog.Logger into tokotachi.
type SlogAdapter struct {
	SlogLogger *slog.Logger
}

// NewSlogAdapter creates a Logger from a *slog.Logger.
func NewSlogAdapter(l *slog.Logger) *SlogAdapter {
	return &SlogAdapter{SlogLogger: l}
}

// Info logs at INFO level via slog.
func (a *SlogAdapter) Info(format string, args ...any) {
	a.SlogLogger.Info(fmt.Sprintf(format, args...))
}

// Warn logs at WARN level via slog.
func (a *SlogAdapter) Warn(format string, args ...any) {
	a.SlogLogger.Warn(fmt.Sprintf(format, args...))
}

// Error logs at ERROR level via slog.
func (a *SlogAdapter) Error(format string, args ...any) {
	a.SlogLogger.Error(fmt.Sprintf(format, args...))
}

// Debug logs at DEBUG level via slog.
func (a *SlogAdapter) Debug(format string, args ...any) {
	a.SlogLogger.Debug(fmt.Sprintf(format, args...))
}

// Log logs at the specified Level via the corresponding slog method.
func (a *SlogAdapter) Log(level Level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	switch level {
	case LevelDebug:
		a.SlogLogger.Debug(msg)
	case LevelInfo:
		a.SlogLogger.Info(msg)
	case LevelWarn:
		a.SlogLogger.Warn(msg)
	case LevelError:
		a.SlogLogger.Error(msg)
	default:
		a.SlogLogger.Info(msg)
	}
}
