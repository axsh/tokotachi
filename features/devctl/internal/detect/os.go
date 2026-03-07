package detect

import "runtime"

// OS represents the detected operating system.
type OS string

const (
	OSLinux   OS = "linux"
	OSMacOS   OS = "macos"
	OSWindows OS = "windows"
)

// CurrentOS returns the detected OS for the current platform.
func CurrentOS() OS {
	switch runtime.GOOS {
	case "darwin":
		return OSMacOS
	case "windows":
		return OSWindows
	default:
		return OSLinux
	}
}
