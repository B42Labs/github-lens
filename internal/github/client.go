package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		baseURL:    "https://api.github.com",
	}
}

func (c *Client) ListOrgRepos(ctx context.Context, org string) ([]GitHubRepo, error) {
	var all []GitHubRepo
	err := c.paginate(ctx, fmt.Sprintf("/orgs/%s/repos?per_page=100&type=all", org), func(body []byte) error {
		var repos []GitHubRepo
		if err := json.Unmarshal(body, &repos); err != nil {
			return err
		}
		all = append(all, repos...)
		return nil
	})
	return all, err
}

func (c *Client) ListIssues(ctx context.Context, org, repo string) ([]GitHubIssue, error) {
	var all []GitHubIssue
	err := c.paginate(ctx, fmt.Sprintf("/repos/%s/%s/issues?per_page=100&state=all", org, repo), func(body []byte) error {
		var issues []GitHubIssue
		if err := json.Unmarshal(body, &issues); err != nil {
			return err
		}
		// Filter out pull requests (they show up in the issues endpoint)
		for _, issue := range issues {
			if issue.PullRequest == nil {
				all = append(all, issue)
			}
		}
		return nil
	})
	return all, err
}

func (c *Client) ListPullRequests(ctx context.Context, org, repo string) ([]GitHubPullRequest, error) {
	var all []GitHubPullRequest
	err := c.paginate(ctx, fmt.Sprintf("/repos/%s/%s/pulls?per_page=100&state=all", org, repo), func(body []byte) error {
		var prs []GitHubPullRequest
		if err := json.Unmarshal(body, &prs); err != nil {
			return err
		}
		all = append(all, prs...)
		return nil
	})
	return all, err
}

var linkNextRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

func (c *Client) paginate(ctx context.Context, path string, handler func([]byte) error) error {
	url := c.baseURL + path
	for url != "" {
		body, headers, err := c.doWithRetry(ctx, url)
		if err != nil {
			return err
		}
		if err := handler(body); err != nil {
			return err
		}
		// Follow Link: rel="next"
		url = ""
		if link := headers.Get("Link"); link != "" {
			if m := linkNextRe.FindStringSubmatch(link); len(m) > 1 {
				url = m[1]
			}
		}
	}
	return nil
}

func (c *Client) doWithRetry(ctx context.Context, url string) ([]byte, http.Header, error) {
	const maxRetries = 5
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second

	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, nil, err
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt == maxRetries-1 {
				return nil, nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
			}
			c.sleep(ctx, backoff(attempt, baseDelay, maxDelay))
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("reading response body: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			return body, resp.Header, nil
		}

		// Rate limit handling
		if resp.StatusCode == http.StatusForbidden {
			// Secondary rate limit: Retry-After header
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if secs, err := strconv.Atoi(retryAfter); err == nil {
					slog.Warn("secondary rate limit hit, waiting", "seconds", secs)
					c.sleep(ctx, time.Duration(secs)*time.Second)
					continue
				}
			}
			// Primary rate limit
			if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining == "0" {
				if resetStr := resp.Header.Get("X-RateLimit-Reset"); resetStr != "" {
					if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
						waitDuration := time.Until(time.Unix(resetUnix, 0)) + time.Second
						if waitDuration > 0 {
							slog.Warn("rate limit exhausted, waiting for reset", "wait", waitDuration.Round(time.Second))
							c.sleep(ctx, waitDuration)
							continue
						}
					}
				}
			}
		}

		// Retry on 5xx
		if resp.StatusCode >= 500 && attempt < maxRetries-1 {
			c.sleep(ctx, backoff(attempt, baseDelay, maxDelay))
			continue
		}

		// Parse error message
		var errResp struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Message != "" {
			return nil, nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, errResp.Message)
		}
		return nil, nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil, nil, fmt.Errorf("request failed after %d retries", maxRetries)
}

func backoff(attempt int, base, max time.Duration) time.Duration {
	delay := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if delay > max {
		delay = max
	}
	// Add jitter: 50-100% of delay
	jitter := time.Duration(rand.Int64N(int64(delay / 2)))
	return delay/2 + jitter
}

func (c *Client) sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
