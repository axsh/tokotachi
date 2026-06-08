package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	_ "modernc.org/sqlite"
)

// Index manages the SQLite index for intake events.
type Index struct {
	db *sql.DB
}

// NewIndex opens or creates the SQLite database at the given path.
// Enables WAL mode and creates tables if not exist.
func NewIndex(dbPath string) (*Index, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode and set busy timeout
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", p, err)
		}
	}

	// Create main table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS intake_events (
			event_id           TEXT PRIMARY KEY,
			content_hash       TEXT NOT NULL,
			content_id         TEXT NOT NULL,
			agent              TEXT NOT NULL,
			branch             TEXT DEFAULT '',
			scope              TEXT NOT NULL DEFAULT 'session',
			branch_package     TEXT DEFAULT '',
			status             TEXT NOT NULL DEFAULT 'pending',
			client_request_id  TEXT UNIQUE,
			task_summary       TEXT NOT NULL,
			stored_at          TEXT NOT NULL,
			created_at         TEXT NOT NULL
		)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create intake_events table: %w", err)
	}

	// Create FTS table
	if _, err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS intake_events_fts
		USING fts5(event_id, task_summary, raw_notes_text)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create FTS table: %w", err)
	}

	return &Index{db: db}, nil
}

// Store inserts event metadata into the index.
// If client_request_id already exists, returns the existing event_id (idempotency).
// Returns (existingEventID, error). existingEventID is non-empty only if idempotent hit.
func (idx *Index) Store(event *agent.IntakeEvent) (string, error) {
	// Check idempotency first
	if event.ClientRequestID != "" {
		var existingID string
		err := idx.db.QueryRow(
			"SELECT event_id FROM intake_events WHERE client_request_id = ?",
			event.ClientRequestID,
		).Scan(&existingID)
		if err == nil {
			return existingID, nil
		}
		if err != sql.ErrNoRows {
			return "", fmt.Errorf("failed to check idempotency: %w", err)
		}
	}

	// Insert into main table
	branch := ""
	if event.Git != nil {
		branch = event.Git.Branch
	}

	clientReqID := sql.NullString{
		String: event.ClientRequestID,
		Valid:  event.ClientRequestID != "",
	}

	bpKey := branchPackageKey(event.BranchPackage)

	_, err := idx.db.Exec(`
		INSERT INTO intake_events (
			event_id, content_hash, content_id, agent, branch, scope,
			branch_package, status, client_request_id, task_summary,
			stored_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?)
	`,
		event.EventID,
		event.ContentHash,
		event.ContentID,
		event.Agent,
		branch,
		event.Scope,
		bpKey,
		clientReqID,
		event.TaskSummary,
		event.Timestamps.StoredAt.Format("2006-01-02T15:04:05Z"),
		event.Timestamps.CreatedAt.Format("2006-01-02T15:04:05Z"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert event: %w", err)
	}

	// Insert into FTS table
	notesText := strings.Join(event.RawNotes, " ")
	_, err = idx.db.Exec(`
		INSERT INTO intake_events_fts (event_id, task_summary, raw_notes_text)
		VALUES (?, ?, ?)
	`, event.EventID, event.TaskSummary, notesText)
	if err != nil {
		return "", fmt.Errorf("failed to insert FTS entry: %w", err)
	}

	return "", nil
}

// GetByEventID retrieves event metadata by event_id.
func (idx *Index) GetByEventID(eventID string) (*EventRecord, error) {
	var r EventRecord
	err := idx.db.QueryRow(`
		SELECT event_id, content_hash, content_id, agent, branch, scope,
		       branch_package, status, client_request_id, task_summary,
		       stored_at, created_at
		FROM intake_events WHERE event_id = ?
	`, eventID).Scan(
		&r.EventID, &r.ContentHash, &r.ContentID, &r.Agent, &r.Branch,
		&r.Scope, &r.BranchPackage, &r.Status, &r.ClientRequestID,
		&r.TaskSummary, &r.StoredAt, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// SearchFTS performs a full-text search on task_summary and raw_notes.
func (idx *Index) SearchFTS(query string) ([]string, error) {
	rows, err := idx.db.Query(
		"SELECT event_id FROM intake_events_fts WHERE intake_events_fts MATCH ?",
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("FTS search failed: %w", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		results = append(results, id)
	}
	return results, rows.Err()
}

// Close closes the database connection.
func (idx *Index) Close() error {
	return idx.db.Close()
}

// branchPackageKey extracts the key string from a BranchPackageInfo.
func branchPackageKey(bp *agent.BranchPackageInfo) string {
	if bp == nil {
		return ""
	}
	return bp.Key
}

// EventRecord represents a row from intake_events table.
type EventRecord struct {
	EventID         string
	ContentHash     string
	ContentID       string
	Agent           string
	Branch          string
	Scope           string
	BranchPackage   string
	Status          string
	ClientRequestID sql.NullString
	TaskSummary     string
	StoredAt        string
	CreatedAt       string
}

// UpdateStatus updates the status of an event in the index.
func (idx *Index) UpdateStatus(eventID, newStatus string) error {
	result, err := idx.db.Exec(
		"UPDATE intake_events SET status = ? WHERE event_id = ?",
		newStatus, eventID,
	)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("event %s not found", eventID)
	}
	return nil
}

// ListPendingByBranch returns all pending events for a given branch.
func (idx *Index) ListPendingByBranch(branch string) ([]EventRecord, error) {
	rows, err := idx.db.Query(`
		SELECT event_id, content_hash, content_id, agent, branch, scope,
		       branch_package, status, client_request_id, task_summary,
		       stored_at, created_at
		FROM intake_events
		WHERE status = 'pending' AND branch = ?
		ORDER BY created_at ASC
	`, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending events: %w", err)
	}
	defer rows.Close()

	var records []EventRecord
	for rows.Next() {
		var r EventRecord
		if err := rows.Scan(
			&r.EventID, &r.ContentHash, &r.ContentID, &r.Agent, &r.Branch,
			&r.Scope, &r.BranchPackage, &r.Status, &r.ClientRequestID,
			&r.TaskSummary, &r.StoredAt, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

