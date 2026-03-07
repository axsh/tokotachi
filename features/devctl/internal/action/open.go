package action

import (
	"github.com/escape-dev/devctl/internal/editor"
)

// Open launches the editor for the given feature.
func (r *Runner) Open(launcher editor.Launcher, opts editor.LaunchOptions) (editor.LaunchResult, error) {
	result, err := launcher.Launch(opts)
	if err != nil {
		return result, err
	}
	if result.Fallback {
		r.Logger.Warn("Used fallback: %s", result.Method)
	} else {
		r.Logger.Info("Editor opened via: %s", result.Method)
	}
	return result, nil
}
