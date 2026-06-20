package microsoft

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type staticChangingSource struct {
	tok *oauth2.Token
}

func (s *staticChangingSource) Token() (*oauth2.Token, error) {
	return s.tok, nil
}

func TestSavingSource_CallsSaveOnNewToken(t *testing.T) {
	var captured *oauth2.Token
	var saveCalls int

	base := &staticChangingSource{tok: &oauth2.Token{
		AccessToken:  "new-access",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(time.Hour),
	}}

	src := &savingSource{
		base: base,
		save: func(tok *oauth2.Token) { captured = tok; saveCalls++ },
	}

	tok, err := src.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok.AccessToken != "new-access" {
		t.Errorf("AccessToken=%q want new-access", tok.AccessToken)
	}
	if saveCalls != 1 {
		t.Errorf("saveCalls=%d want 1", saveCalls)
	}
	if captured == nil || captured.AccessToken != "new-access" {
		t.Errorf("captured=%v want AccessToken new-access", captured)
	}

	tok2, err := src.Token()
	if err != nil {
		t.Fatalf("Token second call: %v", err)
	}
	if tok2.AccessToken != "new-access" {
		t.Errorf("AccessToken=%q want new-access", tok2.AccessToken)
	}
	if saveCalls != 1 {
		t.Errorf("saveCalls=%d want still 1 (no redundant save)", saveCalls)
	}
}

func TestSavingSource_SavesOnRotatedRefreshToken(t *testing.T) {
	var captured *oauth2.Token
	var saveCalls int

	base := &staticChangingSource{tok: &oauth2.Token{
		AccessToken:  "access-v1",
		RefreshToken: "refresh-v1",
		Expiry:       time.Now().Add(time.Hour),
	}}

	src := &savingSource{
		base: base,
		save: func(tok *oauth2.Token) { captured = tok; saveCalls++ },
	}

	if _, err := src.Token(); err != nil {
		t.Fatalf("Token first: %v", err)
	}

	base.tok = &oauth2.Token{
		AccessToken:  "access-v2",
		RefreshToken: "refresh-v2",
		Expiry:       time.Now().Add(time.Hour),
	}

	if _, err := src.Token(); err != nil {
		t.Fatalf("Token second: %v", err)
	}

	if saveCalls != 2 {
		t.Errorf("saveCalls=%d want 2", saveCalls)
	}
	if captured == nil || captured.RefreshToken != "refresh-v2" {
		t.Errorf("captured.RefreshToken=%v want refresh-v2", captured)
	}
}
