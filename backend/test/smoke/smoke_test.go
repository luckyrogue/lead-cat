//go:build smoke

package smoke

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var (
	baseURL = env("SMOKE_BASE_URL", "http://localhost:8080")
	token   = env("SMOKE_TOKEN", "Bearer smoke-owner")
	tokenB  = env("SMOKE_TOKEN_B", "Bearer smoke-stranger")
)

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

func firstID(t *testing.T, body string) string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("parse id from %s: %v", body, err)
	}
	var id string
	if err := json.Unmarshal(m["id"], &id); err != nil || id == "" {
		t.Fatalf("no id in %s", body)
	}
	return id
}

func TestSmoke(t *testing.T) {
	out := must(t, http.MethodGet, "/api/health", "", "")
	if !strings.Contains(out, `"postgres":"ok"`) {
		t.Fatalf("health not ok: %s", out)
	}

	must(t, http.MethodGet, "/api/me", token, "")

	slug := "smoke-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ws := must(t, http.MethodPost, "/api/workspaces", token,
		fmt.Sprintf(`{"name":"Smoke Lair","slug":%q}`, slug))
	wid := firstID(t, ws)

	must(t, http.MethodGet, "/api/workspaces/"+wid, token, "")

	if code, _ := do(t, http.MethodGet, "/api/workspaces/"+wid, tokenB, ""); code != http.StatusForbidden {
		t.Fatalf("IDOR: stranger want 403, got %d", code)
	}

	def := `{"nodes":[{"id":"t1","type":"trigger.manual","parameters":{}}],"edges":[]}`
	sc := must(t, http.MethodPost, "/api/workspaces/"+wid+"/scenarios", token,
		fmt.Sprintf(`{"name":"Smoke run","definition":%s}`, def))
	sid := firstID(t, sc)

	run := must(t, http.MethodPost, "/api/workspaces/"+wid+"/scenarios/"+sid+"/run", token, "")
	if !strings.Contains(run, `"run_id"`) {
		t.Fatalf("run did not return run_id: %s", run)
	}

	timeout := 30 * time.Second
	if s := os.Getenv("SMOKE_RUN_TIMEOUT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			timeout = time.Duration(n) * time.Second
		}
	}
	deadline := time.Now().Add(timeout)
	var status string
	for time.Now().Before(deadline) {
		runs := must(t, http.MethodGet, "/api/workspaces/"+wid+"/scenarios/"+sid+"/runs", token, "")
		var arr []struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal([]byte(runs), &arr); err == nil && len(arr) > 0 {
			status = arr[0].Status
		}
		if status == "success" {
			break
		}
		if status == "failed" {
			t.Fatalf("run failed: %s", runs)
		}
		time.Sleep(time.Second)
	}
	if status != "success" {
		t.Fatalf("timeout waiting for success (last status: %q)", status)
	}

	// 9. integrations verify (optional)
	if os.Getenv("SMOKE_SKIP_VCS") != "1" {
		code, _ := do(t, http.MethodPost, "/api/workspaces/"+wid+"/integrations/verify", token, "")
		if code != http.StatusOK && code != http.StatusBadRequest {
			t.Fatalf("integrations verify: unexpected status %d", code)
		}
	}

	t.Logf("Lead Cat smoke OK (workspace %s, scenario %s)", wid, sid)
}

func TestSmokeMeetings(t *testing.T) {
	// workspace
	slug := "smoke-mtg-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ws := must(t, http.MethodPost, "/api/workspaces", token,
		fmt.Sprintf(`{"name":"Smoke Meetings","slug":%q}`, slug))
	wid := firstID(t, ws)

	// create meeting (participant by email; stub calendar returns a Meet link)
	body := `{"dept":"Разработка","type":"Планёрка","host":"Иванов А.А.",` +
		`"date":"2025-06-02","start":"10:00","end":"11:00","recurrence":"weekly",` +
		`"participants":[{"email":"a@example.com"}]}`
	created := must(t, http.MethodPost, "/api/workspaces/"+wid+"/meetings", token, body)
	if !strings.Contains(created, `"meet_link":"https://meet.google.com/`) {
		t.Fatalf("no meet link in created meeting: %s", created)
	}
	mid := firstID(t, created)

	// list (owner sees it)
	list := must(t, http.MethodGet, "/api/workspaces/"+wid+"/meetings", token, "")
	if !strings.Contains(list, mid) {
		t.Fatalf("created meeting not in list: %s", list)
	}

	// get
	got := must(t, http.MethodGet, "/api/workspaces/"+wid+"/meetings/"+mid, token, "")
	if !strings.Contains(got, `"name":"Разработка | Планёрка | Иванов А.А. | 2025-06-02 | Еженедельно"`) {
		t.Fatalf("unexpected meeting name: %s", got)
	}

	// cancel
	if code, _ := do(t, http.MethodDelete, "/api/workspaces/"+wid+"/meetings/"+mid, token, ""); code != http.StatusNoContent {
		t.Fatalf("delete: want 204, got %d", code)
	}

	t.Logf("meetings smoke OK (workspace %s, meeting %s)", wid, mid)
}
