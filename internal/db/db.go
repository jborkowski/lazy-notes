package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const (
	schemaRecordings = `
CREATE TABLE IF NOT EXISTS recordings (
  recording_id INTEGER PRIMARY KEY,
  created_at TEXT,
  title TEXT,
  audio_path TEXT,
  language TEXT,
  mode_key TEXT,
  status TEXT NOT NULL,
  submitted_at TEXT,
  error TEXT
);`

	schemaMeta = `
CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value TEXT
);`
)

type DB struct {
	sql *sql.DB
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	d := &DB{sql: sqlDB}
	if err := d.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error {
	return d.sql.Close()
}

func (d *DB) migrate() error {
	if _, err := d.sql.Exec(schemaRecordings); err != nil {
		return fmt.Errorf("migrate recordings: %w", err)
	}
	for _, col := range []struct{ name, typ string }{
		{"sw_id", "TEXT"},
		{"note_path", "TEXT"},
		{"published_at", "TEXT"},
		{"body", "TEXT"},
		{"source", "TEXT"},
		{"external_id", "TEXT"},
	} {
		if err := d.addColumnIfNotExists("recordings", col.name, col.typ); err != nil {
			return fmt.Errorf("migrate recordings column %s: %w", col.name, err)
		}
	}
	// Existing rows are HF ingest.
	if _, err := d.sql.Exec(`UPDATE recordings SET source = ? WHERE source IS NULL OR source = ''`, SourceHF); err != nil {
		return fmt.Errorf("backfill source: %w", err)
	}
	if _, err := d.sql.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS recordings_source_external_id
ON recordings(source, external_id)
WHERE external_id IS NOT NULL AND external_id != ''`); err != nil {
		return fmt.Errorf("migrate external_id index: %w", err)
	}
	if _, err := d.sql.Exec(schemaMeta); err != nil {
		return fmt.Errorf("migrate meta: %w", err)
	}
	return nil
}

func (d *DB) addColumnIfNotExists(table, column, colType string) error {
	var count int
	if err := d.sql.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`,
		table,
		column,
	).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := d.sql.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
	return err
}
