package passwordreset

import "time"

// Token is a single-use password reset token.
type Token struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
}
