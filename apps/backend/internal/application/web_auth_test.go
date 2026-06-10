package application

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeMagicRepo struct{ inserted map[string][]byte }

func (f *fakeMagicRepo) InsertMagicLink(_ context.Context, email string, hash []byte, _ time.Time) error {
	f.inserted[email] = hash
	return nil
}
func (f *fakeMagicRepo) ConsumeMagicLink(_ context.Context, hash []byte, _ time.Time) (string, bool, error) {
	for email, h := range f.inserted {
		if string(h) == string(hash) {
			return email, true, nil
		}
	}
	return "", false, nil
}

type fakeMailer struct{ lastTo, lastBody string }

func (m *fakeMailer) Send(_ context.Context, to, _, body string) error {
	m.lastTo, m.lastBody = to, body
	return nil
}

func fixedClock() time.Time { return time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC) }

func TestRequestMagicLinkStoresHashAndEmailsRawToken(t *testing.T) {
	repo := &fakeMagicRepo{inserted: map[string][]byte{}}
	mail := &fakeMailer{}
	svc := newMagicLinkService(repo, mail, "https://app.example.com", 15*time.Minute, fixedClock)
	if err := svc.RequestMagicLink(context.Background(), "u@yandex.ru"); err != nil {
		t.Fatal(err)
	}
	if mail.lastTo != "u@yandex.ru" {
		t.Fatalf("recipient %q", mail.lastTo)
	}
	if repo.inserted["u@yandex.ru"] == nil {
		t.Fatal("hash not stored")
	}
	if !strings.Contains(mail.lastBody, "https://app.example.com/api/auth/web/magic/verify?token=") {
		t.Fatalf("link missing in body: %q", mail.lastBody)
	}

}

func TestVerifyMagicLinkReturnsEmailForIssuedToken(t *testing.T) {
	repo := &fakeMagicRepo{inserted: map[string][]byte{}}
	mail := &fakeMailer{}
	svc := newMagicLinkService(repo, mail, "https://app.example.com", 15*time.Minute, fixedClock)
	_ = svc.RequestMagicLink(context.Background(), "a@b.com")
	raw := mail.lastBody[strings.Index(mail.lastBody, "token=")+len("token="):]
	raw = strings.TrimRight(strings.SplitN(raw, "\"", 2)[0], " \r\n<")
	email, err := svc.VerifyMagicLink(context.Background(), raw)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if email != "a@b.com" {
		t.Fatalf("got %q", email)
	}
}

func TestVerifyMagicLinkRejectsUnknownToken(t *testing.T) {
	repo := &fakeMagicRepo{inserted: map[string][]byte{}}
	svc := newMagicLinkService(repo, &fakeMailer{}, "https://app.example.com", 15*time.Minute, fixedClock)
	if _, err := svc.VerifyMagicLink(context.Background(), "nope"); err == nil {
		t.Fatal("expected error for unknown token")
	}
}
