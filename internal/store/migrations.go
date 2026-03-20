package store

import "database/sql"

const ddl = `
CREATE TABLE IF NOT EXISTS items (
    id              INTEGER PRIMARY KEY,
    github_id       INTEGER NOT NULL UNIQUE,
    type            TEXT NOT NULL CHECK(type IN ('issue', 'pr')),
    state           TEXT NOT NULL CHECK(state IN ('open', 'closed', 'merged')),
    title           TEXT NOT NULL,
    body            TEXT,
    url             TEXT NOT NULL,
    number          INTEGER NOT NULL,
    org             TEXT NOT NULL,
    repo            TEXT NOT NULL,
    author          TEXT NOT NULL,
    author_avatar   TEXT,
    labels          TEXT,
    assignees       TEXT,
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    synced_at       DATETIME NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
    title, body, labels, repo, org, author,
    content=items, content_rowid=id
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER IF NOT EXISTS items_ai AFTER INSERT ON items BEGIN
    INSERT INTO items_fts(rowid, title, body, labels, repo, org, author)
    VALUES (new.id, new.title, new.body, new.labels, new.repo, new.org, new.author);
END;

CREATE TRIGGER IF NOT EXISTS items_ad AFTER DELETE ON items BEGIN
    INSERT INTO items_fts(items_fts, rowid, title, body, labels, repo, org, author)
    VALUES ('delete', old.id, old.title, old.body, old.labels, old.repo, old.org, old.author);
END;

CREATE TRIGGER IF NOT EXISTS items_au AFTER UPDATE ON items BEGIN
    INSERT INTO items_fts(items_fts, rowid, title, body, labels, repo, org, author)
    VALUES ('delete', old.id, old.title, old.body, old.labels, old.repo, old.org, old.author);
    INSERT INTO items_fts(rowid, title, body, labels, repo, org, author)
    VALUES (new.id, new.title, new.body, new.labels, new.repo, new.org, new.author);
END;

CREATE TABLE IF NOT EXISTS sync_log (
    id          INTEGER PRIMARY KEY,
    org         TEXT NOT NULL,
    repo        TEXT NOT NULL,
    started_at  DATETIME NOT NULL,
    finished_at DATETIME,
    status      TEXT,
    items_count INTEGER DEFAULT 0,
    error       TEXT
);

CREATE INDEX IF NOT EXISTS idx_items_org_repo ON items(org, repo);
CREATE INDEX IF NOT EXISTS idx_items_type ON items(type);
CREATE INDEX IF NOT EXISTS idx_items_state ON items(state);
CREATE INDEX IF NOT EXISTS idx_items_author ON items(author);
CREATE INDEX IF NOT EXISTS idx_items_updated_at ON items(updated_at);
`

func Migrate(db *sql.DB) error {
	_, err := db.Exec(ddl)
	return err
}
