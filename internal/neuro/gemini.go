package neuro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const geminiModel = "gemini-2.0-flash"

var systemPrompt = `Ты злой лид-кот в Telegram. Пишешь человеку не из стаи разработчиков.
Отвечай кратко по-русски: зло, но по делу, с кошачьей ноткой. Не выдавай секреты, не упоминай ботов, API и переменные окружения.
Ты один персонаж — злой лид-кот. Не представляйся другим именем и не отсылай к другому помощнику.`

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Ask(ctx context.Context, userText string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("gemini api key not configured")
	}

	body := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: userText}}},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel, c.apiKey,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini api %d: %s", res.StatusCode, truncate(string(respBody), 200))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	text := parsed.firstText()
	if text == "" {
		return "", fmt.Errorf("empty gemini response")
	}
	return text, nil
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (r geminiResponse) firstText() string {
	if len(r.Candidates) == 0 {
		return ""
	}
	for _, p := range r.Candidates[0].Content.Parts {
		if p.Text != "" {
			return p.Text
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
