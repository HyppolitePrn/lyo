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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hyppoliteprn/lyo/internal/api"
	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/passwordreset"
	"github.com/hyppoliteprn/lyo/internal/streaming"
	"github.com/hyppoliteprn/lyo/internal/track"
	"github.com/hyppoliteprn/lyo/internal/user"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
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

type mockGetUserByIDUsecase struct {
	executeFn func(ctx context.Context, id string) (*user.User, error)
}

func (m *mockGetUserByIDUsecase) Execute(ctx context.Context, id string) (*user.User, error) {
	return m.executeFn(ctx, id)
}

type mockUpdateUserByIDUsecase struct {
	executeFn func(ctx context.Context, id string, username, email *string) (*user.User, error)
}

func (m *mockUpdateUserByIDUsecase) Execute(ctx context.Context, id string, username, email *string) (*user.User, error) {
	return m.executeFn(ctx, id, username, email)
}

func newTestAuthSvc() *auth.Service {
	return auth.NewService("test-jwt-secret-at-least-32-chars!", time.Minute, time.Hour)
}

// nopStreamSvc satisfies api.StreamService for tests that don't exercise streaming.
type nopStreamSvc struct{}

func (nopStreamSvc) StartStream(_ context.Context, _, _, _ string) (*streaming.Stream, error) {
	return nil, errors.New("not implemented")
}
func (nopStreamSvc) EndStream(_ context.Context, _, _ string) (*streaming.Stream, error) {
	return nil, errors.New("not implemented")
}
func (nopStreamSvc) GetStream(_ context.Context, _ string) (*streaming.Stream, error) {
	return nil, errors.New("not implemented")
}
func (nopStreamSvc) ListLiveStreams(_ context.Context) ([]streaming.Stream, error) {
	return nil, errors.New("not implemented")
}

// nopFeatureSvc satisfies api.FeatureService for tests that don't exercise feature flags.
type nopFeatureSvc struct{}

func (nopFeatureSvc) IsEnabled(_ context.Context, _ string) bool { return true }

// mockPasswordResetService implements api.PasswordResetService for handler tests.
type mockPasswordResetService struct {
	forgotPasswordFn func(ctx context.Context, email string) error
	resetPasswordFn  func(ctx context.Context, token, password string) error
}

func (m *mockPasswordResetService) ForgotPassword(ctx context.Context, email string) error {
	return m.forgotPasswordFn(ctx, email)
}
func (m *mockPasswordResetService) ResetPassword(ctx context.Context, token, password string) error {
	return m.resetPasswordFn(ctx, token, password)
}

// nopTrackSvc satisfies api.TrackService for tests that don't exercise tracks.
type nopTrackSvc struct{}

func (nopTrackSvc) ListTracks(_ context.Context, _ string, _, _ int) ([]track.Track, error) {
	return nil, errors.New("not implemented")
}
func (nopTrackSvc) CreateTrack(_ context.Context, _, _, _, _ string, _ int) (*track.Track, error) {
	return nil, errors.New("not implemented")
}
func (nopTrackSvc) GetTrack(_ context.Context, _ string) (*track.Track, error) {
	return nil, errors.New("not implemented")
}
func (nopTrackSvc) DeleteTrack(_ context.Context, _, _ string) error {
	return errors.New("not implemented")
}
func (nopTrackSvc) PresignUpload(_ context.Context, _, _ string) (string, string, string, error) {
	return "", "", "", errors.New("not implemented")
}

