package log_test

import (
	"bytes"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/log"
	"github.com/stretchr/testify/assert"
)

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		logFunc  func(l *log.Logger)
		expected string
		notIn    string
	}{
		{
			name:     "info is always visible",
			verbose:  false,
			logFunc:  func(l *log.Logger) { l.Info("hello") },
			expected: "[INFO]",
		},
		{
			name:    "debug hidden when not verbose",
			verbose: false,
			logFunc: func(l *log.Logger) { l.Debug("detail") },
			notIn:   "[DEBUG]",
		},
		{
			name:     "debug visible when verbose",
			verbose:  true,
			logFunc:  func(l *log.Logger) { l.Debug("detail") },
			expected: "[DEBUG]",
		},
		{
			name:     "warn is always visible",
			verbose:  false,
			logFunc:  func(l *log.Logger) { l.Warn("caution") },
			expected: "[WARN]",
		},
		{
			name:     "error is always visible",
			verbose:  false,
			logFunc:  func(l *log.Logger) { l.Error("failure") },
			expected: "[ERROR]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := log.New(&buf, tt.verbose)
			tt.logFunc(l)
			output := buf.String()
			if tt.expected != "" {
				assert.Contains(t, output, tt.expected)
			}
			if tt.notIn != "" {
				assert.NotContains(t, output, tt.notIn)
			}
		})
	}
}
