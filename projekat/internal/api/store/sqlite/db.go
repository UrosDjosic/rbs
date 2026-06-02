package sqlite

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

type DB struct {
	SQL *sql.DB
}

func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	dbc, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	dbc.SetMaxOpenConns(1)
	dbc.SetConnMaxLifetime(5 * time.Minute)
	return &DB{SQL: dbc}, nil
}

func (d *DB) Close() error {
	return d.SQL.Close()
}

func (d *DB) ExecSchema(ctx context.Context, schemaSQL string) error {
	_, err := d.SQL.ExecContext(ctx, schemaSQL)
	return err
}
