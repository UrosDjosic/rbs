PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash BLOB NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tokens (
  token TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS audit_events (
  id TEXT PRIMARY KEY,
  ts TEXT NOT NULL,
  actor_user_id TEXT,
  action TEXT NOT NULL,
  path TEXT NOT NULL,
  method TEXT NOT NULL,
  status INTEGER NOT NULL,
  ip TEXT,
  user_agent TEXT,
  details TEXT,
  FOREIGN KEY(actor_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS functions (
  id TEXT PRIMARY KEY,
  owner_user_id TEXT NOT NULL,
  name TEXT,
  created_at TEXT NOT NULL,
  active_version_id TEXT,
  deployed_at TEXT,
  FOREIGN KEY(owner_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS function_versions (
  id TEXT PRIMARY KEY,
  function_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  status TEXT NOT NULL,
  src_zip_path TEXT NOT NULL,
  src_sha256 TEXT NOT NULL,
  FOREIGN KEY(function_id) REFERENCES functions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS runs (
  id TEXT PRIMARY KEY,
  function_id TEXT NOT NULL,
  version_id TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  finished_at TEXT,
  message TEXT,
  exit_code INTEGER,
  stdout TEXT,
  stderr TEXT,
  FOREIGN KEY(function_id) REFERENCES functions(id) ON DELETE CASCADE,
  FOREIGN KEY(version_id) REFERENCES function_versions(id) ON DELETE CASCADE
);

