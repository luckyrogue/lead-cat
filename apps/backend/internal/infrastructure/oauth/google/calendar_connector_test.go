package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestCalendarConnector_AuthURL(t *testing.T) {
	c := NewCalendarConnector("client-id", "secret")
	u := c.AuthURL("state-1", "challenge-1", "https://app.example.com/cb")
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if q.Get("access_type") != "offline" {
		t.Errorf("access_type=%q want offline", q.Get("access_type"))
	}
	if q.Get("prompt") != "consent" {
		t.Errorf("prompt=%q want consent", q.Get("prompt"))
	}
	if q.Get("code_challenge") != "challenge-1" || q.Get("code_challenge_method") != "S256" {
		t.Errorf("pkce missing: %v", q)
	}
	if q.Get("state") != "state-1" {
		t.Errorf("state=%q", q.Get("state"))
	}
	scope := q.Get("scope")
	if !strings.Contains(scope, "calendar.events") || !strings.Contains(scope, "calendar.readonly") {
		t.Errorf("scope=%q must include calendar.events + calendar.readonly", scope)
	}
}

func TestCalendarConnector_Exchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "at",
			"refresh_token": "rt",
			"expires_in":    3600,
			"scope":         "https://www.googleapis.com/auth/calendar.events https://www.googleapis.com/auth/calendar.readonly",
			"token_type":    "Bearer",
		})
	}))
	defer srv.Close()

	c := NewCalendarConnector("client-id", "secret")
	c.endpoint = oauth2.Endpoint{
		AuthURL:  c.endpoint.AuthURL,
		TokenURL: srv.URL,
	}

	before := time.Now()
	tok, err := c.Exchange(context.Background(), "code-xyz", "verifier-xyz", "https://app.example.com/cb")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if tok.AccessToken != "at" {
		t.Errorf("AccessToken=%q want at", tok.AccessToken)
	}
	if tok.RefreshToken != "rt" {
		t.Errorf("RefreshToken=%q want rt", tok.RefreshToken)
	}
	if tok.Expiry.IsZero() {
		t.Error("Expiry is zero")
	}
	if !tok.Expiry.After(before) {
		t.Errorf("Expiry=%v must be in the future (after %v)", tok.Expiry, before)
	}
	if !tok.Expiry.Before(time.Now().Add(2 * time.Hour)) {
		t.Errorf("Expiry=%v too far in the future for a 3600s token", tok.Expiry)
	}
	if !strings.Contains(tok.Scopes, "calendar.events") || !strings.Contains(tok.Scopes, "calendar.readonly") {
		t.Errorf("Scopes=%q must include calendar.events + calendar.readonly", tok.Scopes)
	}
}
