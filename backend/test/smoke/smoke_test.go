//go:build smoke

package smoke

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var baseURL = env("SMOKE_BASE_URL", "http://localhost:8080")

func do(t *testing.T, method, path, auth, body string) (int, string) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, baseURL+path, rdr)
	if err != nil {
		t.Fatalf("%s %s: build request: %v", method, path, err)
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func must(t *testing.T, method, path, auth, body string) string {
	t.Helper()
	code, out := do(t, method, path, auth, body)
	if code < 200 || code >= 300 {
		t.Fatalf("%s %s: want 2xx, got %d: %s", method, path, code, out)
	}
	return out
}

func TestSmoke(t *testing.T) {
	out := must(t, http.MethodGet, "/api/health", "", "")
	if !strings.Contains(out, `"postgres":"ok"`) {
		t.Fatalf("health not ok: %s", out)
	}
	if token := os.Getenv("SMOKE_MINIAPP_TOKEN"); token != "" {
		must(t, http.MethodGet, "/api/miniapp/me", token, "")
	}
	t.Log("Lead Cat smoke OK")
}
