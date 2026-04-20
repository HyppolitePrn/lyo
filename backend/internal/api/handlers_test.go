package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hyppoliteprn/lyo/internal/api"
	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/user"
)

// mockUserService implements api.UserService for handler tests.
type mockUserService struct {
	registerFn func(ctx context.Context, username, email, password string) (auth.TokenPair, error)
	loginFn    func(ctx context.Context, email, password string) (auth.TokenPair, error)
}

func (m *mockUserService) Register(ctx context.Context, username, email, password string) (auth.TokenPair, error) {
	return m.registerFn(ctx, username, email, password)
}
func (m *mockUserService) Login(ctx context.Context, email, password string) (auth.TokenPair, error) {
	return m.loginFn(ctx, email, password)
}

func newTestAuthSvc() *auth.Service {
	return auth.NewService("test-jwt-secret-at-least-32-chars!", time.Minute, time.Hour)
}

func newTestRouter(userSvc api.UserService, authSvc *auth.Service) http.Handler {
	r := chi.NewRouter()
	strict := api.NewStrictHandlerWithOptions(
		api.NewHandlers(userSvc, authSvc, slog.New(slog.NewTextHandler(io.Discard, nil))),
		nil,
		api.StrictHTTPServerOptions{
			ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
				if he, ok := errors.AsType[*api.HTTPError](err); ok {
					http.Error(w, he.Msg, he.Code)
					return
				}
				http.Error(w, "internal error", http.StatusInternalServerError)
			},
			RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			},
		},
	)
	api.HandlerFromMux(strict, r)
	return r
}

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decodeTokens(t *testing.T, w *httptest.ResponseRecorder) (accessToken, refreshToken string) {
	t.Helper()
	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.AccessToken, resp.RefreshToken
}

// ── Register ──────────────────────────────────────────────────────────────────

func TestRegisterHandler_Success(t *testing.T) {
	authSvc := newTestAuthSvc()
	svc := &mockUserService{
		registerFn: func(_ context.Context, _, _, _ string) (auth.TokenPair, error) {
			return authSvc.Issue("user-1", auth.RoleUser)
		},
	}
	w := post(t, newTestRouter(svc, authSvc), "/auth/register",
		`{"username":"alice","email":"alice@example.com","password":"secret123"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body)
	}
	access, refresh := decodeTokens(t, w)
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	authSvc := newTestAuthSvc()
	svc := &mockUserService{
		registerFn: func(_ context.Context, _, _, _ string) (auth.TokenPair, error) {
			return auth.TokenPair{}, &pgconn.PgError{Code: "23505"}
		},
	}
	w := post(t, newTestRouter(svc, authSvc), "/auth/register",
		`{"username":"alice","email":"alice@example.com","password":"secret123"}`)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body)
	}
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestLoginHandler_Success(t *testing.T) {
	authSvc := newTestAuthSvc()
	svc := &mockUserService{
		loginFn: func(_ context.Context, _, _ string) (auth.TokenPair, error) {
			return authSvc.Issue("user-1", auth.RoleUser)
		},
	}
	w := post(t, newTestRouter(svc, authSvc), "/auth/login",
		`{"email":"alice@example.com","password":"secret123"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	access, refresh := decodeTokens(t, w)
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	authSvc := newTestAuthSvc()
	svc := &mockUserService{
		loginFn: func(_ context.Context, _, _ string) (auth.TokenPair, error) {
			return auth.TokenPair{}, user.ErrInvalidCredentials
		},
	}
	w := post(t, newTestRouter(svc, authSvc), "/auth/login",
		`{"email":"alice@example.com","password":"wrong"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body)
	}
}

// ── RefreshToken ──────────────────────────────────────────────────────────────

func TestRefreshTokenHandler_Success(t *testing.T) {
	authSvc := newTestAuthSvc()
	pair, err := authSvc.Issue("user-1", auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	w := post(t, newTestRouter(nil, authSvc), "/auth/refresh",
		`{"refresh_token":"`+pair.RefreshToken+`"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	access, refresh := decodeTokens(t, w)
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}
}

func TestRefreshTokenHandler_InvalidToken(t *testing.T) {
	authSvc := newTestAuthSvc()
	w := post(t, newTestRouter(nil, authSvc), "/auth/refresh",
		`{"refresh_token":"this.is.not.a.valid.token"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body)
	}
}
