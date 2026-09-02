package middleware_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

type logLine struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

func runLogger(t *testing.T, req *http.Request, next http.Handler) logLine {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	middleware.Logger(logger)(next).ServeHTTP(httptest.NewRecorder(), req)

	var line logLine
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
		t.Fatalf("decode log line %q: %v", buf.String(), err)
	}
	return line
}

func TestLogger_RecordsExplicitStatus(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/tracks", nil)
	line := runLogger(t, req, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	if line.Msg != "request" || line.Method != http.MethodPost || line.Path != "/tracks" {
		t.Fatalf("unexpected log line: %+v", line)
	}
	if line.Status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", line.Status, http.StatusCreated)
	}
}

// A handler that never calls WriteHeader implicitly returned 200.
func TestLogger_DefaultsToOKWhenHeaderNeverWritten(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	line := runLogger(t, req, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	if line.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", line.Status, http.StatusOK)
	}
}

// A hijacked WebSocket upgrade never calls WriteHeader either, but 200 would be
// misleading — it must be logged as 101.
func TestLogger_ReportsSwitchingProtocolsForWebSocketUpgrade(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/streams/1/listen", nil)
	req.Header.Set("Upgrade", "WebSocket") // case-insensitive on purpose
	line := runLogger(t, req, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	if line.Status != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want %d", line.Status, http.StatusSwitchingProtocols)
	}
}

// hijackableRecorder is an httptest.ResponseRecorder that also supports Hijack.
type hijackableRecorder struct {
	*httptest.ResponseRecorder
	hijacked bool
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, nil
}

func TestResponseRecorder_HijackDelegatesToUnderlyingWriter(t *testing.T) {
	rec := &hijackableRecorder{ResponseRecorder: httptest.NewRecorder()}
	var err error
	h := middleware.Logger(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("recorder must expose http.Hijacker")
			}
			_, _, err = hj.Hijack()
		}),
	)
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	if err != nil {
		t.Fatalf("hijack: %v", err)
	}
	if !rec.hijacked {
		t.Fatal("expected hijack to reach the underlying ResponseWriter")
	}
}

func TestResponseRecorder_HijackFailsWhenUnsupported(t *testing.T) {
	var err error
	h := middleware.Logger(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _, err = w.(http.Hijacker).Hijack()
		}),
	)
	// httptest.ResponseRecorder does not implement http.Hijacker.
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	if err == nil || err.Error() != "underlying ResponseWriter does not support hijacking" {
		t.Fatalf("unexpected error: %v", err)
	}
}
