package store

import (
	"fmt"
	"strings"

	"github.com/b42labs/github-lens/internal/github"
)

type queryParts struct {
	where  string
	args   []any
	join   string
	order  string
	limit  string
}

var allowedSortFields = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"title":      true,
}

func (s *Store) ListItems(params ListParams) (*ListResult, error) {
	// Apply defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 25
	}
	if params.PerPage > 100 {
		params.PerPage = 100
	}
	if params.Sort == "" {
		params.Sort = "updated_at"
	}
	if params.Order == "" {
		params.Order = "desc"
	}

	qp := buildListQuery(params)

	// Count total
	countSQL := "SELECT COUNT(*) FROM items" + qp.join
	if qp.where != "" {
		countSQL += " WHERE " + qp.where
	}
	var total int
	if err := s.db.QueryRow(countSQL, qp.args...).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch items
	selectSQL := `SELECT items.id, items.github_id, items.type, items.state, items.title, items.body,
		items.url, items.number, items.org, items.repo, items.author, items.author_avatar,
		items.labels, items.assignees, items.created_at, items.updated_at, items.synced_at
		FROM items` + qp.join
	if qp.where != "" {
		selectSQL += " WHERE " + qp.where
	}
	selectSQL += qp.order + qp.limit

	offset := (params.Page - 1) * params.PerPage
	args := append(qp.args, params.PerPage, offset)

	rows, err := s.db.Query(selectSQL, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []github.Item
	for rows.Next() {
		var item github.Item
		if err := rows.Scan(
			&item.ID, &item.GitHubID, &item.Type, &item.State, &item.Title, &item.Body,
			&item.URL, &item.Number, &item.Org, &item.Repo, &item.Author, &item.AuthorAvatar,
			&item.Labels, &item.Assignees, &item.CreatedAt, &item.UpdatedAt, &item.SyncedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []github.Item{}
	}

	totalPages := total / params.PerPage
	if total%params.PerPage != 0 {
		totalPages++
	}

	return &ListResult{
		Items:      items,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func buildListQuery(params ListParams) queryParts {
	var qp queryParts
	var conditions []string
	var args []any

	// FTS search
	if params.Query != "" {
		qp.join = " JOIN items_fts ON items.id = items_fts.rowid"
		conditions = append(conditions, "items_fts MATCH ?")
		args = append(args, sanitizeFTSQuery(params.Query))
	}

	if params.Type != "" {
		conditions = append(conditions, "items.type = ?")
		args = append(args, params.Type)
	}
	if params.State != "" {
		conditions = append(conditions, "items.state = ?")
		args = append(args, params.State)
	}
	if params.Org != "" {
		conditions = append(conditions, "items.org = ?")
		args = append(args, params.Org)
	}
	if params.Repo != "" {
		conditions = append(conditions, "items.repo = ?")
		args = append(args, params.Repo)
	}
	if params.Author != "" {
		conditions = append(conditions, "items.author = ?")
		args = append(args, params.Author)
	}
	if params.Label != "" {
		// Use comma-delimited matching to prevent false matches (e.g. "bug" matching "debugger").
		// Escape LIKE wildcards in user input.
		conditions = append(conditions, "',' || items.labels || ',' LIKE ? ESCAPE '\\'")
		args = append(args, "%,"+escapeLike(params.Label)+",%")
	}
	if params.Since != "" {
		conditions = append(conditions, "items.updated_at >= ?")
		args = append(args, params.Since)
	}

	if len(conditions) > 0 {
		qp.where = strings.Join(conditions, " AND ")
	}
	qp.args = args

	// Sort - whitelist to prevent SQL injection
	sortField := "items.updated_at"
	if allowedSortFields[params.Sort] {
		sortField = "items." + params.Sort
	}
	order := "DESC"
	if strings.EqualFold(params.Order, "asc") {
		order = "ASC"
	}
	qp.order = fmt.Sprintf(" ORDER BY %s %s", sortField, order)
	qp.limit = " LIMIT ? OFFSET ?"

	return qp
}

// sanitizeFTSQuery wraps each token in double quotes for literal matching,
// preventing FTS5 syntax injection.
func sanitizeFTSQuery(q string) string {
	tokens := strings.Fields(q)
	quoted := make([]string, len(tokens))
	for i, t := range tokens {
		// Remove any existing double quotes
		t = strings.ReplaceAll(t, `"`, "")
		quoted[i] = `"` + t + `"`
	}
	return strings.Join(quoted, " ")
}

// escapeLike escapes LIKE pattern wildcards in user input.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
