package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/b42labs/github-lens/internal/github"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func seedItems(t *testing.T, s *Store) {
	t.Helper()
	now := time.Now().UTC()
	items := []github.Item{
		{GitHubID: 1, Type: "issue", State: "open", Title: "Open issue by alice", URL: "http://x/1", Number: 1, Org: "org1", Repo: "repo1", Author: "alice", Labels: "bug", Assignees: "[]", CreatedAt: now, UpdatedAt: now, SyncedAt: now},
		{GitHubID: 2, Type: "issue", State: "open", Title: "Open issue by bob", URL: "http://x/2", Number: 2, Org: "org1", Repo: "repo1", Author: "bob", Labels: "feature", Assignees: "[]", CreatedAt: now, UpdatedAt: now, SyncedAt: now},
		{GitHubID: 3, Type: "pr", State: "open", Title: "Open PR by alice", URL: "http://x/3", Number: 3, Org: "org1", Repo: "repo2", Author: "alice", Labels: "bug", Assignees: "[]", CreatedAt: now, UpdatedAt: now, SyncedAt: now},
		{GitHubID: 4, Type: "issue", State: "closed", Title: "Closed issue by charlie", URL: "http://x/4", Number: 4, Org: "org1", Repo: "repo1", Author: "charlie", Labels: "", Assignees: "[]", CreatedAt: now, UpdatedAt: now, SyncedAt: now},
		{GitHubID: 5, Type: "pr", State: "merged", Title: "Merged PR by bob", URL: "http://x/5", Number: 5, Org: "org2", Repo: "repo3", Author: "bob", Labels: "feature,bug", Assignees: "[]", CreatedAt: now, UpdatedAt: now, SyncedAt: now},
	}
	if err := s.UpsertItems(items); err != nil {
		t.Fatalf("seeding items: %v", err)
	}
}

func TestListItems_AuthorFilter(t *testing.T) {
	s := setupTestStore(t)
	seedItems(t, s)

	tests := []struct {
		name      string
		params    ListParams
		wantTotal int
	}{
		{
			name:      "filter by author alice with state open",
			params:    ListParams{State: "open", Author: "alice"},
			wantTotal: 2,
		},
		{
			name:      "filter by author bob with state open",
			params:    ListParams{State: "open", Author: "bob"},
			wantTotal: 1,
		},
		{
			name:      "filter by author charlie with state open returns zero",
			params:    ListParams{State: "open", Author: "charlie"},
			wantTotal: 0,
		},
		{
			name:      "filter by author charlie with state closed",
			params:    ListParams{State: "closed", Author: "charlie"},
			wantTotal: 1,
		},
		{
			name:      "no author filter returns all open",
			params:    ListParams{State: "open"},
			wantTotal: 3,
		},
		{
			name:      "author filter without state",
			params:    ListParams{Author: "bob"},
			wantTotal: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.ListItems(tt.params)
			if err != nil {
				t.Fatalf("ListItems: %v", err)
			}
			if result.Total != tt.wantTotal {
				t.Errorf("got total=%d, want %d", result.Total, tt.wantTotal)
			}
			for _, item := range result.Items {
				if tt.params.Author != "" && item.Author != tt.params.Author {
					t.Errorf("item author=%q, want %q", item.Author, tt.params.Author)
				}
				if tt.params.State != "" && item.State != tt.params.State {
					t.Errorf("item state=%q, want %q", item.State, tt.params.State)
				}
			}
		})
	}
}

func TestListItems_SinceFilter(t *testing.T) {
	s := setupTestStore(t)

	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	cutoff := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	items := []github.Item{
		{GitHubID: 100, Type: "issue", State: "open", Title: "Old issue", URL: "http://x/100", Number: 100, Org: "org1", Repo: "repo1", Author: "alice", Labels: "", Assignees: "[]", CreatedAt: old, UpdatedAt: old, SyncedAt: old},
		{GitHubID: 101, Type: "issue", State: "open", Title: "Recent issue", URL: "http://x/101", Number: 101, Org: "org1", Repo: "repo1", Author: "bob", Labels: "", Assignees: "[]", CreatedAt: recent, UpdatedAt: recent, SyncedAt: recent},
		{GitHubID: 102, Type: "pr", State: "open", Title: "Old PR updated recently", URL: "http://x/102", Number: 102, Org: "org1", Repo: "repo1", Author: "alice", Labels: "", Assignees: "[]", CreatedAt: old, UpdatedAt: recent, SyncedAt: recent},
	}
	if err := s.UpsertItems(items); err != nil {
		t.Fatalf("seeding items: %v", err)
	}

	tests := []struct {
		name      string
		since     string
		wantTotal int
	}{
		{
			name:      "since before all items returns all",
			since:     "2024-01-01T00:00:00Z",
			wantTotal: 3,
		},
		{
			name:      "since cutoff returns only recent items",
			since:     cutoff.Format(time.RFC3339),
			wantTotal: 2,
		},
		{
			name:      "since after all items returns none",
			since:     "2026-01-01T00:00:00Z",
			wantTotal: 0,
		},
		{
			name:      "empty since returns all items",
			since:     "",
			wantTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.ListItems(ListParams{Since: tt.since})
			if err != nil {
				t.Fatalf("ListItems: %v", err)
			}
			if result.Total != tt.wantTotal {
				t.Errorf("got total=%d, want %d", result.Total, tt.wantTotal)
			}
		})
	}
}

