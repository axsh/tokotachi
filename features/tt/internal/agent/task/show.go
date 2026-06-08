package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
)

// Show reads and returns the task with the given ID.
// Searches pending/, then completed/, then failed/.
func Show(varDir string, taskID string) (*agent.AgentTask, error) {
	for _, subdir := range []string{"pending", "completed", "failed"} {
		path := filepath.Join(varDir, "tasks", subdir, taskID+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var task agent.AgentTask
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, fmt.Errorf("failed to parse task file %s: %w", path, err)
		}
		return &task, nil
	}
	return nil, fmt.Errorf("task %s not found in pending, completed, or failed", taskID)
}
