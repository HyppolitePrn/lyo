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
	"github.com/hyppoliteprn/lyo/internal/features"
	"github.com/hyppoliteprn/lyo/internal/passwordreset"
	"github.com/hyppoliteprn/lyo/internal/playlist"
	"github.com/hyppoliteprn/lyo/internal/streaming"
	"github.com/hyppoliteprn/lyo/internal/track"
	"github.com/hyppoliteprn/lyo/internal/user"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

// mockUserService implements api.UserService for handler tests.
type mockUserService struct {
	registerFn func(ctx context.Context, username, email, password string) (auth.TokenPair, error)
	loginFn    func(ctx context.Context, email, password string) (auth.TokenPair, error)
	getByIDFn  func(ctx context.Context, id string) (*user.User, error)
}

func (m *mockUserService) Register(ctx context.Context, username, email, password string) (auth.TokenPair, error) {
	return m.registerFn(ctx, username, email, password)
}
func (m *mockUserService) Login(ctx context.Context, email, password string) (auth.TokenPair, error) {
	return m.loginFn(ctx, email, password)
}
func (m *mockUserService) GetByID(ctx context.Context, id string) (*user.User, error) {
	if m.getByIDFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.getByIDFn(ctx, id)
}
func (m *mockUserService) AddFavoriteTrack(_ context.Context, _, _ string) error    { return nil }
func (m *mockUserService) RemoveFavoriteTrack(_ context.Context, _, _ string) error { return nil }
func (m *mockUserService) AddFavoriteStream(_ context.Context, _, _ string) error   { return nil }
func (m *mockUserService) RemoveFavoriteStream(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockUserService) AddFavoritePlaylist(_ context.Context, _, _ string) error { return nil }
func (m *mockUserService) RemoveFavoritePlaylist(_ context.Context, _, _ string) error {
	return nil
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
func (nopFeatureSvc) All(_ context.Context) ([]features.Flag, error) {
	return nil, errors.New("not implemented")
}
func (nopFeatureSvc) Toggle(_ context.Context, _ string, _ bool) (*features.Flag, error) {
	return nil, errors.New("not implemented")
}

// mockFeatureSvc implements api.FeatureService with overridable behavior for admin handler tests.
type mockFeatureSvc struct {
	allFn    func(ctx context.Context) ([]features.Flag, error)
	toggleFn func(ctx context.Context, name string, enabled bool) (*features.Flag, error)
}

func (m *mockFeatureSvc) IsEnabled(_ context.Context, _ string) bool { return true }
func (m *mockFeatureSvc) All(ctx context.Context) ([]features.Flag, error) {
	return m.allFn(ctx)
}
func (m *mockFeatureSvc) Toggle(ctx context.Context, name string, enabled bool) (*features.Flag, error) {
	return m.toggleFn(ctx, name, enabled)
}

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

// nopPlaylistSvc satisfies api.PlaylistService for tests that don't exercise playlists.
type nopPlaylistSvc struct{}

func (nopPlaylistSvc) ListByOwner(_ context.Context, _ string) ([]playlist.Playlist, error) {
	return nil, errors.New("not implemented")
}
func (nopPlaylistSvc) Create(_ context.Context, _, _, _ string, _ bool) (*playlist.Playlist, error) {
	return nil, errors.New("not implemented")
}
func (nopPlaylistSvc) Get(_ context.Context, _ string) (*playlist.Playlist, error) {
	return nil, errors.New("not implemented")
}
func (nopPlaylistSvc) Update(_ context.Context, _, _ string, _, _ *string, _ *bool) (*playlist.Playlist, error) {
	return nil, errors.New("not implemented")
}
func (nopPlaylistSvc) Delete(_ context.Context, _, _ string) error {
	return errors.New("not implemented")
}
func (nopPlaylistSvc) AddTrack(_ context.Context, _, _, _ string) (*playlist.Playlist, error) {
	return nil, errors.New("not implemented")
}
func (nopPlaylistSvc) RemoveTrack(_ context.Context, _, _, _ string) (*playlist.Playlist, error) {
	return nil, errors.New("not implemented")
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
		api.NewHandlers(userSvc, authSvc, nopStreamSvc{}, nopFeatureSvc{}, pwResetSvc, nopTrackSvc{}, nopPlaylistSvc{}, getUserByIDUC, updateUserByIDUC, nil, nil,
			slog.New(slog.NewTextHandler(io.Discard, nil))),
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

func newTestRouterWithFeatureSvc(authSvc *auth.Service, featureSvc api.FeatureService) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Authenticate(authSvc))
	strict := api.NewStrictHandlerWithOptions(
		api.NewHandlers(nil, authSvc, nopStreamSvc{}, featureSvc, nil, nopTrackSvc{}, nopPlaylistSvc{}, nil, nil, nil, nil,
			slog.New(slog.NewTextHandler(io.Discard, nil))),
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
	svc := &mockUserService{getByIDFn: func(context.Context, string) (*user.User, error) {
		return &user.User{Role: auth.RoleUser}, nil
	}}
	w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/refresh",
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

// ── Timeout handling ──────────────────────────────────────────────────────────

// A context deadline anywhere below the handler must surface as 503, never 500:
// the client should retry, not treat it as a bug.
func TestAuthHandlers_TimeoutIs503(t *testing.T) {
	authSvc := newTestAuthSvc()
	userID := uuid.New()

	t.Run("register", func(t *testing.T) {
		svc := &mockUserService{
			registerFn: func(context.Context, string, string, string) (auth.TokenPair, error) {
				return auth.TokenPair{}, context.DeadlineExceeded
			},
		}
		w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/register",
			`{"username":"alice","email":"alice@example.com","password":"secret123"}`)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
		}
	})

	t.Run("login", func(t *testing.T) {
		svc := &mockUserService{
			loginFn: func(context.Context, string, string) (auth.TokenPair, error) {
				return auth.TokenPair{}, context.DeadlineExceeded
			},
		}
		w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/login",
			`{"email":"alice@example.com","password":"secret123"}`)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
		}
	})

	t.Run("forgot password", func(t *testing.T) {
		pwSvc := &mockPasswordResetService{
			forgotPasswordFn: func(context.Context, string) error { return context.DeadlineExceeded },
		}
		w := post(t, newTestRouter(nil, authSvc, nil, nil, pwSvc), "/auth/forgot-password",
			`{"email":"alice@example.com"}`)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
		}
	})

	t.Run("reset password", func(t *testing.T) {
		pwSvc := &mockPasswordResetService{
			resetPasswordFn: func(context.Context, string, string) error { return context.DeadlineExceeded },
		}
		w := post(t, newTestRouter(nil, authSvc, nil, nil, pwSvc), "/auth/reset-password",
			`{"token":"t","password":"newsecret123"}`)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
		}
	})

	pair, err := authSvc.Issue(userID.String(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get user", func(t *testing.T) {
		uc := &mockGetUserByIDUsecase{
			executeFn: func(context.Context, string) (*user.User, error) { return nil, context.DeadlineExceeded },
		}
		w := get(t, newTestRouter(nil, authSvc, uc, nil, nil), "/users/me", pair.AccessToken)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
		}
	})

	t.Run("update user", func(t *testing.T) {
		uc := &mockUpdateUserByIDUsecase{
			executeFn: func(context.Context, string, *string, *string) (*user.User, error) {
				return nil, context.DeadlineExceeded
			},
		}
		w := patch(t, newTestRouter(nil, authSvc, nil, uc, nil), "/users/me", pair.AccessToken, `{"username":"x"}`)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
		}
	})
}

