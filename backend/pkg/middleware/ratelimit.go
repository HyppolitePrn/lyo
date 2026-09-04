package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitPolicy allows Requests calls per Window, per client.
//
// It is enforced as a token bucket rather than a fixed window: the bucket
// holds Requests tokens and refills at Requests/Window, so a client may still
// burst up to the quota (a mobile screen firing several calls as it opens is
// normal traffic) while a sustained caller is held to the average rate. A
// fixed window would instead let a caller spend the whole quota at the end of
// one window and again at the start of the next — twice the intended rate
// across the boundary, which is precisely the shape of a credential-stuffing
// run.
type RateLimitPolicy struct {
	Requests int
	Window   time.Duration
}

func (p RateLimitPolicy) enabled() bool {
	return p.Requests > 0 && p.Window > 0
}

// RateLimitConfig holds the two tiers the API enforces.
type RateLimitConfig struct {
	Enabled bool
	// Default applies to every request that is not exempt.
	Default RateLimitPolicy
	// Auth applies instead of Default to the credential endpoints under
	// /auth/. It is deliberately much tighter: those are the only routes
	// where guessing repeatedly is worth anything to an attacker.
	Auth RateLimitPolicy
}

// maxTrackedClients caps the bucket map. Every distinct client address costs
// an entry, so without a ceiling a spoofed-source flood would turn the limiter
// itself into the memory exhaustion it exists to prevent. Reaching the cap
// drops the whole table: buckets are pure soft state, and refilling from empty
// only ever grants clients tokens they were about to earn anyway.
const maxTrackedClients = 50_000

type bucket struct {
	tokens float64
	last   time.Time
}

type limiter struct {
	policy  RateLimitPolicy
	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time
}

func newLimiter(policy RateLimitPolicy, now func() time.Time) *limiter {
	return &limiter{policy: policy, buckets: map[string]*bucket{}, now: now}
}

// allow consumes one token for key. It returns whether the call is allowed,
// how many tokens remain, and how long until the next one is available.
func (l *limiter) allow(key string) (ok bool, remaining int, retryAfter time.Duration) {
	refillPerSec := float64(l.policy.Requests) / l.policy.Window.Seconds()
	capacity := float64(l.policy.Requests)
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, seen := l.buckets[key]
	switch {
	case seen:
		elapsed := now.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens = min(capacity, b.tokens+elapsed*refillPerSec)
			b.last = now
		}
	default:
		if len(l.buckets) >= maxTrackedClients {
			clear(l.buckets)
		}
		b = &bucket{tokens: capacity, last: now}
		l.buckets[key] = b
	}

	if b.tokens < 1 {
		// Seconds until the bucket holds a whole token again.
		wait := time.Duration((1 - b.tokens) / refillPerSec * float64(time.Second))
		return false, 0, max(wait, time.Second)
	}

	b.tokens--
	return true, int(b.tokens), 0
}

// RateLimit rejects clients that exceed their quota with 429.
//
// The client key is the request's remote address. Behind the TLS reverse proxy
// that is the proxy for every caller, so this middleware is only meaningful
// when chi's RealIP has already rewritten RemoteAddr from X-Forwarded-For —
// which main.go wires in exactly when TRUSTED_PROXY says a proxy is in front.
// On a directly exposed socket the header is caller-controlled, and trusting
// it would let an attacker mint a fresh quota per request.
func RateLimit(cfg RateLimitConfig) func(http.Handler) http.Handler {
	return rateLimitWithClock(cfg, time.Now)
}

func rateLimitWithClock(cfg RateLimitConfig, now func() time.Time) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	def := newLimiter(cfg.Default, now)
	authL := newLimiter(cfg.Auth, now)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := def
			if isAuthRoute(r.URL.Path) {
				l = authL
			}
			if isRateLimitExempt(r.URL.Path) || !l.policy.enabled() {
				next.ServeHTTP(w, r)
				return
			}

			key := clientKey(r)
			ok, remaining, retryAfter := l.allow(key)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.policy.Requests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			if !ok {
				secs := int(retryAfter.Round(time.Second).Seconds())
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				writeRateLimited(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isAuthRoute reports whether path is one of the credential endpoints.
// /auth/refresh is included: a refresh token is a credential too, and it is
// the one an attacker holds after stealing a stale token.
func isAuthRoute(path string) bool {
	return strings.HasPrefix(path, "/auth/")
}

// isRateLimitExempt covers the two callers that are infrastructure rather than
// users: the container/uptime health probe, which polls far above any sane
// per-client quota, and Grafana's alert webhook, which authenticates with a
// shared secret and is 404 at the edge anyway.
func isRateLimitExempt(path string) bool {
	return path == "/health" || strings.HasPrefix(path, "/internal/")
}

// clientKey identifies the caller. The port is stripped so that a client's
// successive connections share one bucket instead of getting a fresh quota
// with every new ephemeral port.
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// writeRateLimited answers in the API's Error schema ({code, message}) so a
// throttled client parses this the same way it parses every other failure.
func writeRateLimited(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    http.StatusTooManyRequests,
		"message": "too many requests",
	})
}
