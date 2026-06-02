package sqlite

import (
	"context"
	"time"
)

type AuditEvent struct {
	ID          string
	Ts          time.Time
	ActorUserID *string
	Action      string
	Path        string
	Method      string
	Status      int
	IP          *string
	UserAgent   *string
	Details     *string
}

func (d *DB) InsertAudit(ctx context.Context, e AuditEvent) error {
	_, err := d.SQL.ExecContext(ctx, `
INSERT INTO audit_events (
  id, ts, actor_user_id, action, path, method, status, ip, user_agent, details
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, e.ID,
		e.Ts.Format(time.RFC3339Nano),
		e.ActorUserID,
		e.Action,
		e.Path,
		e.Method,
		e.Status,
		e.IP,
		e.UserAgent,
		e.Details,
	)
	return err
}
