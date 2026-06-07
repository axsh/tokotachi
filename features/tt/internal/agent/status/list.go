package status

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// ListOptions holds filter options for listing events.
type ListOptions struct {
	Status     string // "pending", "processed", "failed", "ignored"
	Agent      string
	Branch     string
	Query      string // FTS query
	PathPrefix string
	From       string // ISO8601
	To         string // ISO8601
	Format     string // "table" or "json"
	Limit      int
}

// ListItem represents a single event in list output.
type ListItem struct {
	EventID     string `json:"event_id"`
	Agent       string `json:"agent"`
	Branch      string `json:"branch"`
	Status      string `json:"status"`
	TaskSummary string `json:"task_summary"`
	CreatedAt   string `json:"created_at"`
}

// List retrieves events matching the filter criteria.
func List(varDir string, opts ListOptions) ([]ListItem, error) {
	dbPath := filepath.Join(varDir, "intake", "index.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}
	defer db.Close()

	// Build query
	var conditions []string
	var args []any

	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.Agent != "" {
		conditions = append(conditions, "agent = ?")
		args = append(args, opts.Agent)
	}
	if opts.Branch != "" {
		conditions = append(conditions, "branch = ?")
		args = append(args, opts.Branch)
	}
	if opts.From != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, opts.From)
	}
	if opts.To != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, opts.To)
	}

	// FTS query
	if opts.Query != "" {
		conditions = append(conditions, "event_id IN (SELECT event_id FROM intake_events_fts WHERE intake_events_fts MATCH ?)")
		args = append(args, opts.Query)
	}

	query := "SELECT event_id, agent, branch, status, task_summary, created_at FROM intake_events"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var items []ListItem
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(&item.EventID, &item.Agent, &item.Branch, &item.Status, &item.TaskSummary, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
