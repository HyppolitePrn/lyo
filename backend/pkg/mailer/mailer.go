package mailer

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/hyppoliteprn/lyo/pkg/config"
)

// Sender sends transactional email.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

type smtpSender struct {
	host string
	port int
	user string
	pass string
	from string
}

// New returns an SMTP-backed Sender configured from cfg.
func New(cfg config.MailConfig) Sender {
	return &smtpSender{
		host: cfg.Host,
		port: cfg.Port,
		user: cfg.User,
		pass: cfg.Pass,
		from: cfg.From,
	}
}

func (s *smtpSender) Send(ctx context.Context, to, subject, body string) error {
	errCh := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf("%s:%d", s.host, s.port)
		// Only authenticate if credentials are configured: local dev SMTP
		// catchers (e.g. Mailpit) don't advertise the AUTH extension, and
		// smtp.SendMail errors out if an Auth is supplied to a server that
		// doesn't support it.
		var auth smtp.Auth
		if s.user != "" {
			auth = smtp.PlainAuth("", s.user, s.pass, s.host)
		}
		msg := fmt.Sprintf(
			"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
			s.from, to, subject, body,
		)
		errCh <- smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("mailer: send: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("mailer: send: %w", ctx.Err())
	}
}
