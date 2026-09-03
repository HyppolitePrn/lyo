package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/hyppoliteprn/lyo/internal/auth"
)

type contextKey string

const (
	claimsKey        contextKey = "claims"
	tokenRejectedKey contextKey = "token_rejected"
)

// Authenticate extracts and validates the Bearer token.
// It stores the claims in the request context for downstream handlers.
func Authenticate(svc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := svc.Verify(token)
			if err != nil {
				// Still permissive (ADR 007) — the request continues as
				// anonymous. But record that a token *was* presented and
				// failed, so a handler can answer 401 "your session is over"
				// instead of 403 "you lack the role", which is what an expired
				// token otherwise looks like to the caller.
				ctx := context.WithValue(r.Context(), tokenRejectedKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns a middleware that rejects requests below the given role.
func RequireRole(min auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || !claims.Role.AtLeast(min) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// TokenRejected reports whether the request carried a Bearer token that failed
// verification — expired, tampered with, or signed by another key. It is the
// only way to tell "logged out" from "never logged in", since Authenticate
// treats both as anonymous.
func TokenRejected(ctx context.Context) bool {
	rejected, _ := ctx.Value(tokenRejectedKey).(bool)
	return rejected
}

// ClaimsFromContext retrieves the JWT claims stored by Authenticate.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*auth.Claims)
	return c, ok
}
