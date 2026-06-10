package auth

import "strings"

// SubEmail builds the platform_users.auth_sub key for an email identity.
func SubEmail(email string) string {
	return "email:" + strings.ToLower(strings.TrimSpace(email))
}
