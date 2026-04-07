package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

type DB struct{ db *sql.DB }

type Pipeline struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Steps       []Step `json:"steps"`
	Schedule    string `json:"schedule,omitempty"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	RunCount    int    `json:"run_count"`
	LastRun     string `json:"last_run,omitempty"`
	LastStatus  string `json:"last_status,omitempty"`
}

type Step struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Config map[string]string `json:"config,omitempty"`
}

type Run struct {
	ID          string       `json:"id"`
	PipelineID  string       `json:"pipeline_id"`
	Status      string       `json:"status"`
	StartedAt   string       `json:"started_at"`
	FinishedAt  string       `json:"finished_at,omitempty"`
	DurationMs  int          `json:"duration_ms"`
	StepResults []StepResult `json:"step_results"`
	Error       string       `json:"error,omitempty"`
}

type StepResult struct {
	StepName   string `json:"step_name"`
	Status     string `json:"status"`
	DurationMs int    `json:"duration_ms"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
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
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS pipelines (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT DEFAULT '', steps_json TEXT DEFAULT '[]', schedule TEXT DEFAULT '', enabled INTEGER DEFAULT 1, created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS runs (id TEXT PRIMARY KEY, pipeline_id TEXT NOT NULL, status TEXT DEFAULT 'running', started_at TEXT, finished_at TEXT DEFAULT '', duration_ms INTEGER DEFAULT 0, results_json TEXT DEFAULT '[]', error TEXT DEFAULT '', created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE INDEX IF NOT EXISTS idx_runs_pipeline ON runs(pipeline_id)`,
	} {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(resource TEXT NOT NULL,record_id TEXT NOT NULL,data TEXT NOT NULL DEFAULT '{}',PRIMARY KEY(resource, record_id))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) hydrate(p *Pipeline) {
	d.db.QueryRow(`SELECT COUNT(*) FROM runs WHERE pipeline_id=?`, p.ID).Scan(&p.RunCount)
	d.db.QueryRow(`SELECT started_at FROM runs WHERE pipeline_id=? ORDER BY started_at DESC LIMIT 1`, p.ID).Scan(&p.LastRun)
	d.db.QueryRow(`SELECT status FROM runs WHERE pipeline_id=? ORDER BY started_at DESC LIMIT 1`, p.ID).Scan(&p.LastStatus)
}

func (d *DB) CreatePipeline(p *Pipeline) error {
	p.ID = genID()
	p.CreatedAt = now()
	if p.Steps == nil {
		p.Steps = []Step{}
	}
	sj, _ := json.Marshal(p.Steps)
	en := 1
	if !p.Enabled {
		en = 0
	}
	_, err := d.db.Exec(`INSERT INTO pipelines (id,name,description,steps_json,schedule,enabled,created_at) VALUES (?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Description, string(sj), p.Schedule, en, p.CreatedAt)
	return err
}

func (d *DB) GetPipeline(id string) *Pipeline {
	var p Pipeline
	var sj string
	var en int
	if err := d.db.QueryRow(`SELECT id,name,description,steps_json,schedule,enabled,created_at FROM pipelines WHERE id=?`, id).Scan(&p.ID, &p.Name, &p.Description, &sj, &p.Schedule, &en, &p.CreatedAt); err != nil {
		return nil
	}
	json.Unmarshal([]byte(sj), &p.Steps)
	p.Enabled = en == 1
	d.hydrate(&p)
	return &p
}

func (d *DB) ListPipelines() []Pipeline {
	rows, _ := d.db.Query(`SELECT id,name,description,steps_json,schedule,enabled,created_at FROM pipelines ORDER BY name ASC`)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []Pipeline
	for rows.Next() {
		var p Pipeline
		var sj string
		var en int
		rows.Scan(&p.ID, &p.Name, &p.Description, &sj, &p.Schedule, &en, &p.CreatedAt)
		json.Unmarshal([]byte(sj), &p.Steps)
		p.Enabled = en == 1
		d.hydrate(&p)
		out = append(out, p)
	}
	return out
}

func (d *DB) UpdatePipeline(id string, p *Pipeline) error {
	sj, _ := json.Marshal(p.Steps)
	en := 1
	if !p.Enabled {
		en = 0
	}
	_, err := d.db.Exec(`UPDATE pipelines SET name=?,description=?,steps_json=?,schedule=?,enabled=? WHERE id=?`,
		p.Name, p.Description, string(sj), p.Schedule, en, id)
	return err
}

func (d *DB) DeletePipeline(id string) error {
	d.db.Exec(`DELETE FROM runs WHERE pipeline_id=?`, id)
	_, err := d.db.Exec(`DELETE FROM pipelines WHERE id=?`, id)
	return err
}

func (d *DB) SaveRun(r *Run) error {
	r.ID = genID()
	rj, _ := json.Marshal(r.StepResults)
	_, err := d.db.Exec(`INSERT INTO runs (id,pipeline_id,status,started_at,finished_at,duration_ms,results_json,error) VALUES (?,?,?,?,?,?,?,?)`,
		r.ID, r.PipelineID, r.Status, r.StartedAt, r.FinishedAt, r.DurationMs, string(rj), r.Error)
	return err
}

func (d *DB) ListRuns(pipelineID string, limit int) []Run {
	if limit <= 0 {
		limit = 20
	}
	rows, _ := d.db.Query(`SELECT id,pipeline_id,status,started_at,finished_at,duration_ms,results_json,error FROM runs WHERE pipeline_id=? ORDER BY started_at DESC LIMIT ?`, pipelineID, limit)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var r Run
		var rj string
		rows.Scan(&r.ID, &r.PipelineID, &r.Status, &r.StartedAt, &r.FinishedAt, &r.DurationMs, &rj, &r.Error)
		json.Unmarshal([]byte(rj), &r.StepResults)
		out = append(out, r)
	}
	return out
}

func (d *DB) GetRun(id string) *Run {
	var r Run
	var rj string
	if err := d.db.QueryRow(`SELECT id,pipeline_id,status,started_at,finished_at,duration_ms,results_json,error FROM runs WHERE id=?`, id).Scan(&r.ID, &r.PipelineID, &r.Status, &r.StartedAt, &r.FinishedAt, &r.DurationMs, &rj, &r.Error); err != nil {
		return nil
	}
	json.Unmarshal([]byte(rj), &r.StepResults)
	return &r
}

type Stats struct {
	Pipelines int `json:"pipelines"`
	Runs      int `json:"runs"`
	Active    int `json:"active"`
}

func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM pipelines`).Scan(&s.Pipelines)
	d.db.QueryRow(`SELECT COUNT(*) FROM runs`).Scan(&s.Runs)
	d.db.QueryRow(`SELECT COUNT(*) FROM pipelines WHERE enabled=1`).Scan(&s.Active)
	return s
}

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
