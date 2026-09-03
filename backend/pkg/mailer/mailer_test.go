package mailer_test

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hyppoliteprn/lyo/pkg/config"
	"github.com/hyppoliteprn/lyo/pkg/mailer"
)

// fakeSMTP is a minimal SMTP server that speaks just enough of the protocol for
// net/smtp's SendMail to complete, and records the transcript it received.
type fakeSMTP struct {
	host       string
	port       int
	mu         sync.Mutex
	transcript strings.Builder
	// advertiseAuth controls whether the EHLO reply lists AUTH PLAIN.
	advertiseAuth bool
}

func startFakeSMTP(t *testing.T, advertiseAuth bool) *fakeSMTP {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	addr := ln.Addr().(*net.TCPAddr)
	s := &fakeSMTP{host: "127.0.0.1", port: addr.Port, advertiseAuth: advertiseAuth}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()
	return s
}

func (s *fakeSMTP) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }

	write("220 fake ESMTP")
	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			s.mu.Lock()
			s.transcript.WriteString(line + "\n")
			s.mu.Unlock()
			if line == "." {
				inData = false
				write("250 OK")
			}
			continue
		}

		switch {
		case strings.HasPrefix(line, "EHLO"):
			if s.advertiseAuth {
				write("250-fake")
				write("250 AUTH PLAIN")
			} else {
				write("250 fake")
			}
		case strings.HasPrefix(line, "HELO"):
			write("250 fake")
		case strings.HasPrefix(line, "AUTH"):
			write("235 authenticated")
		case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
			s.mu.Lock()
			s.transcript.WriteString(line + "\n")
			s.mu.Unlock()
			write("250 OK")
		case line == "DATA":
			inData = true
			write("354 send data")
		case line == "QUIT":
			write("221 bye")
			return
		default:
			write("250 OK")
		}
	}
}

func (s *fakeSMTP) text() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transcript.String()
}

func (s *fakeSMTP) cfg(user, pass string) config.MailConfig {
	return config.MailConfig{
		Host: s.host,
		Port: s.port,
		User: user,
		Pass: pass,
		From: "noreply@lyo.app",
	}
}

func TestSend_DeliversMessageWithoutAuth(t *testing.T) {
	srv := startFakeSMTP(t, false)

	err := mailer.New(srv.cfg("", "")).Send(context.Background(), "alice@example.com", "Reset your Lyo password", "click here")
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	got := srv.text()
	for _, want := range []string{
		"MAIL FROM:<noreply@lyo.app>",
		"RCPT TO:<alice@example.com>",
		"Subject: Reset your Lyo password",
		"To: alice@example.com",
		"click here",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("transcript missing %q:\n%s", want, got)
		}
	}
}

func TestSend_AuthenticatesWhenCredentialsAreConfigured(t *testing.T) {
	srv := startFakeSMTP(t, true)

	err := mailer.New(srv.cfg("smtp-user", "smtp-pass")).Send(context.Background(), "bob@example.com", "hi", "body")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(srv.text(), "RCPT TO:<bob@example.com>") {
		t.Errorf("message was not delivered:\n%s", srv.text())
	}
}

func TestSend_WrapsTransportError(t *testing.T) {
	// Nothing is listening on this port.
	cfg := config.MailConfig{Host: "127.0.0.1", Port: 1, From: "noreply@lyo.app"}

	err := mailer.New(cfg).Send(context.Background(), "alice@example.com", "hi", "body")
	if err == nil {
		t.Fatal("expected a send error")
	}
	if !strings.HasPrefix(err.Error(), "mailer: send: ") {
		t.Errorf("error not wrapped: %v", err)
	}
}

func TestSend_HonoursContextCancellation(t *testing.T) {
	// A listener that accepts but never speaks makes SendMail block on the greeting.
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Hold the connection open without sending the 220 greeting.
		time.Sleep(2 * time.Second)
		_ = conn.Close()
	}()

	addr := ln.Addr().(*net.TCPAddr)
	cfg := config.MailConfig{Host: "127.0.0.1", Port: addr.Port, From: "noreply@lyo.app"}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = mailer.New(cfg).Send(ctx, "alice@example.com", "hi", "body")
	if err == nil {
		t.Fatal("expected the send to be cut short by the context")
	}
	if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Errorf("expected a deadline error, got %v (port %s)", err, strconv.Itoa(addr.Port))
	}
}
