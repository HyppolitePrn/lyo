package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

func newAuthSvc() *auth.Service {
	return auth.NewService("test-jwt-secret-at-least-32-chars!", time.Minute, time.Hour)
}

// claimsProbe records whether claims reached the handler, and passes through.
func claimsProbe(seen **auth.Claims, called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		if c, ok := middleware.ClaimsFromContext(r.Context()); ok {
			*seen = c
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthenticate_ValidBearerTokenStoresClaims(t *testing.T) {
	svc := newAuthSvc()
	pair, err := svc.Issue("user-1", auth.RoleBroadcaster)
	if err != nil {
		t.Fatal(err)
	}

	var seen *auth.Claims
	var called bool
	h := middleware.Authenticate(svc)(claimsProbe(&seen, &called))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if seen == nil {
		t.Fatal("expected claims in context")
	}
	if seen.UserID != "user-1" || seen.Role != auth.RoleBroadcaster {
		t.Fatalf("unexpected claims: %+v", seen)
	}
}

// Authenticate is permissive by design: a missing, malformed or invalid token
// must still reach the handler, just without claims.
func TestAuthenticate_PermissiveOnBadOrMissingToken(t *testing.T) {
	svc := newAuthSvc()
	tests := map[string]string{
		"no header":     "",
		"not bearer":    "Basic dXNlcjpwYXNz",
		"invalid token": "Bearer not.a.jwt",
		"empty token":   "Bearer ",
		"wrong secret":  "Bearer " + mustIssueWithOtherSecret(t),
	}

	for name, header := range tests {
		t.Run(name, func(t *testing.T) {
			var seen *auth.Claims
			var called bool
			h := middleware.Authenticate(svc)(claimsProbe(&seen, &called))

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if !called {
				t.Fatal("next handler must still be called")
			}
			if seen != nil {
				t.Fatalf("expected no claims in context, got %+v", seen)
			}
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}
		})
	}
}

func mustIssueWithOtherSecret(t *testing.T) string {
	t.Helper()
	other := auth.NewService("a-completely-different-secret-32b!", time.Minute, time.Hour)
	pair, err := other.Issue("user-1", auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	return pair.AccessToken
}

func TestClaimsFromContext_Absent(t *testing.T) {
	if _, ok := middleware.ClaimsFromContext(context.Background()); ok {
		t.Fatal("expected no claims in a bare context")
	}
}

func TestRequireRole(t *testing.T) {
	svc := newAuthSvc()

	tests := []struct {
		name     string
		role     auth.Role // "" means unauthenticated
		min      auth.Role
		wantCode int
	}{
		{"unauthenticated", "", auth.RoleUser, http.StatusForbidden},
		{"below required role", auth.RoleUser, auth.RoleBroadcaster, http.StatusForbidden},
		{"exact role", auth.RoleBroadcaster, auth.RoleBroadcaster, http.StatusOK},
		{"above required role", auth.RoleAdmin, auth.RoleBroadcaster, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			h := middleware.Authenticate(svc)(middleware.RequireRole(tt.min)(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					called = true
					w.WriteHeader(http.StatusOK)
				}),
			))

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			if tt.role != "" {
				pair, err := svc.Issue("user-1", tt.role)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if called != (tt.wantCode == http.StatusOK) {
				t.Fatalf("next called = %v, want %v", called, tt.wantCode == http.StatusOK)
			}
		})
	}
}

// TestAuthenticate_RecordsRejectedToken: the middleware stays permissive — it
// never rejects — but a handler must still be able to tell a caller who never
// logged in from one whose session ran out.
func TestAuthenticate_RecordsRejectedToken(t *testing.T) {
	svc := auth.NewService("test-jwt-secret-at-least-32-chars!", time.Minute, time.Hour)

	tests := map[string]struct {
		header       string
		wantRejected bool
		wantClaims   bool
	}{
		"no header":     {"", false, false},
		"invalid token": {"Bearer not.a.valid.jwt", true, false},
		"non-bearer":    {"Basic abc123", false, false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var gotRejected, gotClaims bool
			h := middleware.Authenticate(svc)(http.HandlerFunc(
				func(_ http.ResponseWriter, r *http.Request) {
					gotRejected = middleware.TokenRejected(r.Context())
					_, gotClaims = middleware.ClaimsFromContext(r.Context())
				}))

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			// Permissive in every case: the request always reaches the handler.
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, the middleware must never reject", rec.Code)
			}
			if gotRejected != tc.wantRejected {
				t.Errorf("TokenRejected = %v, want %v", gotRejected, tc.wantRejected)
			}
			if gotClaims != tc.wantClaims {
				t.Errorf("claims present = %v, want %v", gotClaims, tc.wantClaims)
			}
		})
	}
}

func TestAuthenticate_ValidTokenIsNotMarkedRejected(t *testing.T) {
	svc := auth.NewService("test-jwt-secret-at-least-32-chars!", time.Minute, time.Hour)
	pair, err := svc.Issue("user-1", auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	var rejected, hasClaims bool
	h := middleware.Authenticate(svc)(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			rejected = middleware.TokenRejected(r.Context())
			_, hasClaims = middleware.ClaimsFromContext(r.Context())
		}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if rejected {
		t.Error("a valid token must not be marked rejected")
	}
	if !hasClaims {
		t.Error("expected claims in the context")
	}
}
