package auth_test

import (
	"testing"

	"github.com/hyppoliteprn/lyo/internal/auth"
)

func TestRoleAtLeast(t *testing.T) {
	tests := []struct {
		role, min auth.Role
		want      bool
	}{
		{auth.RoleAnonymous, auth.RoleAnonymous, true},
		{auth.RoleAnonymous, auth.RoleUser, false},
		{auth.RoleUser, auth.RoleAnonymous, true},
		{auth.RoleUser, auth.RoleBroadcaster, false},
		{auth.RoleBroadcaster, auth.RoleUser, true},
		{auth.RoleBroadcaster, auth.RoleBroadcaster, true},
		{auth.RoleBroadcaster, auth.RoleAdmin, false},
		{auth.RoleAdmin, auth.RoleBroadcaster, true},
		{auth.RoleAdmin, auth.RoleAdmin, true},
		// An unknown role weighs 0, i.e. no more than anonymous.
		{auth.Role("moderator"), auth.RoleUser, false},
		{auth.Role("moderator"), auth.RoleAnonymous, true},
	}

	for _, tt := range tests {
		if got := tt.role.AtLeast(tt.min); got != tt.want {
			t.Errorf("Role(%q).AtLeast(%q) = %v, want %v", tt.role, tt.min, got, tt.want)
		}
	}
}
