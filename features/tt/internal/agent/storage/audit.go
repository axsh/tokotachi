package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
)

// AuditLog appends audit entries as NDJSON.
type AuditLog struct {
	logDir string
}

// NewAuditLog creates a new AuditLog.
func NewAuditLog(logDir string) *AuditLog {
	return &AuditLog{logDir: logDir}
}

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	EventID   string `json:"event_id"`
	Agent     string `json:"agent"`
	Status    string `json:"status"`
	Code      string `json:"code"`
	Summary   string `json:"task_summary,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Append appends an audit entry to the appropriate log file.
// Success/warnings go to agent-notify.ndjson, failures to agent-notify-error.ndjson.
func (al *AuditLog) Append(event *agent.IntakeEvent, result *agent.NotifyResult) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		EventID:   result.EventID,
		Agent:     event.Agent,
		Status:    result.Status,
		Code:      result.Code,
		Summary:   event.TaskSummary,
		Message:   result.Message,
	}

	var logFile string
	if result.Status == "rejected" {
		logFile = filepath.Join(al.logDir, "agent-notify-error.ndjson")
	} else {
		logFile = filepath.Join(al.logDir, "agent-notify.ndjson")
	}

	return al.appendEntry(logFile, entry)
}

// appendEntry marshals and appends an entry to the specified file.
func (al *AuditLog) appendEntry(filePath string, entry AuditEntry) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}
	return nil
}
