package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/b42labs/github-lens/internal/config"
	"github.com/b42labs/github-lens/internal/github"
	"github.com/b42labs/github-lens/internal/store"
	"golang.org/x/sync/errgroup"
)

var ErrSyncInProgress = errors.New("sync already in progress")

type Status struct {
	Running  bool       `json:"running"`
	LastRun  *time.Time `json:"last_run,omitempty"`
	Progress string     `json:"progress,omitempty"`
}

type Service struct {
	cfg    *config.Config
	client *github.Client
	store  *store.Store

	mu       sync.Mutex
	running  bool
	progress string
	lastRun  *time.Time
	appCtx   context.Context
}

func NewService(cfg *config.Config, client *github.Client, store *store.Store) *Service {
	return &Service{
		cfg:    cfg,
		client: client,
		store:  store,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.appCtx = ctx
	interval := s.cfg.SyncInterval()
	if interval == 0 {
		slog.Info("auto-sync disabled")
		return
	}

	slog.Info("starting auto-sync", "interval", interval)

	// Initial sync
	go func() {
		if err := s.runSync(ctx); err != nil {
			slog.Error("initial sync failed", "error", err)
		}
	}()

	// Periodic sync
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.runSync(ctx); err != nil {
					slog.Error("periodic sync failed", "error", err)
				}
			}
		}
	}()
}

func (s *Service) TriggerSync(_ context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrSyncInProgress
	}
	s.running = true
	s.progress = "starting"
	s.mu.Unlock()

	// Use the app-level context so the sync survives after the HTTP request ends
	ctx := s.appCtx
	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		defer func() {
			now := time.Now()
			s.mu.Lock()
			s.running = false
			s.lastRun = &now
			s.progress = ""
			s.mu.Unlock()
		}()
		if err := s.doSync(ctx); err != nil {
			slog.Error("manual sync failed", "error", err)
		}
	}()
	return nil
}

// SyncAndWait runs a full sync synchronously, blocking until complete.
func (s *Service) SyncAndWait(ctx context.Context) error {
	return s.runSync(ctx)
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Status{
		Running:  s.running,
		LastRun:  s.lastRun,
		Progress: s.progress,
	}
}

func (s *Service) runSync(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrSyncInProgress
	}
	s.running = true
	s.progress = "starting"
	s.mu.Unlock()

	defer func() {
		now := time.Now()
		s.mu.Lock()
		s.running = false
		s.lastRun = &now
		s.progress = ""
		s.mu.Unlock()
	}()

	return s.doSync(ctx)
}

// doSync performs the actual sync work. Caller must manage the running state.
func (s *Service) doSync(ctx context.Context) error {
	slog.Info("sync started")
	start := time.Now()

	for _, org := range s.cfg.Organizations {
		if err := s.syncOrg(ctx, org); err != nil {
			return fmt.Errorf("syncing org %s: %w", org.Name, err)
		}
	}

	slog.Info("sync completed", "duration", time.Since(start).Round(time.Millisecond))
	return nil
}

func (s *Service) syncOrg(ctx context.Context, org config.OrgConfig) error {
	s.setProgress(fmt.Sprintf("fetching repos for %s", org.Name))

	repos, err := s.client.ListOrgRepos(ctx, org.Name)
	if err != nil {
		return fmt.Errorf("listing repos: %w", err)
	}

	filtered := filterRepos(repos, org)
	slog.Info("syncing org", "org", org.Name, "repos", len(filtered))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(s.cfg.Sync.Concurrency)

	for _, repo := range filtered {
		g.Go(func() error {
			return s.syncRepo(gctx, org.Name, repo.Name)
		})
	}

	return g.Wait()
}

