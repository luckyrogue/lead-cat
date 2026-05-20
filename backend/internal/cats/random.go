package cats

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

var catSearches = []string{
	"angry cats",
	"angry cat",
	"grumpy cat",
	"mad cat",
	"evil cat",
}

var extraSubs = []string{"angrycats", "grumpycats"}

func RandomImageURL(ctx context.Context) string {
	client := &http.Client{Timeout: 10 * time.Second}

	var candidates []func() (string, error)
	for _, q := range catSearches {
		query := q
		candidates = append(candidates, func() (string, error) {
			return fetchCatSearch(ctx, client, query)
		})
	}
	for _, sub := range extraSubs {
		subreddit := sub
		candidates = append(candidates, func() (string, error) {
			return fetchSubredditHot(ctx, client, subreddit)
		})
	}
	candidates = append(candidates, func() (string, error) {
		return fetchSubredditHot(ctx, client, "cat")
	})

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	for _, try := range candidates {
		if url, err := try(); err == nil && url != "" {
			return url
		}
	}

	return fallbackURLs[rand.IntN(len(fallbackURLs))]
}

type redditListing struct {
	Data struct {
		Children []redditChild `json:"children"`
	} `json:"data"`
}

type redditChild struct {
	Data redditPost `json:"data"`
}

type redditPost struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	IsGallery bool   `json:"is_gallery"`
	MediaMeta map[string]struct {
		Status string `json:"status"`
		S      struct {
			U string `json:"u"`
		} `json:"s"`
	} `json:"media_metadata"`
	GalleryData struct {
		Items []struct {
			MediaID string `json:"media_id"`
		} `json:"items"`
	} `json:"gallery_data"`
	Preview struct {
		Images []struct {
			Source struct {
				URL string `json:"url"`
			} `json:"source"`
		} `json:"images"`
	} `json:"preview"`
}

func fetchCatSearch(ctx context.Context, client *http.Client, query string) (string, error) {
	url := fmt.Sprintf(
		"https://www.reddit.com/r/cat/search.json?q=%s&restrict_sr=on&sort=top&t=all&limit=50&raw_json=1",
		strings.ReplaceAll(query, " ", "+"),
	)
	return fetchListing(ctx, client, url, "r/cat?q="+query)
}

func fetchSubredditHot(ctx context.Context, client *http.Client, sub string) (string, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=50&raw_json=1", sub)
	posts, err := loadListing(ctx, client, url)
	if err != nil {
		return "", err
	}
	if sub == "cat" {
		posts = filterAngryPosts(posts)
	}
	return pickRandomURL(posts, "r/"+sub)
}

func fetchListing(ctx context.Context, client *http.Client, url, label string) (string, error) {
	posts, err := loadListing(ctx, client, url)
	if err != nil {
		return "", err
	}
	return pickRandomURL(posts, label)
}

func loadListing(ctx context.Context, client *http.Client, url string) ([]redditPost, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "lead-cat/1.0 (by /u/jaryq-lab)")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit %s: status %d", url, res.StatusCode)
	}

	var listing redditListing
	if err := json.NewDecoder(res.Body).Decode(&listing); err != nil {
		return nil, err
	}

	posts := make([]redditPost, 0, len(listing.Data.Children))
	for _, c := range listing.Data.Children {
		posts = append(posts, c.Data)
	}
	return posts, nil
}

var angryKeywords = []string{
	"angry", "grumpy", "mad", "evil", "rage", "fury", "hiss", "glare",
	"злой", "злая", "сердит", "бесит", "people hate",
}

func filterAngryPosts(posts []redditPost) []redditPost {
	var out []redditPost
	for _, p := range posts {
		title := strings.ToLower(p.Title)
		for _, kw := range angryKeywords {
			if strings.Contains(title, kw) {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func pickRandomURL(posts []redditPost, label string) (string, error) {
	var urls []string
	for _, p := range posts {
		urls = append(urls, urlsFromPost(p)...)
	}
	if len(urls) == 0 {
		return "", fmt.Errorf("no images: %s", label)
	}
	return urls[rand.IntN(len(urls))], nil
}

func urlsFromPost(p redditPost) []string {
	if p.IsGallery && len(p.GalleryData.Items) > 0 {
		var urls []string
		for _, item := range p.GalleryData.Items {
			meta, ok := p.MediaMeta[item.MediaID]
			if !ok || meta.Status != "valid" || meta.S.U == "" {
				continue
			}
			urls = append(urls, unescapeReddit(meta.S.U))
		}
		if len(urls) > 0 {
			return urls
		}
	}

	if u := pickDirectURL(p.URL); u != "" {
		return []string{u}
	}

	if len(p.Preview.Images) > 0 && p.Preview.Images[0].Source.URL != "" {
		return []string{unescapeReddit(p.Preview.Images[0].Source.URL)}
	}
	return nil
}

func pickDirectURL(direct string) string {
	lower := strings.ToLower(direct)
	if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".webm") {
		return ""
	}
	if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".gif") ||
		strings.Contains(lower, "i.redd.it") || strings.Contains(lower, "i.imgur.com") ||
		strings.Contains(lower, "preview.redd.it") {
		return unescapeReddit(direct)
	}
	return ""
}

func unescapeReddit(s string) string {
	return strings.ReplaceAll(s, "&amp;", "&")
}

var fallbackURLs = []string{
	"https://i.imgur.com/8nLFCVP.jpeg",
	"https://i.imgur.com/Cxagv.jpg",
	"https://i.imgur.com/0BqnGbM.jpeg",
	"https://i.imgur.com/JEeqy.jpg",
	"https://i.imgur.com/9EPb9fK.jpeg",
	"https://i.imgur.com/k6zih.jpg",
	"https://i.imgur.com/4uM0ta8.jpeg",
	"https://i.imgur.com/njA8Vsy.jpeg",
}
