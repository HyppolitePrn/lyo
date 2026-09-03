package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/user"
)

// The tests below pin one property: no HTTP request can raise the role of an
// account unless the caller already holds that role. They are regression
// tests, not descriptions of a feature — each one fails the day someone wires
// a caller-supplied role into registration, profile updates or token refresh.

func roleOf(t *testing.T, authSvc *auth.Service, token string) auth.Role {
	t.Helper()
	claims, err := authSvc.Verify(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	return claims.Role
}

func testUser(t *testing.T, role auth.Role) *user.User {
	t.Helper()
	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}
	return &user.User{
		ID:       pgtype.UUID{Bytes: id, Valid: true},
		Username: "alice",
		Email:    "alice@example.com",
		Role:     role,
	}
}

// A "role" field in the registration body must not reach the account: the
// service takes no role argument at all, so the token comes back as a plain user.
func TestRegisterHandler_IgnoresClientSuppliedRole(t *testing.T) {
	authSvc := newTestAuthSvc()
	svc := &mockUserService{
		registerFn: func(_ context.Context, _, _, _ string) (auth.TokenPair, error) {
			// Mirrors user.Service.Register, which always passes RoleUser.
			return authSvc.Issue("user-1", auth.RoleUser)
		},
	}
	w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/register",
		`{"username":"mallory","email":"m@example.com","password":"secret123","role":"admin"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body)
	}
	access, _ := decodeTokens(t, w)
	if got := roleOf(t, authSvc, access); got != auth.RoleUser {
		t.Fatalf("registration granted role %q, want %q", got, auth.RoleUser)
	}
}

// PATCH /users/me carries username and email only. A role in the body must
// leave the stored role untouched.
func TestUpdateUserByIDHandler_IgnoresClientSuppliedRole(t *testing.T) {
	authSvc := newTestAuthSvc()
	pair, err := authSvc.Issue(uuid.New().String(), auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	stored := testUser(t, auth.RoleUser)
	uc := &mockUpdateUserByIDUsecase{
		executeFn: func(_ context.Context, _ string, username, _ *string) (*user.User, error) {
			if username != nil {
				stored.Username = *username
			}
			return stored, nil
		},
	}

	w := patch(t, newTestRouter(nil, authSvc, nil, uc, nil), "/users/me", pair.AccessToken,
		`{"username":"mallory","role":"admin"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	if stored.Role != auth.RoleUser {
		t.Fatalf("profile update changed role to %q", stored.Role)
	}
}

// A refresh token minted while the account was an admin must not keep minting
// admin tokens after the demotion: the new pair carries the role the database
// holds now, not the one baked into the token being exchanged.
func TestRefreshTokenHandler_UsesStoredRoleNotTokenRole(t *testing.T) {
	authSvc := newTestAuthSvc()
	pair, err := authSvc.Issue("user-1", auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	svc := &mockUserService{
		getByIDFn: func(context.Context, string) (*user.User, error) {
			return testUser(t, auth.RoleUser), nil // demoted since the token was issued
		},
	}
	w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/refresh",
		`{"refresh_token":"`+pair.RefreshToken+`"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	access, refresh := decodeTokens(t, w)
	if got := roleOf(t, authSvc, access); got != auth.RoleUser {
		t.Fatalf("access token kept role %q after demotion, want %q", got, auth.RoleUser)
	}
	if got := roleOf(t, authSvc, refresh); got != auth.RoleUser {
		t.Fatalf("refresh token kept role %q after demotion, want %q", got, auth.RoleUser)
	}
}

// A refresh token outliving its account is not a valid credential.
func TestRefreshTokenHandler_DeletedUser(t *testing.T) {
	authSvc := newTestAuthSvc()
	pair, err := authSvc.Issue("user-1", auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	svc := &mockUserService{
		getByIDFn: func(context.Context, string) (*user.User, error) {
			return nil, user.ErrNotFound
		},
	}
	w := post(t, newTestRouter(svc, authSvc, nil, nil, nil), "/auth/refresh",
		`{"refresh_token":"`+pair.RefreshToken+`"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body)
	}
}

// The admin surface answers 403 to a self-minted token from another issuer:
// forging a role means forging the signature, which needs JWT_SECRET.
func TestAdminEndpoint_RejectsTokenSignedWithAnotherSecret(t *testing.T) {
	authSvc := newTestAuthSvc()
	attackerSvc := auth.NewService("attacker-secret-at-least-32-chars!!", time.Minute, time.Hour)
	forged, err := attackerSvc.Issue("user-1", auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := authSvc.Verify(forged.AccessToken); err == nil {
		t.Fatal("a token signed with another secret verified against the server key")
	}

	w := get(t, newTestRouter(nil, authSvc, nil, nil, nil), "/admin/supervision", forged.AccessToken)
	if w.Code == http.StatusOK {
		t.Fatalf("forged admin token was accepted: %d %s", w.Code, w.Body)
	}
}
