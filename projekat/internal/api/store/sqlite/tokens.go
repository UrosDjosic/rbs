package sqlite

import (
	"context"
	"database/sql"
	"time"
)

type Token struct {
	Token     string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (d *DB) InsertToken(ctx context.Context, t Token) error {
	_, err := d.SQL.ExecContext(ctx, `
INSERT INTO tokens (token, user_id, created_at, expires_at)
VALUES (?, ?, ?, ?)
`, t.Token, t.UserID, t.CreatedAt.Format(time.RFC3339Nano), t.ExpiresAt.Format(time.RFC3339Nano))
	return err
}

func (d *DB) GetToken(ctx context.Context, token string) (*Token, error) {
	row := d.SQL.QueryRowContext(ctx, `
SELECT token, user_id, created_at, expires_at
FROM tokens
WHERE token = ?
`, token)
	var t Token
	var created, expires string
	if err := row.Scan(&t.Token, &t.UserID, &created, &expires); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ct, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	et, err := time.Parse(time.RFC3339Nano, expires)
	if err != nil {
		return nil, err
	}
	t.CreatedAt = ct
	t.ExpiresAt = et
	return &t, nil
}
