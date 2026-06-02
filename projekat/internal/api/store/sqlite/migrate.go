package sqlite

import (
	"context"
	"strings"
)

// Migrate applies additive schema changes for existing databases.
func (d *DB) Migrate(ctx context.Context) error {
	alters := []string{
		`ALTER TABLE functions ADD COLUMN active_version_id TEXT`,
		`ALTER TABLE functions ADD COLUMN deployed_at TEXT`,
	}
	for _, q := range alters {
		if _, err := d.SQL.ExecContext(ctx, q); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				return err
			}
		}
	}
	return nil
}
