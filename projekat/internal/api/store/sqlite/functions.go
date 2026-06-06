package sqlite

import (
	"context"
	"database/sql"
	"time"
)

type Function struct {
	ID          string
	OwnerUserID string
	Name        *string
	CreatedAt   time.Time
}

type FunctionVersion struct {
	ID         string
	FunctionID string
	CreatedAt  time.Time
	Status     string
	SrcZipPath string
	SrcSHA256  string
}

func (d *DB) InsertFunction(ctx context.Context, f Function) error {
	_, err := d.SQL.ExecContext(ctx, `
INSERT INTO functions (id, owner_user_id, name, created_at)
VALUES (?, ?, ?, ?)
`, f.ID, f.OwnerUserID, f.Name, f.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (d *DB) InsertFunctionVersion(ctx context.Context, v FunctionVersion) error {
	_, err := d.SQL.ExecContext(ctx, `
INSERT INTO function_versions (id, function_id, created_at, status, src_zip_path, src_sha256)
VALUES (?, ?, ?, ?, ?, ?)
`, v.ID, v.FunctionID, v.CreatedAt.Format(time.RFC3339Nano), v.Status, v.SrcZipPath, v.SrcSHA256)
	return err
}

func (d *DB) UpdateFunctionVersionStatus(ctx context.Context, versionID, status string) error {
	_, err := d.SQL.ExecContext(ctx, `
UPDATE function_versions SET status = ? WHERE id = ?
`, status, versionID)
	return err
}

type FunctionRow struct {
	FunctionID string  `json:"function_id"`
	VersionID  string  `json:"version_id"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
	Name       *string `json:"name,omitempty"`
	InvokeURL  *string `json:"invoke_url,omitempty"`
}

func (d *DB) ListFunctions(ctx context.Context, ownerUserID string, limit int) ([]FunctionRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := d.SQL.QueryContext(ctx, `
SELECT f.id, v.id, v.status, v.created_at, f.name, f.deployed_at
FROM functions f
JOIN function_versions v ON v.function_id = f.id
WHERE f.owner_user_id = ?
ORDER BY v.created_at DESC
LIMIT ?
`, ownerUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FunctionRow
	for rows.Next() {
		var r FunctionRow
		var deployed sql.NullString
		if err := rows.Scan(&r.FunctionID, &r.VersionID, &r.Status, &r.CreatedAt, &r.Name, &deployed); err != nil {
			return nil, err
		}
		_ = deployed
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (d *DB) GetFunctionOwner(ctx context.Context, functionID string) (string, error) {
	row := d.SQL.QueryRowContext(ctx, `SELECT owner_user_id FROM functions WHERE id = ?`, functionID)
	var owner string
	if err := row.Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return owner, nil
}

func (d *DB) GetLatestVersion(ctx context.Context, functionID string) (*FunctionVersion, error) {
	row := d.SQL.QueryRowContext(ctx, `
SELECT id, function_id, created_at, status, src_zip_path, src_sha256
FROM function_versions
WHERE function_id = ?
ORDER BY created_at DESC
LIMIT 1
`, functionID)
	var v FunctionVersion
	var created string
	if err := row.Scan(&v.ID, &v.FunctionID, &created, &v.Status, &v.SrcZipPath, &v.SrcSHA256); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	v.CreatedAt = t
	return &v, nil
}

func (d *DB) DeployFunction(ctx context.Context, functionID, versionID string, deployedAt time.Time) error {
	tx, err := d.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
UPDATE functions SET active_version_id = ?, deployed_at = ? WHERE id = ?
`, versionID, deployedAt.Format(time.RFC3339Nano), functionID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE function_versions SET status = 'deployed' WHERE id = ?
`, versionID); err != nil {
		return err
	}
	return tx.Commit()
}

type DeployedFunction struct {
	FunctionID      string
	ActiveVersionID string
	DeployedAt      time.Time
}

func (d *DB) GetDeployedFunction(ctx context.Context, functionID string) (*DeployedFunction, error) {
	row := d.SQL.QueryRowContext(ctx, `
SELECT id, active_version_id, deployed_at
FROM functions
WHERE id = ? AND active_version_id IS NOT NULL AND deployed_at IS NOT NULL
`, functionID)
	var df DeployedFunction
	var deployed string
	if err := row.Scan(&df.FunctionID, &df.ActiveVersionID, &deployed); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, deployed)
	if err != nil {
		return nil, err
	}
	df.DeployedAt = t
	return &df, nil
}
