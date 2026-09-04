package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// clock is a hand-advanced time source: the limiter refills by elapsed time,
// so driving it explicitly keeps these tests deterministic and instant.
type clock struct{ t time.Time }

func (c *clock) now() time.Time      { return c.t }
func (c *clock) add(d time.Duration) { c.t = c.t.Add(d) }

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
}

func call(t *testing.T, h http.Handler, path, addr string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	req.RemoteAddr = addr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func testConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled: true,
		Default: RateLimitPolicy{Requests: 3, Window: time.Minute},
		Auth:    RateLimitPolicy{Requests: 2, Window: time.Minute},
	}
}

func TestRateLimit_AllowsUpToQuotaThenRejects(t *testing.T) {
	c := &clock{t: time.Unix(0, 0)}
	h := rateLimitWithClock(testConfig(), c.now)(okHandler())

	for i := range 3 {
		if got := call(t, h, "/streams", "10.0.0.1:1111").Code; got != http.StatusTeapot {
			t.Fatalf("request %d: status = %d, want %d", i+1, got, http.StatusTeapot)
		}
	}

	rec := call(t, h, "/streams", "10.0.0.1:1111")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("a 429 must tell the client when to come back")
	}
	if got := rec.Header().Get("X-RateLimit-Limit"); got != "3" {
		t.Errorf("X-RateLimit-Limit = %q, want %q", got, "3")
	}
}

// Two clients must not share a bucket, or one noisy caller locks everyone out.
func TestRateLimit_IsPerClient(t *testing.T) {
	c := &clock{t: time.Unix(0, 0)}
	h := rateLimitWithClock(testConfig(), c.now)(okHandler())

	for range 3 {
		call(t, h, "/streams", "10.0.0.1:1111")
	}
	if got := call(t, h, "/streams", "10.0.0.2:2222").Code; got != http.StatusTeapot {
		t.Fatalf("second client status = %d, want %d", got, http.StatusTeapot)
	}
}

// A new ephemeral port is the same caller, not a fresh quota.
func TestRateLimit_IgnoresSourcePort(t *testing.T) {
	c := &clock{t: time.Unix(0, 0)}
	h := rateLimitWithClock(testConfig(), c.now)(okHandler())

	for range 3 {
		call(t, h, "/streams", "10.0.0.1:1111")
	}
	if got := call(t, h, "/streams", "10.0.0.1:60999").Code; got != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", got, http.StatusTooManyRequests)
	}
}

func TestRateLimit_RefillsOverTime(t *testing.T) {
	c := &clock{t: time.Unix(0, 0)}
	h := rateLimitWithClock(testConfig(), c.now)(okHandler())

	for range 3 {
		call(t, h, "/streams", "10.0.0.1:1111")
	}
	if got := call(t, h, "/streams", "10.0.0.1:1111").Code; got != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want a rejection", got)
	}

	// 3 per minute refills one token every 20s.
	c.add(21 * time.Second)
	if got := call(t, h, "/streams", "10.0.0.1:1111").Code; got != http.StatusTeapot {
		t.Fatalf("after refill status = %d, want %d", got, http.StatusTeapot)
	}
}

// The credential endpoints get the tighter tier — that is the whole point of
// having two.
func TestRateLimit_AuthRoutesUseTheStricterPolicy(t *testing.T) {
	c := &clock{t: time.Unix(0, 0)}
	h := rateLimitWithClock(testConfig(), c.now)(okHandler())

	for i := range 2 {
		if got := call(t, h, "/auth/login", "10.0.0.1:1111").Code; got != http.StatusTeapot {
			t.Fatalf("request %d: status = %d", i+1, got)
		}
	}
	if got := call(t, h, "/auth/login", "10.0.0.1:1111").Code; got != http.StatusTooManyRequests {
		t.Fatalf("third login status = %d, want %d", got, http.StatusTooManyRequests)
	}

	// The general quota is untouched: throttling sign-in must not throttle
	// browsing for the same client.
	if got := call(t, h, "/streams", "10.0.0.1:1111").Code; got != http.StatusTeapot {
		t.Fatalf("browsing status = %d, want %d", got, http.StatusTeapot)
	}
}

// A blocked probe would take the container down; a blocked alert webhook would
// lose the incident that the alert exists to report.
func TestRateLimit_ExemptsInfrastructureRoutes(t *testing.T) {
	c := &clock{t: time.Unix(0, 0)}
	h := rateLimitWithClock(testConfig(), c.now)(okHandler())

	for _, path := range []string{"/health", "/internal/alerts"} {
		for i := range 10 {
			if got := call(t, h, path, "10.0.0.1:1111").Code; got != http.StatusTeapot {
				t.Fatalf("%s request %d: status = %d, want %d", path, i+1, got, http.StatusTeapot)
			}
		}
	}
}

func TestRateLimit_DisabledIsAPassthrough(t *testing.T) {
	cfg := testConfig()
	cfg.Enabled = false
	c := &clock{t: time.Unix(0, 0)}
	h := rateLimitWithClock(cfg, c.now)(okHandler())

	for i := range 20 {
		if got := call(t, h, "/streams", "10.0.0.1:1111").Code; got != http.StatusTeapot {
			t.Fatalf("request %d: status = %d, want %d", i+1, got, http.StatusTeapot)
		}
	}
}

// A zero-valued policy means "no limit for this tier" rather than "reject
// everything" — otherwise a missing env var takes the API offline.
func TestRateLimit_ZeroPolicyDoesNotBlock(t *testing.T) {
	cfg := RateLimitConfig{Enabled: true}
	c := &clock{t: time.Unix(0, 0)}
	h := rateLimitWithClock(cfg, c.now)(okHandler())

	for i := range 20 {
		if got := call(t, h, "/auth/login", "10.0.0.1:1111").Code; got != http.StatusTeapot {
			t.Fatalf("request %d: status = %d, want %d", i+1, got, http.StatusTeapot)
		}
	}
}

func TestRateLimit_EvictsWhenTheTableIsFull(t *testing.T) {
	c := &clock{t: time.Unix(0, 0)}
	l := newLimiter(RateLimitPolicy{Requests: 1, Window: time.Minute}, c.now)

	for i := range maxTrackedClients {
		l.allow(strconv.Itoa(i))
	}
	if len(l.buckets) != maxTrackedClients {
		t.Fatalf("tracked %d clients, want %d", len(l.buckets), maxTrackedClients)
	}

	l.allow("one-too-many")
	if len(l.buckets) > maxTrackedClients {
		t.Fatalf("tracked %d clients, want the table capped at %d",
			len(l.buckets), maxTrackedClients)
	}
}
