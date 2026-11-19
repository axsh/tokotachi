-- Backlog knowledge DB (SQLite cache). Applied by backlog.Open.
-- uuid: RFC 4122 string issued once at create time; NOT the PRIMARY KEY.
-- Table PK remains (phase, branch, local_id). Restore may reassign local_id; uuid is the durable identity.
CREATE TABLE IF NOT EXISTS items (
  phase TEXT NOT NULL,
  branch TEXT NOT NULL,
  local_id TEXT NOT NULL,
  slug TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT '未確認',
  state_date TEXT,
  kind TEXT NOT NULL DEFAULT '',
  ask_to TEXT NOT NULL DEFAULT '',
  due TEXT NOT NULL DEFAULT '',
  origin TEXT NOT NULL DEFAULT '',
  decision TEXT NOT NULL DEFAULT '',
  problem TEXT NOT NULL DEFAULT '',
  current_handling TEXT NOT NULL DEFAULT '',
  options TEXT NOT NULL DEFAULT '',
  questions TEXT NOT NULL DEFAULT '',
  after_decision TEXT NOT NULL DEFAULT '',
  history TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  uuid TEXT NOT NULL,
  PRIMARY KEY (phase, branch, local_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS items_uuid ON items(uuid);