func (s *Service) syncRepo(ctx context.Context, org, repo string) error {
	s.setProgress(fmt.Sprintf("syncing %s/%s", org, repo))

	logID, err := s.store.LogSyncStart(org, repo)
	if err != nil {
		slog.Warn("failed to log sync start", "error", err)
	}

	var items []github.Item
	var syncErr error

	// Fetch issues
	issues, err := s.client.ListIssues(ctx, org, repo)
	if err != nil {
		syncErr = fmt.Errorf("fetching issues: %w", err)
	} else {
		for _, issue := range issues {
			items = append(items, convertIssue(org, repo, issue))
		}
	}

	// Fetch PRs
	if syncErr == nil {
		prs, err := s.client.ListPullRequests(ctx, org, repo)
		if err != nil {
			syncErr = fmt.Errorf("fetching PRs: %w", err)
		} else {
			for _, pr := range prs {
				items = append(items, convertPR(org, repo, pr))
			}
		}
	}

	// Upsert items
	if syncErr == nil && len(items) > 0 {
		if err := s.store.UpsertItems(items); err != nil {
			syncErr = fmt.Errorf("upserting items: %w", err)
		}
	}

	// Log result
	status := "success"
	errStr := ""
	if syncErr != nil {
		status = "error"
		errStr = syncErr.Error()
	}
	if logID > 0 {
		if err := s.store.LogSyncFinish(logID, status, len(items), errStr); err != nil {
			slog.Warn("failed to log sync finish", "error", err)
		}
	}

	if syncErr != nil {
		slog.Error("sync repo failed", "org", org, "repo", repo, "error", syncErr)
	} else {
		slog.Info("sync repo complete", "org", org, "repo", repo, "items", len(items))
	}
	return syncErr
}

func (s *Service) setProgress(msg string) {
	s.mu.Lock()
	s.progress = msg
	s.mu.Unlock()
}

func filterRepos(repos []github.GitHubRepo, org config.OrgConfig) []github.GitHubRepo {
	var filtered []github.GitHubRepo
	includeSet := toSet(org.IncludeRepos)
	excludeSet := toSet(org.ExcludeRepos)

	for _, r := range repos {
		if r.Archived || r.Disabled {
			continue
		}
		if len(includeSet) > 0 {
			if _, ok := includeSet[r.Name]; !ok {
				continue
			}
		}
		if _, ok := excludeSet[r.Name]; ok {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func toSet(slice []string) map[string]struct{} {
	s := make(map[string]struct{}, len(slice))
	for _, v := range slice {
		s[v] = struct{}{}
	}
	return s
}

func convertIssue(org, repo string, issue github.GitHubIssue) github.Item {
	state := issue.State
	return github.Item{
		GitHubID:     issue.ID,
		Type:         "issue",
		State:        state,
		Title:        issue.Title,
		Body:         issue.Body,
		URL:          issue.HTMLURL,
		Number:       issue.Number,
		Org:          org,
		Repo:         repo,
		Author:       issue.User.Login,
		AuthorAvatar: issue.User.AvatarURL,
		Labels:       joinLabels(issue.Labels),
		Assignees:    marshalAssignees(issue.Assignees),
		CreatedAt:    issue.CreatedAt,
		UpdatedAt:    issue.UpdatedAt,
		SyncedAt:     time.Now().UTC(),
	}
}

func convertPR(org, repo string, pr github.GitHubPullRequest) github.Item {
	state := pr.State
	if pr.MergedAt != nil {
		state = "merged"
	}
	return github.Item{
		GitHubID:     pr.ID,
		Type:         "pr",
		State:        state,
		Title:        pr.Title,
		Body:         pr.Body,
		URL:          pr.HTMLURL,
		Number:       pr.Number,
		Org:          org,
		Repo:         repo,
		Author:       pr.User.Login,
		AuthorAvatar: pr.User.AvatarURL,
		Labels:       joinLabels(pr.Labels),
		Assignees:    marshalAssignees(pr.Assignees),
		CreatedAt:    pr.CreatedAt,
		UpdatedAt:    pr.UpdatedAt,
		SyncedAt:     time.Now().UTC(),
	}
}

func joinLabels(labels []github.GitHubLabel) string {
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.Name
	}
	return strings.Join(names, ",")
}

func marshalAssignees(users []github.GitHubUser) string {
	if len(users) == 0 {
		return "[]"
	}
	type assignee struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}
	list := make([]assignee, len(users))
	for i, u := range users {
		list[i] = assignee{Login: u.Login, AvatarURL: u.AvatarURL}
	}
	b, _ := json.Marshal(list)
	return string(b)
}
