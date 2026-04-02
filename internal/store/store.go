package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Pipeline struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Source       string   `json:"source"`
	Destination  string   `json:"destination"`
	Schedule     string   `json:"schedule"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(dataDir, "pipeline.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS pipelines (
			id TEXT PRIMARY KEY,\n\t\t\tname TEXT DEFAULT '',\n\t\t\tsource TEXT DEFAULT '',\n\t\t\tdestination TEXT DEFAULT '',\n\t\t\tschedule TEXT DEFAULT '',\n\t\t\tstatus TEXT DEFAULT 'idle',
			created_at TEXT DEFAULT (datetime('now'))
		)`)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func (d *DB) Create(e *Pipeline) error {
	e.ID = genID()
	e.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err := d.db.Exec(`INSERT INTO pipelines (id, name, source, destination, schedule, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Name, e.Source, e.Destination, e.Schedule, e.Status, e.CreatedAt)
	return err
}

func (d *DB) Get(id string) *Pipeline {
	row := d.db.QueryRow(`SELECT id, name, source, destination, schedule, status, created_at FROM pipelines WHERE id=?`, id)
	var e Pipeline
	if err := row.Scan(&e.ID, &e.Name, &e.Source, &e.Destination, &e.Schedule, &e.Status, &e.CreatedAt); err != nil {
		return nil
	}
	return &e
}

func (d *DB) List() []Pipeline {
	rows, err := d.db.Query(`SELECT id, name, source, destination, schedule, status, created_at FROM pipelines ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []Pipeline
	for rows.Next() {
		var e Pipeline
		if err := rows.Scan(&e.ID, &e.Name, &e.Source, &e.Destination, &e.Schedule, &e.Status, &e.CreatedAt); err != nil {
			continue
		}
		result = append(result, e)
	}
	return result
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM pipelines WHERE id=?`, id)
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM pipelines`).Scan(&n)
	return n
}
