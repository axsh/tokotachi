package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/axsh/tokotachi/features/tt/internal/agent/storage"
)

// Show retrieves a single IntakeEvent by event_id.
func Show(varDir, eventID string) (*agent.IntakeEvent, error) {
	dbPath := filepath.Join(varDir, "intake", "index.db")
	idx, err := storage.NewIndex(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}
	defer idx.Close()

	// Look up the record in the index
	_, err = idx.GetByEventID(eventID)
	if err != nil {
		return nil, fmt.Errorf("event not found: %s: %w", eventID, err)
	}

	// Search for the JSON file in the intake directory
	intakeDir := filepath.Join(varDir, "intake")
	var eventFile string
	_ = filepath.Walk(intakeDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil
		}
		if info.Name() == eventID+".json" {
			eventFile = path
			return filepath.SkipAll
		}
		return nil
	})

	if eventFile == "" {
		return nil, fmt.Errorf("event file not found for %s", eventID)
	}

	data, err := os.ReadFile(eventFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read event file: %w", err)
	}

	var event agent.IntakeEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event file: %w", err)
	}

	return &event, nil
}

const redactedValue = "<redacted>"

// RedactProvenance replaces provenance fields with <redacted>.
// Does NOT modify the original event. Returns a copy.
func RedactProvenance(event *agent.IntakeEvent) *agent.IntakeEvent {
	copied := *event
	copied.Provenance = agent.Provenance{
		Hostname:       redactedValue,
		User:           redactedValue,
		Cwd:            redactedValue,
		WrapperVersion: event.Provenance.WrapperVersion,
	}
	return &copied
}
