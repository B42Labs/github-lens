package api

import "net/http"

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/items", h.ListItems)
	mux.HandleFunc("GET /api/items/{id}", h.GetItem)
	mux.HandleFunc("POST /api/sync", h.TriggerSync)
	mux.HandleFunc("GET /api/sync/status", h.SyncStatus)
	mux.HandleFunc("GET /api/config/orgs", h.ListOrgs)
	mux.HandleFunc("GET /api/repos", h.ListRepos)
	mux.HandleFunc("GET /api/labels", h.ListLabels)
	mux.HandleFunc("GET /api/authors", h.ListAuthors)
	mux.HandleFunc("GET /api/stats", h.GetStats)

	// Frontend (SPA catch-all)
	mux.Handle("/", h.frontendHandler())

	return Chain(mux, CORSWithOrigin(h.config.Server.CORSOrigin), RequestLogger, Recovery)
}
