package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/b42labs/github-lens/internal/config"
	"github.com/b42labs/github-lens/internal/store"
	"github.com/b42labs/github-lens/internal/sync"
)

type Handler struct {
	store    *store.Store
	syncSvc  *sync.Service
	config   *config.Config
	frontend fs.FS
}

func NewHandler(s *store.Store, syncSvc *sync.Service, cfg *config.Config, frontend fs.FS) *Handler {
	return &Handler{
		store:    s,
		syncSvc:  syncSvc,
		config:   cfg,
		frontend: frontend,
	}
}

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	params := store.ListParams{
		Query:   r.URL.Query().Get("q"),
		Type:    r.URL.Query().Get("type"),
		State:   r.URL.Query().Get("state"),
		Org:     r.URL.Query().Get("org"),
		Repo:    r.URL.Query().Get("repo"),
		Author:  r.URL.Query().Get("author"),
		Label:   r.URL.Query().Get("label"),
		Sort:    r.URL.Query().Get("sort"),
		Order:   r.URL.Query().Get("order"),
		Page:    intParam(r, "page", 1),
		PerPage: intParam(r, "per_page", 25),
	}
	if s := r.URL.Query().Get("since"); s != "" {
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_SINCE", "since must be RFC3339 format")
			return
		}
		params.Since = s
	}

	result, err := h.store.ListItems(params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid item id")
		return
	}

	item, err := h.store.GetItem(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}
	if item == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "item not found")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	err := h.syncSvc.TriggerSync(r.Context())
	if err == sync.ErrSyncInProgress {
		writeError(w, http.StatusConflict, "SYNC_IN_PROGRESS", "sync is already running")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SYNC_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sync started"})
}

func (h *Handler) SyncStatus(w http.ResponseWriter, r *http.Request) {
	status := h.syncSvc.Status()
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	orgs := make([]map[string]any, len(h.config.Organizations))
	for i, o := range h.config.Organizations {
		orgs[i] = map[string]any{
			"name":          o.Name,
			"include_repos": o.IncludeRepos,
			"exclude_repos": o.ExcludeRepos,
		}
	}
	writeJSON(w, http.StatusOK, orgs)
}

func (h *Handler) ListRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := h.store.ListRepos()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repos)
}

func (h *Handler) ListAuthors(w http.ResponseWriter, r *http.Request) {
	authors, err := h.store.ListAuthors()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, authors)
}

func (h *Handler) ListLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := h.store.ListLabels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, labels)
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) frontendHandler() http.Handler {
	fileServer := http.FileServer(http.FS(h.frontend))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		// Check if file exists
		f, err := h.frontend.Open(path[1:]) // strip leading /
		if err != nil {
			// SPA fallback: serve 200.html for unknown paths
			r.URL.Path = "/200.html"
			fileServer.ServeHTTP(w, r)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}

func intParam(r *http.Request, name string, defaultVal int) int {
	s := r.URL.Query().Get(name)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