func TestListItems_SinceWithOtherFilters(t *testing.T) {
	s := setupTestStore(t)

	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	cutoff := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	items := []github.Item{
		{GitHubID: 200, Type: "issue", State: "open", Title: "Old open issue", URL: "http://x/200", Number: 200, Org: "org1", Repo: "repo1", Author: "alice", Labels: "", Assignees: "[]", CreatedAt: old, UpdatedAt: old, SyncedAt: old},
		{GitHubID: 201, Type: "issue", State: "open", Title: "Recent open issue", URL: "http://x/201", Number: 201, Org: "org1", Repo: "repo1", Author: "alice", Labels: "", Assignees: "[]", CreatedAt: recent, UpdatedAt: recent, SyncedAt: recent},
		{GitHubID: 202, Type: "pr", State: "open", Title: "Recent open PR", URL: "http://x/202", Number: 202, Org: "org1", Repo: "repo1", Author: "bob", Labels: "", Assignees: "[]", CreatedAt: recent, UpdatedAt: recent, SyncedAt: recent},
		{GitHubID: 203, Type: "issue", State: "closed", Title: "Recent closed issue", URL: "http://x/203", Number: 203, Org: "org1", Repo: "repo1", Author: "alice", Labels: "", Assignees: "[]", CreatedAt: recent, UpdatedAt: recent, SyncedAt: recent},
	}
	if err := s.UpsertItems(items); err != nil {
		t.Fatalf("seeding items: %v", err)
	}

	// Since + State + Type + Author = all filters compose via AND
	result, err := s.ListItems(ListParams{
		Since:  cutoff.Format(time.RFC3339),
		State:  "open",
		Type:   "issue",
		Author: "alice",
	})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("got total=%d, want 1", result.Total)
	}
	if len(result.Items) == 1 && result.Items[0].Title != "Recent open issue" {
		t.Errorf("got title=%q, want 'Recent open issue'", result.Items[0].Title)
	}
}

func TestListItems_LabelFilter(t *testing.T) {
	s := setupTestStore(t)
	seedItems(t, s)

	result, err := s.ListItems(ListParams{Label: "bug"})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("got total=%d, want 3", result.Total)
	}
}

func TestListItems_CombinedFilters(t *testing.T) {
	s := setupTestStore(t)
	seedItems(t, s)

	result, err := s.ListItems(ListParams{
		State:  "open",
		Author: "alice",
		Type:   "issue",
	})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("got total=%d, want 1", result.Total)
	}
	if len(result.Items) == 1 {
		if result.Items[0].Title != "Open issue by alice" {
			t.Errorf("got title=%q, want 'Open issue by alice'", result.Items[0].Title)
		}
	}
}

func TestListItems_Pagination(t *testing.T) {
	s := setupTestStore(t)
	seedItems(t, s)

	result, err := s.ListItems(ListParams{PerPage: 2, Page: 1})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("got total=%d, want 5", result.Total)
	}
	if len(result.Items) != 2 {
		t.Errorf("got %d items, want 2", len(result.Items))
	}
	if result.TotalPages != 3 {
		t.Errorf("got totalPages=%d, want 3", result.TotalPages)
	}
}

func TestListAuthors(t *testing.T) {
	s := setupTestStore(t)
	seedItems(t, s)

	authors, err := s.ListAuthors()
	if err != nil {
		t.Fatalf("ListAuthors: %v", err)
	}
	want := []string{"alice", "bob", "charlie"}
	if len(authors) != len(want) {
		t.Fatalf("got %d authors, want %d", len(authors), len(want))
	}
	for i, a := range authors {
		if a != want[i] {
			t.Errorf("authors[%d]=%q, want %q", i, a, want[i])
		}
	}
}

func TestListLabels(t *testing.T) {
	s := setupTestStore(t)
	seedItems(t, s)

	labels, err := s.ListLabels()
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	want := []string{"bug", "feature"}
	if len(labels) != len(want) {
		t.Fatalf("got %d labels, want %d", len(labels), len(want))
	}
	for i, l := range labels {
		if l != want[i] {
			t.Errorf("labels[%d]=%q, want %q", i, l, want[i])
		}
	}
}

func TestListRepos(t *testing.T) {
	s := setupTestStore(t)
	seedItems(t, s)

	repos, err := s.ListRepos()
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
	if len(repos) != 3 {
		t.Fatalf("got %d repos, want 3", len(repos))
	}
}
