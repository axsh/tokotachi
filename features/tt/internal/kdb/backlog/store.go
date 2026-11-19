package backlog

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no sqlite3 CLI, no cgo)
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the embedded schema.sql text.
func Schema() string { return schemaSQL }

// Store is the SQLite cache of backlog items.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens (creating if needed) the SQLite cache at dbPath and applies the
// embedded schema.
func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, bizErr("create database directory: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, bizErr("open database %s: %v", dbPath, err)
	}
	// One connection keeps transactions simple and avoids lock contention.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, bizErr("configure database: %v", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, bizErr("failed to apply schema.sql: %v", err)
	}
	return &Store{db: db, path: dbPath}, nil
}

// Path returns the database file path.
func (s *Store) Path() string { return s.path }

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

var (
	colList      = strings.Join(Columns, ", ")
	placeholders = strings.TrimSuffix(strings.Repeat("?, ", len(Columns)), ", ")
)

func itemArgs(it Item) []any {
	vals := itemValues(it)
	args := make([]any, len(vals))
	for i, v := range vals {
		args[i] = v
	}
	if it.StateDate == nil {
		args[6] = nil // state_date is the 7th column
	}
	return args
}

type rowScanner interface{ Scan(dest ...any) error }

func scanItem(r rowScanner) (Item, error) {
	var it Item
	var stateDate sql.NullString
	err := r.Scan(&it.Phase, &it.Branch, &it.LocalID, &it.Slug, &it.Title, &it.State, &stateDate,
		&it.Kind, &it.AskTo, &it.Due, &it.Origin, &it.Decision, &it.Problem, &it.CurrentHandling,
		&it.Options, &it.Questions, &it.AfterDecision, &it.History, &it.CreatedAt, &it.UpdatedAt, &it.UUID)
	if err != nil {
		return Item{}, err
	}
	if stateDate.Valid {
		v := stateDate.String
		it.StateDate = &v
	}
	return it, nil
}

// Insert adds a new item; a key or uuid clash yields a UNIQUE constraint error.
func (s *Store) Insert(it Item) error {
	_, err := s.db.Exec("INSERT INTO items ("+colList+") VALUES ("+placeholders+")", itemArgs(it)...)
	return err
}

// Upsert inserts or replaces by primary key (INSERT OR REPLACE).
func (s *Store) Upsert(it Item) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO items ("+colList+") VALUES ("+placeholders+")", itemArgs(it)...)
	return err
}

// Update rewrites every non-key column of the row identified by its logical key.
func (s *Store) Update(it Item) error {
	args := itemArgs(it)
	set := make([]string, 0, len(Columns))
	setArgs := make([]any, 0, len(Columns)+3)
	for i, c := range Columns {
		if c == "phase" || c == "branch" || c == "local_id" {
			continue
		}
		set = append(set, c+" = ?")
		setArgs = append(setArgs, args[i])
	}
	setArgs = append(setArgs, it.Phase, it.Branch, it.LocalID)
	res, err := s.db.Exec("UPDATE items SET "+strings.Join(set, ", ")+" WHERE phase = ? AND branch = ? AND local_id = ?", setArgs...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetByKey returns the item with the logical key, if any.
func (s *Store) GetByKey(phase, branch, id string) (Item, bool, error) {
	row := s.db.QueryRow("SELECT "+colList+" FROM items WHERE phase = ? AND branch = ? AND local_id = ?", phase, branch, id)
	it, err := scanItem(row)
	if err == sql.ErrNoRows {
		return Item{}, false, nil
	}
	return it, err == nil, err
}

// GetByUUID returns the item with the uuid, if any.
func (s *Store) GetByUUID(uuid string) (Item, bool, error) {
	row := s.db.QueryRow("SELECT "+colList+" FROM items WHERE uuid = ?", uuid)
	it, err := scanItem(row)
	if err == sql.ErrNoRows {
		return Item{}, false, nil
	}
	return it, err == nil, err
}

// ListAll returns every item ordered by phase, branch, local_id.
func (s *Store) ListAll() ([]Item, error) {
	rows, err := s.db.Query("SELECT " + colList + " FROM items ORDER BY phase, branch, local_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// Count returns the number of rows.
func (s *Store) Count() (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&n)
	return n, err
}

// NextLocalID returns the next free zero-padded (%03d) id within phase/branch.
func (s *Store) NextLocalID(phase, branch string) (string, error) {
	var n int
	err := s.db.QueryRow(
		"SELECT COALESCE(MAX(CAST(local_id AS INTEGER)), -1) + 1 FROM items WHERE phase = ? AND branch = ?",
		phase, branch).Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%03d", n), nil
}

// ReplaceAll deletes every row and inserts items in one transaction; on any
// error the transaction is rolled back and the cache is unchanged.
func (s *Store) ReplaceAll(items []Item) error {
	return s.inTx(true, items)
}

// UpsertAll inserts or replaces items in one transaction.
func (s *Store) UpsertAll(items []Item) error {
	return s.inTx(false, items)
}

func (s *Store) inTx(deleteFirst bool, items []Item) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit
	if deleteFirst {
		if _, err := tx.Exec("DELETE FROM items"); err != nil {
			return err
		}
	}
	verb := "INSERT"
	if !deleteFirst {
		verb = "INSERT OR REPLACE"
	}
	stmt, err := tx.Prepare(verb + " INTO items (" + colList + ") VALUES (" + placeholders + ")")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, it := range SortItems(items) {
		if _, err := stmt.Exec(itemArgs(it)...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DumpJSON writes all items as one compact JSON array line to path.
func (s *Store) DumpJSON(path string) (int, error) {
	items, err := s.ListAll()
	if err != nil {
		return 0, err
	}
	data, err := CompactArrayBytes(items)
	if err != nil {
		return 0, err
	}
	if err := writeFileBytes(path, data); err != nil {
		return 0, err
	}
	return len(items), nil
}
