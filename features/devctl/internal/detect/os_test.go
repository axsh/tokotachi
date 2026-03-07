package detect_test

import (
	"runtime"
	"testing"

	"github.com/escape-dev/devctl/internal/detect"
	"github.com/stretchr/testify/assert"
)

func TestDetectOS(t *testing.T) {
	got := detect.CurrentOS()
	switch runtime.GOOS {
	case "linux":
		assert.Equal(t, detect.OSLinux, got)
	case "darwin":
		assert.Equal(t, detect.OSMacOS, got)
	case "windows":
		assert.Equal(t, detect.OSWindows, got)
	default:
		t.Fatalf("unexpected GOOS: %s", runtime.GOOS)
	}
}
