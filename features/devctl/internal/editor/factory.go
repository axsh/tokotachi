package editor

import (
	"fmt"

	"github.com/escape-dev/devctl/internal/detect"
)

// NewLauncher returns the appropriate Launcher for the given editor.
func NewLauncher(ed detect.Editor) (Launcher, error) {
	switch ed {
	case detect.EditorVSCode:
		return &VSCode{}, nil
	case detect.EditorCursor:
		return &Cursor{}, nil
	case detect.EditorAG:
		return &AG{}, nil
	case detect.EditorClaude:
		return &Claude{}, nil
	default:
		return nil, fmt.Errorf("no launcher for editor: %s", ed)
	}
}
