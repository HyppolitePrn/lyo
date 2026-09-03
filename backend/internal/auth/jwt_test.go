package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/hyppoliteprn/lyo/internal/auth"
)

func newSvc() *auth.Service {
	return auth.NewService("test-jwt-secret-at-least-32-chars!", time.Minute, time.Hour)
}

func TestIssueThenVerify_RoundTrips(t *testing.T) {
	svc := newSvc()

	pair, err := svc.Issue("user-1", auth.RoleBroadcaster)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected both tokens to be non-empty")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Fatal("access and refresh tokens must differ (different TTLs)")
	}

	for name, token := range map[string]string{"access": pair.AccessToken, "refresh": pair.RefreshToken} {
		claims, err := svc.Verify(token)
		if err != nil {
			t.Fatalf("verify %s: %v", name, err)
		}
		if claims.UserID != "user-1" {
			t.Errorf("%s: user id = %q, want %q", name, claims.UserID, "user-1")
		}
		if claims.Role != auth.RoleBroadcaster {
			t.Errorf("%s: role = %q, want %q", name, claims.Role, auth.RoleBroadcaster)
		}
	}
}

func TestVerify_Malformed(t *testing.T) {
	if _, err := newSvc().Verify("not.a.jwt"); err == nil {
		t.Fatal("expected an error for a malformed token")
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	pair, err := newSvc().Issue("user-1", auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	other := auth.NewService("a-completely-different-secret-32b!", time.Minute, time.Hour)
	if _, err := other.Verify(pair.AccessToken); err == nil {
		t.Fatal("expected an error when the signing secret differs")
	}
}

func TestVerify_Expired(t *testing.T) {
	svc := auth.NewService("test-jwt-secret-at-least-32-chars!", -time.Minute, -time.Minute)
	pair, err := svc.Issue("user-1", auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Verify(pair.AccessToken); err == nil {
		t.Fatal("expected an error for an expired token")
	}
}

// A token signed with "alg": "none" must be rejected: accepting it would let
// anyone forge claims without knowing the secret.
func TestVerify_RejectsNonHMACSigningMethod(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodNone, &auth.Claims{
		UserID: "attacker",
		Role:   auth.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newSvc().Verify(signed); err == nil {
		t.Fatal("expected alg=none token to be rejected")
	}
}