func newTestRouter(
	userSvc api.UserService,
	authSvc *auth.Service,
	getUserByIDUC api.GetUserByIDUsecase,
	updateUserByIDUC api.UpdateUserByIDUsecase,
	pwResetSvc api.PasswordResetService,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Authenticate(authSvc))
	strict := api.NewStrictHandlerWithOptions(
		api.NewHandlers(userSvc, authSvc, nopStreamSvc{}, nopFeatureSvc{}, pwResetSvc, nopTrackSvc{}, getUserByIDUC, updateUserByIDUC, slog.New(slog.NewTextHandler(io.Discard, nil))),
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

func get(t *testing.T, h http.Handler, path, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func patch(t *testing.T, h http.Handler, path, bearerToken, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
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
	w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/register",
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
	w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/register",
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
	w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/login",
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
	w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/login",
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
	w := post(t, newTestRouter(nil, authSvc, nil, nil, nil), "/auth/refresh",
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
	w := post(t, newTestRouter(nil, authSvc, nil, nil, nil), "/auth/refresh",
		`{"refresh_token":"this.is.not.a.valid.token"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body)
	}
}

// ── ForgotPassword / ResetPassword ───────────────────────────────────────────

func TestForgotPasswordHandler_AlwaysReturns200(t *testing.T) {
	authSvc := newTestAuthSvc()

	for _, email := range []string{"alice@example.com", "unknown@example.com"} {
		pwSvc := &mockPasswordResetService{
			forgotPasswordFn: func(_ context.Context, _ string) error { return nil },
		}
		w := post(t, newTestRouter(nil, authSvc, nil, nil, pwSvc), "/auth/forgot-password",
			`{"email":"`+email+`"}`)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d: %s", email, w.Code, w.Body)
		}
	}
}

func TestResetPasswordHandler_InvalidToken_Returns400(t *testing.T) {
	authSvc := newTestAuthSvc()
	pwSvc := &mockPasswordResetService{
		resetPasswordFn: func(_ context.Context, _, _ string) error {
			return passwordreset.ErrInvalidToken
		},
	}
	w := post(t, newTestRouter(nil, authSvc, nil, nil, pwSvc), "/auth/reset-password",
		`{"token":"bad-token","password":"newsecret123"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body)
	}
}

func TestResetPasswordHandler_Success(t *testing.T) {
	authSvc := newTestAuthSvc()
	pwSvc := &mockPasswordResetService{
		resetPasswordFn: func(_ context.Context, _, _ string) error { return nil },
	}
	w := post(t, newTestRouter(nil, authSvc, nil, nil, pwSvc), "/auth/reset-password",
		`{"token":"good-token","password":"newsecret123"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
}

func TestGetUserByIDHandler_Success(t *testing.T) {
	authSvc := newTestAuthSvc()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	getUserByIDUC := &mockGetUserByIDUsecase{
		executeFn: func(_ context.Context, id string) (*user.User, error) {
			if id != userID.String() {
				t.Fatalf("expected lookup for %s, got %s", userID, id)
			}
			return &user.User{
				ID:                  pgtype.UUID{Bytes: userID, Valid: true},
				Username:            "alice",
				Email:               "alice@example.com",
				Role:                auth.RoleUser,
				FavoriteTrackIDs:    []pgtype.UUID{},
				FavoriteStreamIDs:   []pgtype.UUID{},
				FavoritePlaylistIDs: []pgtype.UUID{},
				CreatedAt:           now,
				UpdatedAt:           now,
			}, nil
		},
	}

	pair, err := authSvc.Issue(userID.String(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	w := get(t, newTestRouter(nil, authSvc, getUserByIDUC, nil, nil), "/users/me", pair.AccessToken)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var got api.User
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Username != "alice" || string(got.Email) != "alice@example.com" || got.Role != api.Role(auth.RoleUser) {
		t.Fatalf("unexpected user payload: %+v", got)
	}
}

func TestGetUserByIDHandler_Unauthenticated(t *testing.T) {
	authSvc := newTestAuthSvc()

	w := get(t, newTestRouter(nil, authSvc, nil, nil, nil), "/users/me", "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body)
	}
}

// ── UpdateUserByID ────────────────────────────────────────────────────────────

func TestUpdateUserByIDHandler_Success(t *testing.T) {
	authSvc := newTestAuthSvc()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	updateUserByIDUC := &mockUpdateUserByIDUsecase{
		executeFn: func(_ context.Context, id string, username, email *string) (*user.User, error) {
			if id != userID.String() {
				t.Fatalf("expected update for %s, got %s", userID, id)
			}
			if username == nil || *username != "newname" {
				t.Fatalf("expected username %q, got %v", "newname", username)
			}
			if email != nil {
				t.Fatalf("expected nil email, got %v", *email)
			}
			return &user.User{
				ID:                  pgtype.UUID{Bytes: userID, Valid: true},
				Username:            "newname",
				Email:               "alice@example.com",
				Role:                auth.RoleUser,
				FavoriteTrackIDs:    []pgtype.UUID{},
				FavoriteStreamIDs:   []pgtype.UUID{},
				FavoritePlaylistIDs: []pgtype.UUID{},
				CreatedAt:           now,
				UpdatedAt:           now,
			}, nil
		},
	}

	pair, err := authSvc.Issue(userID.String(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	w := patch(t, newTestRouter(nil, authSvc, nil, updateUserByIDUC, nil), "/users/me", pair.AccessToken,
		`{"username":"newname"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var got api.User
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Username != "newname" {
		t.Fatalf("unexpected user payload: %+v", got)
	}
}

func TestUpdateUserByIDHandler_Unauthenticated(t *testing.T) {
	authSvc := newTestAuthSvc()

	w := patch(t, newTestRouter(nil, authSvc, nil, nil, nil), "/users/me", "", `{"username":"newname"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body)
	}
}

func TestUpdateUserByIDHandler_DuplicateEmail(t *testing.T) {
	authSvc := newTestAuthSvc()
	userID := uuid.New()

	updateUserByIDUC := &mockUpdateUserByIDUsecase{
		executeFn: func(_ context.Context, _ string, _, _ *string) (*user.User, error) {
			return nil, &pgconn.PgError{Code: "23505"}
		},
	}

	pair, err := authSvc.Issue(userID.String(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	w := patch(t, newTestRouter(nil, authSvc, nil, updateUserByIDUC, nil), "/users/me", pair.AccessToken,
		`{"email":"taken@example.com"}`)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body)
	}
}
