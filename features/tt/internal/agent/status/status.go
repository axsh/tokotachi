package status

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	_ "modernc.org/sqlite"
)

// StatusCounts holds event counts by status.
type StatusCounts struct {
	Pending   int `json:"pending"`
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
	Ignored   int `json:"ignored"`
}

// StatusReport holds the status summary.
type StatusReport struct {
	MemoryRoot          string       `json:"memory_root"`
	CurrentBranch       string       `json:"current_branch"`
	Counts              StatusCounts `json:"counts"`
	CurrentBranchCounts *StatusCounts `json:"current_branch_counts,omitempty"`
	OldestPending       string       `json:"oldest_pending,omitempty"`
	IndexHealth         string       `json:"index_health"`
}

// GetStatus computes the status report.
// memoryRoot: "prompts/memory" (for display)
// varDir: "prompts/memory/var" (for file counting)
// currentBranch: from git (caller provides)
func GetStatus(memoryRoot, varDir, currentBranch string) (*StatusReport, error) {
	intakeDir := filepath.Join(varDir, "intake")

	report := &StatusReport{
		MemoryRoot:    memoryRoot,
		CurrentBranch: currentBranch,
	}

	// Count files by status
	report.Counts = StatusCounts{
		Pending:   countJSONFiles(filepath.Join(intakeDir, "pending")),
		Processed: countJSONFiles(filepath.Join(intakeDir, "processed")),
		Failed:    countJSONFiles(filepath.Join(intakeDir, "failed")),
		Ignored:   countJSONFiles(filepath.Join(intakeDir, "ignored")),
	}

	// Oldest pending timestamp (ISO8601 seconds precision)
	report.OldestPending = findOldestPendingTimestamp(filepath.Join(intakeDir, "pending"))

	// Index health
	dbPath := filepath.Join(intakeDir, "index.db")
	report.IndexHealth = checkIndexHealth(dbPath)

	// Branch-specific counts from index
	if currentBranch != "" && report.IndexHealth == "ok" {
		branchCounts := queryBranchCounts(dbPath, currentBranch)
		if branchCounts != nil {
			report.CurrentBranchCounts = branchCounts
		}
	}

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

// findOldestPendingTimestamp reads all pending events and returns
// the oldest created_at as ISO8601 seconds precision string.
func findOldestPendingTimestamp(pendingDir string) string {
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
	return oldest.UTC().Format("2006-01-02T15:04:05Z")
}

// checkIndexHealth checks if the SQLite index is healthy.
// Returns "ok", "missing", or "error".
func checkIndexHealth(dbPath string) string {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return "missing"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "error"
	}
	defer db.Close()

	var result string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil || result != "ok" {
		return "error"
	}

	return "ok"
}

// queryBranchCounts queries the index for counts specific to a branch.
func queryBranchCounts(dbPath, branch string) *StatusCounts {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	counts := &StatusCounts{}
	rows, err := db.Query(
		"SELECT status, COUNT(*) FROM intake_events WHERE branch = ? GROUP BY status",
		branch,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var s string
		var c int
		if err := rows.Scan(&s, &c); err != nil {
			continue
		}
		switch s {
		case "pending":
			counts.Pending = c
		case "processed":
			counts.Processed = c
		case "failed":
			counts.Failed = c
		case "ignored":
			counts.Ignored = c
		}
	}

	return counts
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
