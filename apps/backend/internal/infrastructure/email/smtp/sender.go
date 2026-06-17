package smtp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
)

type Sender struct{ host, port, user, pass, from string }

func New(host, port, user, pass, from string) *Sender {
	return &Sender{host: host, port: port, user: user, pass: pass, from: from}
}

func envelopeFrom(from string) string {
	start := strings.LastIndex(from, "<")
	end := strings.LastIndex(from, ">")
	if start >= 0 && end > start {
		return strings.TrimSpace(from[start+1 : end])
	}
	return strings.TrimSpace(from)
}

// Send delivers an HTML-only email. Kept for callers (web-auth magic links)
// that have no plain-text alternative.
func (s *Sender) Send(_ context.Context, to, subject, htmlBody string) error {
	return s.send(to, subject, "", htmlBody, "")
}

// SendMultipart delivers a multipart/alternative email (text + HTML). When
// listUnsubscribe is non-empty it is advertised via the List-Unsubscribe and
// List-Unsubscribe-Post headers (RFC 8058 one-click). A missing textBody falls
// back to HTML-only.
func (s *Sender) SendMultipart(_ context.Context, to, subject, textBody, htmlBody, listUnsubscribe string) error {
	return s.send(to, subject, textBody, htmlBody, listUnsubscribe)
}

func (s *Sender) send(to, subject, textBody, htmlBody, listUnsubscribe string) error {
	msg, err := s.buildMessage(to, subject, textBody, htmlBody, listUnsubscribe)
	if err != nil {
		return err
	}
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}
	return smtp.SendMail(s.host+":"+s.port, auth, envelopeFrom(s.from), []string{to}, msg)
}

// buildMessage assembles the RFC 5322 message. With a textBody it produces a
// multipart/alternative (text first, HTML last); without one, an HTML-only body.
func (s *Sender) buildMessage(to, subject, textBody, htmlBody, listUnsubscribe string) ([]byte, error) {
	var msg bytes.Buffer
	header := func(k, v string) { fmt.Fprintf(&msg, "%s: %s\r\n", k, v) }
	header("From", s.from)
	header("To", to)
	header("Subject", mime.QEncoding.Encode("UTF-8", subject))
	header("MIME-Version", "1.0")
	if listUnsubscribe != "" {
		header("List-Unsubscribe", "<"+listUnsubscribe+">")
		header("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
	}

	if textBody == "" {
		header("Content-Type", "text/html; charset=UTF-8")
		msg.WriteString("\r\n")
		msg.WriteString(htmlBody)
		return msg.Bytes(), nil
	}

	// multipart/alternative: least-preferred (text) first, most-preferred (HTML) last.
	mw := multipart.NewWriter(&msg)
	header("Content-Type", `multipart/alternative; boundary="`+mw.Boundary()+`"`)
	msg.WriteString("\r\n")

	for _, part := range []struct{ ctype, body string }{
		{"text/plain; charset=UTF-8", textBody},
		{"text/html; charset=UTF-8", htmlBody},
	} {
		w, err := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {part.ctype}})
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, part.body); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}
