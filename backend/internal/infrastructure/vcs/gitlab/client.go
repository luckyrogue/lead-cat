package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	domainvcs "github.com/Jaryq-Lab/notify-bot/internal/domain/vcs"
)

type Client struct {
	Token     string
	Namespace string
	BaseURL   string
	http      *http.Client
}

func New(token, namespace, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	return &Client{
		Token:     strings.TrimSpace(token),
		Namespace: strings.TrimSpace(namespace),
		BaseURL:   strings.TrimRight(baseURL, "/"),
		http:      &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.Token != "" && c.Namespace != ""
}

func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return fmt.Errorf("gitlab client not configured")
	}
	path := url.PathEscape(c.Namespace)
	u := fmt.Sprintf("%s/api/v4/groups/%s", c.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.Token)
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gitlab ping %d: %s", res.StatusCode, truncate(string(body), 200))
	}
	return nil
}

func (c *Client) ListCommits(ctx context.Context, author string, since, until time.Time, loc *time.Location) ([]domainvcs.Commit, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("gitlab client not configured")
	}
	author = strings.TrimSpace(author)
	if author == "" {
		return nil, fmt.Errorf("empty gitlab author")
	}
	userID, err := c.resolveUserID(ctx, author)
	if err != nil {
		return nil, err
	}
	sinceS := since.In(loc).Format(time.RFC3339)
	untilS := until.In(loc).Format(time.RFC3339)
	u := fmt.Sprintf("%s/api/v4/users/%d/events?after=%s&before=%s&per_page=100",
		c.BaseURL, userID, url.QueryEscape(sinceS), url.QueryEscape(untilS))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.Token)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab events %d: %s", res.StatusCode, truncate(string(body), 200))
	}
	var events []struct {
		ActionName string `json:"action_name"`
		ProjectID  int    `json:"project_id"`
		PushData   *struct {
			CommitTitle string `json:"commit_title"`
			CommitTo    string `json:"commit_to"`
		} `json:"push_data"`
	}
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, err
	}
	var out []domainvcs.Commit
	seen := make(map[string]bool)
	for _, ev := range events {
		if ev.ActionName != "pushed to" && ev.ActionName != "pushed new" {
			continue
		}
		if ev.PushData == nil {
			continue
		}
		sha := ev.PushData.CommitTo
		if len(sha) > 7 {
			sha = sha[:7]
		}
		key := sha + ev.PushData.CommitTitle
		if seen[key] {
			continue
		}
		seen[key] = true
		msg := ev.PushData.CommitTitle
		if len(msg) > 80 {
			msg = msg[:77] + "…"
		}
		out = append(out, domainvcs.Commit{
			SHA:     sha,
			Repo:    c.Namespace,
			Message: msg,
		})
	}
	return out, nil
}

func (c *Client) resolveUserID(ctx context.Context, username string) (int, error) {
	u := fmt.Sprintf("%s/api/v4/users?username=%s", c.BaseURL, url.QueryEscape(username))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.Token)
	res, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("gitlab user lookup %d", res.StatusCode)
	}
	var users []struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body, &users); err != nil {
		return 0, err
	}
	if len(users) == 0 {
		return 0, fmt.Errorf("gitlab user %q not found", username)
	}
	return users[0].ID, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
