package github

import "time"

type GitHubUser struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type GitHubLabel struct {
	Name string `json:"name"`
}

type GitHubPullRequestRef struct {
	URL string `json:"url"`
}

type GitHubIssue struct {
	ID           int64                 `json:"id"`
	Number       int                   `json:"number"`
	Title        string                `json:"title"`
	Body         string                `json:"body"`
	State        string                `json:"state"`
	HTMLURL      string                `json:"html_url"`
	User         GitHubUser            `json:"user"`
	Labels       []GitHubLabel         `json:"labels"`
	Assignees    []GitHubUser          `json:"assignees"`
	PullRequest  *GitHubPullRequestRef `json:"pull_request,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

type GitHubPullRequest struct {
	ID        int64        `json:"id"`
	Number    int          `json:"number"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	State     string       `json:"state"`
	HTMLURL   string       `json:"html_url"`
	User      GitHubUser   `json:"user"`
	Labels    []GitHubLabel `json:"labels"`
	Assignees []GitHubUser `json:"assignees"`
	MergedAt  *time.Time   `json:"merged_at,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type GitHubRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Archived bool   `json:"archived"`
	Disabled bool   `json:"disabled"`
}

type Item struct {
	ID           int64     `json:"id"`
	GitHubID     int64     `json:"github_id"`
	Type         string    `json:"type"`
	State        string    `json:"state"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	URL          string    `json:"url"`
	Number       int       `json:"number"`
	Org          string    `json:"org"`
	Repo         string    `json:"repo"`
	Author       string    `json:"author"`
	AuthorAvatar string    `json:"author_avatar"`
	Labels       string    `json:"labels"`
	Assignees    string    `json:"assignees"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	SyncedAt     time.Time `json:"synced_at"`
}
