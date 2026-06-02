package sqlite

import (
	"context"
	"database/sql"
	"time"
)

type Run struct {
	ID         string
	FunctionID string
	VersionID  string
	Status     string
	CreatedAt  time.Time
	FinishedAt *time.Time
	Message    *string
}

func (d *DB) InsertRun(ctx context.Context, r Run) error {
	var finished *string
	if r.FinishedAt != nil {
		s := r.FinishedAt.Format(time.RFC3339Nano)
		finished = &s
	}
	_, err := d.SQL.ExecContext(ctx, `
INSERT INTO runs (id, function_id, version_id, status, created_at, finished_at, message)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, r.ID, r.FunctionID, r.VersionID, r.Status, r.CreatedAt.Format(time.RFC3339Nano), finished, r.Message)
	return err
}

func (d *DB) GetRun(ctx context.Context, runID string) (*Run, error) {
	row := d.SQL.QueryRowContext(ctx, `
SELECT id, function_id, version_id, status, created_at, finished_at, message
FROM runs WHERE id = ?
`, runID)
	var r Run
	var created, finished sql.NullString
	var msg sql.NullString
	if err := row.Scan(&r.ID, &r.FunctionID, &r.VersionID, &r.Status, &created, &finished, &msg); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created.String)
	if err != nil {
		return nil, err
	}
	r.CreatedAt = t
	if finished.Valid {
		ft, err := time.Parse(time.RFC3339Nano, finished.String)
		if err != nil {
			return nil, err
		}
		r.FinishedAt = &ft
	}
	if msg.Valid {
		m := msg.String
		r.Message = &m
	}
	return &r, nil
}
