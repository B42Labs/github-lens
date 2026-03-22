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
