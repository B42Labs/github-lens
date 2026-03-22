package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/b42labs/github-lens/internal/config"
	"github.com/b42labs/github-lens/internal/store"
)

func setupTestHandler(t *testing.T) *Handler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return &Handler{
		store:  s,
		config: &config.Config{},
	}
}

func TestListItems_InvalidSince(t *testing.T) {
	h := setupTestHandler(t)

	tests := []struct {
		name       string
		since      string
		wantStatus int
	}{
		{"valid RFC3339", "2025-06-01T00:00:00Z", http.StatusOK},
		{"invalid string", "banana", http.StatusBadRequest},
		{"date only", "2025-06-01", http.StatusBadRequest},
		{"empty (no param)", "", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/items"
			if tt.since != "" {
				url += "?since=" + tt.since
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			h.ListItems(w, req)
			if w.Code != tt.wantStatus {
				var body map[string]string
				_ = json.Unmarshal(w.Body.Bytes(), &body)
				t.Errorf("since=%q: got status %d, want %d (body: %v)", tt.since, w.Code, tt.wantStatus, body)
			}
		})
	}
}
