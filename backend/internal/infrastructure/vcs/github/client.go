package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Commit struct {
	SHA     string
	Repo    string
	Message string
}

type Client struct {
	token string
	http  *http.Client
	org   string
}

func NewClient(token, org string) *Client {
	return &Client{
		token: strings.TrimSpace(token),
		org:   strings.TrimSpace(org),
		http:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.token != "" && c.org != ""
}

func (c *Client) ListCommits(ctx context.Context, author string, since, until time.Time, loc *time.Location) ([]Commit, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("github client not configured")
	}
	author = strings.TrimSpace(author)
	if author == "" {
		return nil, fmt.Errorf("empty github author")
	}
	if loc == nil {
		loc = time.UTC
	}

	from := since.In(loc).Format("2006-01-02")
	to := until.In(loc).Format("2006-01-02")
	q := fmt.Sprintf("org:%s author:%s committer-date:%s..%s", c.org, author, from, to)

	var all []Commit
	for page := 1; page <= 5; page++ {
		items, total, err := c.searchPage(ctx, q, page)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(all) >= total || len(items) == 0 {
			break
		}
	}
	return all, nil
}

func (c *Client) searchPage(ctx context.Context, q string, page int) ([]Commit, int, error) {
	u := url.URL{
		Scheme:   "https",
		Host:     "api.github.com",
		Path:     "/search/commits",
		RawQuery: url.Values{"q": {q}, "per_page": {"100"}, "page": {fmt.Sprint(page)}}.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("github search %d: %s", res.StatusCode, truncate(string(body), 200))
	}

	var parsed searchCommitsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, 0, err
	}

	out := make([]Commit, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		msg := strings.TrimSpace(strings.Split(item.Commit.Message, "\n")[0])
		if len(msg) > 80 {
			msg = msg[:77] + "…"
		}
		out = append(out, Commit{
			SHA:     shortSHA(item.SHA),
			Repo:    item.Repository.FullName,
			Message: msg,
		})
	}
	return out, parsed.TotalCount, nil
}

type searchCommitsResponse struct {
	TotalCount int `json:"total_count"`
	Items      []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"items"`
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
