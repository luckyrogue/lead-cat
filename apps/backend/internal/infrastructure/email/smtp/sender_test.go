package smtp

import (
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func TestEnvelopeFrom(t *testing.T) {
	cases := map[string]string{
		"Lead Cat <hi@leadcat.app>": "hi@leadcat.app",
		"hi@leadcat.app":            "hi@leadcat.app",
		"  spaced@x.io  ":           "spaced@x.io",
	}
	for in, want := range cases {
		if got := envelopeFrom(in); got != want {
			t.Errorf("envelopeFrom(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildMessage_HTMLOnly(t *testing.T) {
	s := New("smtp.test", "587", "", "", "Lead Cat <hi@leadcat.app>")
	raw, err := s.buildMessage("you@x.io", "Hello", "", "<h1>Hi</h1>", "")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	m, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ct := m.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if lu := m.Header.Get("List-Unsubscribe"); lu != "" {
		t.Fatalf("unexpected List-Unsubscribe = %q", lu)
	}
	body, _ := io.ReadAll(m.Body)
	if !strings.Contains(string(body), "<h1>Hi</h1>") {
		t.Fatalf("body missing html: %q", body)
	}
}

func TestBuildMessage_Multipart(t *testing.T) {
	s := New("smtp.test", "587", "", "", "hi@leadcat.app")
	raw, err := s.buildMessage("you@x.io", "Напоминание: «Синк»", "plain version", "<h1>rich</h1>", "https://app/profile")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	m, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Subject is RFC 2047 encoded on the wire but decodes back to the original.
	subj, err := (&mime.WordDecoder{}).DecodeHeader(m.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	if subj != "Напоминание: «Синк»" {
		t.Fatalf("subject = %q", subj)
	}

	if lu := m.Header.Get("List-Unsubscribe"); lu != "<https://app/profile>" {
		t.Fatalf("List-Unsubscribe = %q", lu)
	}
	if lup := m.Header.Get("List-Unsubscribe-Post"); lup != "List-Unsubscribe=One-Click" {
		t.Fatalf("List-Unsubscribe-Post = %q", lup)
	}

	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/alternative" {
		t.Fatalf("Content-Type = %q (%v)", m.Header.Get("Content-Type"), err)
	}

	mr := multipart.NewReader(m.Body, params["boundary"])
	var types []string
	var bodies []string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		mt, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		types = append(types, mt)
		b, _ := io.ReadAll(p)
		bodies = append(bodies, string(b))
	}
	// text/plain MUST come before text/html in multipart/alternative.
	if len(types) != 2 || types[0] != "text/plain" || types[1] != "text/html" {
		t.Fatalf("parts = %v, want [text/plain text/html]", types)
	}
	if bodies[0] != "plain version" || bodies[1] != "<h1>rich</h1>" {
		t.Fatalf("part bodies = %v", bodies)
	}
}
