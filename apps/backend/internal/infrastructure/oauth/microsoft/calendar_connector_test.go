package microsoft

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
	c := NewCalendarConnector("cid", "secret")
	u := c.AuthURL("st", "chal", "https://app.example.com/cb")
	q, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	v := q.Query()
	if v.Get("prompt") != "consent" {
		t.Errorf("prompt=%q want consent", v.Get("prompt"))
	}
	if v.Get("code_challenge") != "chal" || v.Get("code_challenge_method") != "S256" {
		t.Errorf("pkce missing: %v", v)
	}
	if v.Get("state") != "st" {
		t.Errorf("state=%q", v.Get("state"))
	}
	scope := v.Get("scope")
	for _, want := range []string{"Calendars.ReadWrite", "OnlineMeetings.ReadWrite", "offline_access"} {
		if !strings.Contains(scope, want) {
			t.Errorf("scope %q missing %q", scope, want)
		}
	}
}

func TestCalendarConnector_Exchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "at",
			"refresh_token": "rt",
			"expires_in":    3600,
			"scope":         "https://graph.microsoft.com/Calendars.ReadWrite offline_access",
			"token_type":    "Bearer",
		})
	}))
	defer srv.Close()

	c := NewCalendarConnector("cid", "secret")
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
	if !strings.Contains(tok.Scopes, "Calendars.ReadWrite") || !strings.Contains(tok.Scopes, "offline_access") {
		t.Errorf("Scopes=%q must include Calendars.ReadWrite + offline_access", tok.Scopes)
	}
}