// ── Unexpected service failures ───────────────────────────────────────────────

func TestAuthHandlers_UnexpectedErrorIs500(t *testing.T) {
	authSvc := newTestAuthSvc()
	sentinel := errors.New("db down")

	t.Run("register", func(t *testing.T) {
		svc := &mockUserService{
			registerFn: func(context.Context, string, string, string) (auth.TokenPair, error) {
				return auth.TokenPair{}, sentinel
			},
		}
		w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/register",
			`{"username":"alice","email":"alice@example.com","password":"secret123"}`)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
		}
	})

	t.Run("login", func(t *testing.T) {
		svc := &mockUserService{
			loginFn: func(context.Context, string, string) (auth.TokenPair, error) {
				return auth.TokenPair{}, sentinel
			},
		}
		w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/login",
			`{"email":"alice@example.com","password":"secret123"}`)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
		}
	})

	t.Run("reset password", func(t *testing.T) {
		pwSvc := &mockPasswordResetService{
			resetPasswordFn: func(context.Context, string, string) error { return sentinel },
		}
		w := post(t, newTestRouter(nil, authSvc, nil, nil, pwSvc), "/auth/reset-password",
			`{"token":"t","password":"newsecret123"}`)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
		}
	})
}

// A ForgotPassword failure is logged but never surfaced: reporting it would let
// a caller probe which addresses exist.
func TestForgotPasswordHandler_HidesServiceFailure(t *testing.T) {
	pwSvc := &mockPasswordResetService{
		forgotPasswordFn: func(context.Context, string) error { return errors.New("smtp down") },
	}
	w := post(t, newTestRouter(nil, newTestAuthSvc(), nil, nil, pwSvc), "/auth/forgot-password",
		`{"email":"alice@example.com"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body)
	}
}

// ── Not found ─────────────────────────────────────────────────────────────────

// A token for a user who no longer exists must read as unauthenticated, not 500.
func TestUserHandlers_DeletedUserIs401(t *testing.T) {
	authSvc := newTestAuthSvc()
	pair, err := authSvc.Issue(uuid.NewString(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get", func(t *testing.T) {
		uc := &mockGetUserByIDUsecase{
			executeFn: func(context.Context, string) (*user.User, error) { return nil, user.ErrNotFound },
		}
		w := get(t, newTestRouter(nil, authSvc, uc, nil, nil), "/users/me", pair.AccessToken)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401: %s", w.Code, w.Body)
		}
	})

	t.Run("update", func(t *testing.T) {
		uc := &mockUpdateUserByIDUsecase{
			executeFn: func(context.Context, string, *string, *string) (*user.User, error) {
				return nil, user.ErrNotFound
			},
		}
		w := patch(t, newTestRouter(nil, authSvc, nil, uc, nil), "/users/me", pair.AccessToken, `{"username":"x"}`)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401: %s", w.Code, w.Body)
		}
	})
}

func TestUserHandlers_UnexpectedErrorIs500(t *testing.T) {
	authSvc := newTestAuthSvc()
	sentinel := errors.New("db down")
	pair, err := authSvc.Issue(uuid.NewString(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	uc := &mockGetUserByIDUsecase{
		executeFn: func(context.Context, string) (*user.User, error) { return nil, sentinel },
	}
	if w := get(t, newTestRouter(nil, authSvc, uc, nil, nil), "/users/me", pair.AccessToken); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}

	uuc := &mockUpdateUserByIDUsecase{
		executeFn: func(context.Context, string, *string, *string) (*user.User, error) { return nil, sentinel },
	}
	w := patch(t, newTestRouter(nil, authSvc, nil, uuc, nil), "/users/me", pair.AccessToken, `{"username":"x"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}
}

// UpdateUserByID accepts an email-only patch and converts the typed email.
func TestUpdateUserByIDHandler_EmailOnly(t *testing.T) {
	authSvc := newTestAuthSvc()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	var gotEmail *string
	uc := &mockUpdateUserByIDUsecase{
		executeFn: func(_ context.Context, _ string, username, email *string) (*user.User, error) {
			gotEmail = email
			if username != nil {
				t.Fatalf("username should stay nil, got %q", *username)
			}
			return &user.User{
				ID: pgtype.UUID{Bytes: userID, Valid: true}, Username: "alice",
				Email: "new@example.com", Role: auth.RoleUser,
				FavoriteTrackIDs: []pgtype.UUID{}, FavoriteStreamIDs: []pgtype.UUID{},
				FavoritePlaylistIDs: []pgtype.UUID{}, CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	pair, err := authSvc.Issue(userID.String(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	w := patch(t, newTestRouter(nil, authSvc, nil, uc, nil), "/users/me", pair.AccessToken,
		`{"email":"new@example.com"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if gotEmail == nil || *gotEmail != "new@example.com" {
		t.Fatalf("email = %v", gotEmail)
	}
}

// ── HTTPError ─────────────────────────────────────────────────────────────────

func TestHTTPError_Error(t *testing.T) {
	err := &api.HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}

	if got := err.Error(); got != "503: request timeout" {
		t.Fatalf("Error() = %q", got)
	}
}

// ── ListFeatureFlags ────────────────────────────────────────────────────────

func TestListFeatureFlagsHandler_Success(t *testing.T) {
	authSvc := newTestAuthSvc()
	now := time.Now().UTC().Truncate(time.Second)
	flagID := uuid.New().String()

	featureSvc := &mockFeatureSvc{
		allFn: func(_ context.Context) ([]features.Flag, error) {
			return []features.Flag{
				{ID: flagID, Name: "chat_websocket", Enabled: false, Description: "live chat", UpdatedAt: now},
			}, nil
		},
	}

	pair, err := authSvc.Issue(uuid.New().String(), auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	w := get(t, newTestRouterWithFeatureSvc(authSvc, featureSvc), "/admin/features", pair.AccessToken)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var got api.FeatureFlagList
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Name != "chat_websocket" {
		t.Fatalf("unexpected flag list payload: %+v", got)
	}
}

func TestListFeatureFlagsHandler_NonAdminForbidden(t *testing.T) {
	authSvc := newTestAuthSvc()
	featureSvc := &mockFeatureSvc{}

	pair, err := authSvc.Issue(uuid.New().String(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	w := get(t, newTestRouterWithFeatureSvc(authSvc, featureSvc), "/admin/features", pair.AccessToken)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body)
	}
}

func TestListFeatureFlagsHandler_Unauthenticated(t *testing.T) {
	authSvc := newTestAuthSvc()
	featureSvc := &mockFeatureSvc{}

	w := get(t, newTestRouterWithFeatureSvc(authSvc, featureSvc), "/admin/features", "")

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body)
	}
}

// ── ToggleFeatureFlag ───────────────────────────────────────────────────────

func TestToggleFeatureFlagHandler_Success(t *testing.T) {
	authSvc := newTestAuthSvc()
	now := time.Now().UTC().Truncate(time.Second)
	flagID := uuid.New().String()

	featureSvc := &mockFeatureSvc{
		toggleFn: func(_ context.Context, name string, enabled bool) (*features.Flag, error) {
			if name != "chat_websocket" || !enabled {
				t.Fatalf("unexpected toggle args: name=%s enabled=%v", name, enabled)
			}
			return &features.Flag{ID: flagID, Name: name, Enabled: enabled, Description: "live chat", UpdatedAt: now}, nil
		},
	}

	pair, err := authSvc.Issue(uuid.New().String(), auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	w := patch(t, newTestRouterWithFeatureSvc(authSvc, featureSvc), "/admin/features/chat_websocket/toggle", pair.AccessToken,
		`{"enabled":true}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var got api.FeatureFlag
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "chat_websocket" || !got.Enabled {
		t.Fatalf("unexpected flag payload: %+v", got)
	}
}

func TestToggleFeatureFlagHandler_NonAdminForbidden(t *testing.T) {
	authSvc := newTestAuthSvc()
	featureSvc := &mockFeatureSvc{}

	pair, err := authSvc.Issue(uuid.New().String(), auth.RoleBroadcaster)
	if err != nil {
		t.Fatal(err)
	}

	w := patch(t, newTestRouterWithFeatureSvc(authSvc, featureSvc), "/admin/features/chat_websocket/toggle", pair.AccessToken,
		`{"enabled":true}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body)
	}
}

func TestToggleFeatureFlagHandler_NotFound(t *testing.T) {
	authSvc := newTestAuthSvc()
	featureSvc := &mockFeatureSvc{
		toggleFn: func(_ context.Context, _ string, _ bool) (*features.Flag, error) {
			return nil, features.ErrNotFound
		},
	}

	pair, err := authSvc.Issue(uuid.New().String(), auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	w := patch(t, newTestRouterWithFeatureSvc(authSvc, featureSvc), "/admin/features/does_not_exist/toggle", pair.AccessToken,
		`{"enabled":true}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body)
	}
}

// ── GetPublicFeatureFlags ───────────────────────────────────────────────────

// The client reads its flags before login, so an anonymous caller must get a
// plain name-to-state map — no id, description or timestamp.
func TestGetPublicFeatureFlagsHandler_AnonymousSucceeds(t *testing.T) {
	authSvc := newTestAuthSvc()
	now := time.Now().UTC().Truncate(time.Second)

	featureSvc := &mockFeatureSvc{
		allFn: func(_ context.Context) ([]features.Flag, error) {
			return []features.Flag{
				{ID: uuid.New().String(), Name: "live_streaming", Enabled: true, Description: "live", UpdatedAt: now},
				{ID: uuid.New().String(), Name: "chat_websocket", Enabled: false, Description: "chat", UpdatedAt: now},
			}, nil
		},
	}

	w := get(t, newTestRouterWithFeatureSvc(authSvc, featureSvc), "/features", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var got map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 || !got["live_streaming"] || got["chat_websocket"] {
		t.Fatalf("unexpected flag map: %+v", got)
	}
	if strings.Contains(w.Body.String(), "description") {
		t.Fatalf("public payload leaks flag metadata: %s", w.Body)
	}
}

func TestGetPublicFeatureFlagsHandler_TimeoutIs503(t *testing.T) {
	featureSvc := &mockFeatureSvc{
		allFn: func(context.Context) ([]features.Flag, error) {
			return nil, context.DeadlineExceeded
		},
	}

	w := get(t, newTestRouterWithFeatureSvc(newTestAuthSvc(), featureSvc), "/features", "")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
	}
}
