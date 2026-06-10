package smtp

import (
	"context"
	"fmt"
	"net/smtp"
)

// Sender implements application.EmailSender using net/smtp.
type Sender struct{ host, port, user, pass, from string }

// New creates an SMTP Sender. If user is empty, no authentication is attempted.
func New(host, port, user, pass, from string) *Sender {
	return &Sender{host: host, port: port, user: user, pass: pass, from: from}
}

// Send delivers a transactional HTML email.
func (s *Sender) Send(_ context.Context, to, subject, htmlBody string) error {
	addr := s.host + ":" + s.port
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, htmlBody)
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
