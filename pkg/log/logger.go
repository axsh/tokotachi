// Package log provides a public Logger interface for dependency injection.
//
// External projects can implement this interface to inject custom loggers
// (e.g., log/slog, zerolog, zap) when using the tokotachi library.
// The default implementation is provided by internal/log.
package log

// Level represents log severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger abstracts logging output.
// External projects can inject their own Logger implementations.
type Logger interface {
	// Info logs an informational message.
	Info(format string, args ...any)
	// Warn logs a warning message.
	Warn(format string, args ...any)
	// Error logs an error message.
	Error(format string, args ...any)
	// Debug logs a debug message.
	Debug(format string, args ...any)
	// Log logs a message at the specified level with an auto-generated prefix.
	Log(level Level, format string, args ...any)
}
