package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
)

// StatusReport holds the status summary.
type StatusReport struct {
	PendingCount     int    `json:"pending_count"`
	ProcessedCount   int    `json:"processed_count"`
	FailedCount      int    `json:"failed_count"`
	IgnoredCount     int    `json:"ignored_count"`
	OldestPendingAge string `json:"oldest_pending_age,omitempty"`
	IndexHealth      string `json:"index_health"`
	CurrentBranch    string `json:"current_branch,omitempty"`
}

// GetStatus computes the status report.
func GetStatus(varDir string) (*StatusReport, error) {
	intakeDir := filepath.Join(varDir, "intake")

	report := &StatusReport{
		IndexHealth: "unavailable",
	}

	report.PendingCount = countJSONFiles(filepath.Join(intakeDir, "pending"))
	report.ProcessedCount = countJSONFiles(filepath.Join(intakeDir, "processed"))
	report.FailedCount = countJSONFiles(filepath.Join(intakeDir, "failed"))
	report.IgnoredCount = countJSONFiles(filepath.Join(intakeDir, "ignored"))

	// Oldest pending age
	oldest := findOldestPendingAge(filepath.Join(intakeDir, "pending"))
	if oldest != "" {
		report.OldestPendingAge = oldest
	}

	// Index health
	dbPath := filepath.Join(intakeDir, "index.db")
	report.IndexHealth = checkIndexHealth(dbPath)

	return report, nil
}

// countJSONFiles counts .json files recursively under the given directory.
func countJSONFiles(dir string) int {
	count := 0
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			count++
		}
		return nil
	})
	return count
}

// findOldestPendingAge reads the oldest JSON file in pending/ and returns a human-readable age.
func findOldestPendingAge(pendingDir string) string {
	var oldest time.Time
	_ = filepath.Walk(pendingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var event agent.IntakeEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil
		}
		if oldest.IsZero() || event.Timestamps.CreatedAt.Before(oldest) {
			oldest = event.Timestamps.CreatedAt
		}
		return nil
	})
	if oldest.IsZero() {
		return ""
	}
	age := time.Since(oldest)
	return formatDuration(age)
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// checkIndexHealth checks if the SQLite index is healthy.
func checkIndexHealth(dbPath string) string {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return "unavailable"
	}
	return "ok"
}
