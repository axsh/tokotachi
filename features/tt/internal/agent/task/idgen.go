package task

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// generateTaskID generates a new ULID-based task ID with "T-" prefix.
func generateTaskID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return "T-" + id.String(), nil
}

// generateKnowledgeID generates a new ULID-based knowledge atom ID with "K-" prefix.
func generateKnowledgeID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return "K-" + id.String(), nil
}
