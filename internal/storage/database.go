package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Database wraps a SQLite connection and provides all data access methods.
type Database struct {
	db *sql.DB
}

// Open creates or opens a SQLite database at the given path with WAL mode.
// Use ":memory:" for in-memory testing.
func Open(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	d := &Database{db: db}
	if err := d.createTables(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return d, nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	return d.db.Close()
}

// ExecRaw executes a raw SQL statement (exposed for testing).
func (d *Database) ExecRaw(query string, args ...interface{}) {
	_, _ = d.db.Exec(query, args...)
}

// Tables returns all user table names (for testing).
func (d *Database) Tables() []string {
	rows, err := d.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}
	return tables
}

func (d *Database) createTables() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS products (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT    NOT NULL,
			category   TEXT    NOT NULL,
			first_seen DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(name, category)
		)`,
		`CREATE TABLE IF NOT EXISTS price_history (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL REFERENCES products(id),
			shop       TEXT    NOT NULL,
			price      REAL    NOT NULL,
			currency   TEXT    NOT NULL,
			old_price  REAL,
			url        TEXT    NOT NULL,
			timestamp  DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS deal_notifications (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id  INTEGER  NOT NULL REFERENCES products(id),
			shop        TEXT     NOT NULL,
			price       REAL     NOT NULL,
			notified_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS exchange_rates (
			currency   TEXT PRIMARY KEY,
			rate       REAL     NOT NULL,
			fetched_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
	}

	for _, stmt := range statements {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("creating table: %w", err)
		}
	}
	return nil
}
