package sqlite

import (
	"context"
	"database/sql"
	"time"
)

type User struct {
	ID           string
	Username     string
	PasswordHash []byte
	CreatedAt    time.Time
}

func (d *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := d.SQL.QueryRowContext(ctx, `
SELECT id, username, password_hash, created_at
FROM users
WHERE username = ?
`, username)
	var u User
	var created string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	u.CreatedAt = t
	return &u, nil
}

func (d *DB) InsertUser(ctx context.Context, u User) error {
	_, err := d.SQL.ExecContext(ctx, `
INSERT INTO users (id, username, password_hash, created_at)
VALUES (?, ?, ?, ?)
`, u.ID, u.Username, u.PasswordHash, u.CreatedAt.Format(time.RFC3339Nano))
	return err
}
