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
	ExitCode   *int
	Stdout     *string
	Stderr     *string
}

func (d *DB) InsertRun(ctx context.Context, r Run) error {
	var finished *string
	if r.FinishedAt != nil {
		s := r.FinishedAt.Format(time.RFC3339Nano)
		finished = &s
	}
	_, err := d.SQL.ExecContext(ctx, `
INSERT INTO runs (id, function_id, version_id, status, created_at, finished_at, message, exit_code, stdout, stderr)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, r.ID, r.FunctionID, r.VersionID, r.Status, r.CreatedAt.Format(time.RFC3339Nano), finished, r.Message, r.ExitCode, r.Stdout, r.Stderr)
	return err
}

func (d *DB) GetRun(ctx context.Context, runID string) (*Run, error) {
	row := d.SQL.QueryRowContext(ctx, `
SELECT id, function_id, version_id, status, created_at, finished_at, message, exit_code, stdout, stderr
FROM runs WHERE id = ?
`, runID)
	var r Run
	var created, finished sql.NullString
	var msg, stdout, stderr sql.NullString
	var exitCode sql.NullInt64
	if err := row.Scan(&r.ID, &r.FunctionID, &r.VersionID, &r.Status, &created, &finished, &msg, &exitCode, &stdout, &stderr); err != nil {
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
	if exitCode.Valid {
		code := int(exitCode.Int64)
		r.ExitCode = &code
	}
	if stdout.Valid {
		s := stdout.String
		r.Stdout = &s
	}
	if stderr.Valid {
		s := stderr.String
		r.Stderr = &s
	}
	return &r, nil
}
