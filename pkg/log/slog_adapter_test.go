package log_test

import (
	"bytes"
	"log/slog"
	"testing"

	pkglog "github.com/axsh/tokotachi/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestSlogAdapter_Info(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := pkglog.NewSlogAdapter(l)

	adapter.Info("hello %s", "world")
	assert.Contains(t, buf.String(), "hello world")
	assert.Contains(t, buf.String(), "INFO")
}

func TestSlogAdapter_Warn(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := pkglog.NewSlogAdapter(l)

	adapter.Warn("caution %d", 42)
	assert.Contains(t, buf.String(), "caution 42")
	assert.Contains(t, buf.String(), "WARN")
}

func TestSlogAdapter_Error(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := pkglog.NewSlogAdapter(l)

	adapter.Error("failure: %v", "timeout")
	assert.Contains(t, buf.String(), "failure: timeout")
	assert.Contains(t, buf.String(), "ERROR")
}

func TestSlogAdapter_Debug(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := pkglog.NewSlogAdapter(l)

	adapter.Debug("detail %s", "trace")
	assert.Contains(t, buf.String(), "detail trace")
	assert.Contains(t, buf.String(), "DEBUG")
}

func TestSlogAdapter_Log(t *testing.T) {
	tests := []struct {
		name     string
		level    pkglog.Level
		expected string
	}{
		{name: "log at debug", level: pkglog.LevelDebug, expected: "DEBUG"},
		{name: "log at info", level: pkglog.LevelInfo, expected: "INFO"},
		{name: "log at warn", level: pkglog.LevelWarn, expected: "WARN"},
		{name: "log at error", level: pkglog.LevelError, expected: "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
			adapter := pkglog.NewSlogAdapter(l)

			adapter.Log(tt.level, "msg %d", 1)
			assert.Contains(t, buf.String(), "msg 1")
			assert.Contains(t, buf.String(), tt.expected)
		})
	}
}

func TestSlogAdapter_ImplementsLogger(t *testing.T) {
	var _ pkglog.Logger = &pkglog.SlogAdapter{}
}
