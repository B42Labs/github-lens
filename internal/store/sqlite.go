package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/b42labs/github-lens/internal/github"
	_ "modernc.org/sqlite"
)

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

type Store struct {
	db *sql.DB
}

type ListParams struct {
	Query   string
	Type    string
	State   string
	Org     string
	Repo    string
	Author  string
	Label   string
	Sort    string
	Order   string
	Page    int
	PerPage int
}

type ListResult struct {
	Items      []github.Item `json:"items"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
	TotalPages int           `json:"total_pages"`
}

type Stats struct {
	OpenIssues int `json:"open_issues"`
	OpenPRs    int `json:"open_prs"`
	RepoCount  int `json:"repo_count"`
}

type SyncLog struct {
	ID         int64      `json:"id"`
	Org        string     `json:"org"`
	Repo       string     `json:"repo"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Status     string     `json:"status"`
	ItemsCount int        `json:"items_count"`
	Error      string     `json:"error,omitempty"`
}

func New(dbPath string) (*Store, error) {
	// Append pragmas to the DSN so they apply to every connection in the pool
	dsn := dbPath + "?_pragma=busy_timeout%3d10000&_pragma=journal_mode%3dWAL&_pragma=synchronous%3dNORMAL&_pragma=foreign_keys%3dON"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// SQLite only supports one concurrent writer. Limit the pool to a single
	// connection so all writes are serialized and we never hit SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) UpsertItem(item github.Item) error {
	_, err := s.db.Exec(`
		INSERT INTO items (github_id, type, state, title, body, url, number, org, repo, author, author_avatar, labels, assignees, created_at, updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(github_id) DO UPDATE SET
			state=excluded.state,
			title=excluded.title,
			body=excluded.body,
			url=excluded.url,
			labels=excluded.labels,
			assignees=excluded.assignees,
			updated_at=excluded.updated_at,
			synced_at=excluded.synced_at
	`, item.GitHubID, item.Type, item.State, item.Title, item.Body, item.URL,
		item.Number, item.Org, item.Repo, item.Author, item.AuthorAvatar,
		item.Labels, item.Assignees, item.CreatedAt, item.UpdatedAt, item.SyncedAt)
	return err
}

func (s *Store) UpsertItems(items []github.Item) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO items (github_id, type, state, title, body, url, number, org, repo, author, author_avatar, labels, assignees, created_at, updated_at, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(github_id) DO UPDATE SET
			state=excluded.state,
			title=excluded.title,
			body=excluded.body,
			url=excluded.url,
			labels=excluded.labels,
			assignees=excluded.assignees,
			updated_at=excluded.updated_at,
			synced_at=excluded.synced_at
	`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, item := range items {
		_, err := stmt.Exec(
			item.GitHubID, item.Type, item.State, item.Title, item.Body, item.URL,
			item.Number, item.Org, item.Repo, item.Author, item.AuthorAvatar,
			item.Labels, item.Assignees, item.CreatedAt, item.UpdatedAt, item.SyncedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetItem(id int64) (*github.Item, error) {
	var item github.Item
	err := s.db.QueryRow(`
		SELECT id, github_id, type, state, title, body, url, number, org, repo,
		       author, author_avatar, labels, assignees, created_at, updated_at, synced_at
		FROM items WHERE id = ?
	`, id).Scan(
		&item.ID, &item.GitHubID, &item.Type, &item.State, &item.Title, &item.Body,
		&item.URL, &item.Number, &item.Org, &item.Repo, &item.Author, &item.AuthorAvatar,
		&item.Labels, &item.Assignees, &item.CreatedAt, &item.UpdatedAt, &item.SyncedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

type RepoInfo struct {
	Org  string `json:"org"`
	Repo string `json:"repo"`
}

func (s *Store) ListRepos() ([]RepoInfo, error) {
	rows, err := s.db.Query(`SELECT DISTINCT org, repo FROM items ORDER BY org, repo`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var repos []RepoInfo
	for rows.Next() {
		var r RepoInfo
		if err := rows.Scan(&r.Org, &r.Repo); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	if repos == nil {
		repos = []RepoInfo{}
	}
	return repos, nil
}

func (s *Store) ListLabels() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT labels FROM items WHERE labels != ''`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	seen := make(map[string]struct{})
	for rows.Next() {
		var csv string
		if err := rows.Scan(&csv); err != nil {
			return nil, err
		}
		for _, l := range splitCSV(csv) {
			if l != "" {
				seen[l] = struct{}{}
			}
		}
	}

	labels := make([]string, 0, len(seen))
	for l := range seen {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	return labels, nil
}

func (s *Store) ListAuthors() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT author FROM items ORDER BY author`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var authors []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}
	if authors == nil {
		authors = []string{}
	}
	return authors, nil
}

func (s *Store) GetStats() (*Stats, error) {
	var stats Stats
	err := s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN type = 'issue' AND state = 'open' THEN 1 END),
			COUNT(CASE WHEN type = 'pr' AND state = 'open' THEN 1 END),
			COUNT(DISTINCT org || '/' || repo)
		FROM items
	`).Scan(&stats.OpenIssues, &stats.OpenPRs, &stats.RepoCount)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *Store) LogSyncStart(org, repo string) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO sync_log (org, repo, started_at) VALUES (?, ?, ?)`,
		org, repo, time.Now().UTC(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) LogSyncFinish(id int64, status string, itemsCount int, syncErr string) error {
	_, err := s.db.Exec(
		`UPDATE sync_log SET finished_at=?, status=?, items_count=?, error=? WHERE id=?`,
		time.Now().UTC(), status, itemsCount, syncErr, id,
	)
	return err
}

func (s *Store) GetLastSync() (*SyncLog, error) {
	var log SyncLog
	var finishedAt sql.NullTime
	var syncErr sql.NullString
	err := s.db.QueryRow(`
		SELECT id, org, repo, started_at, finished_at, status, items_count, error
		FROM sync_log ORDER BY started_at DESC LIMIT 1
	`).Scan(&log.ID, &log.Org, &log.Repo, &log.StartedAt, &finishedAt, &log.Status, &log.ItemsCount, &syncErr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if finishedAt.Valid {
		log.FinishedAt = &finishedAt.Time
	}
	if syncErr.Valid {
		log.Error = syncErr.String
	}
	return &log, nil
}
