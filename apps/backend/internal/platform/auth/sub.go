package auth

import "strings"

func SubEmail(email string) string {
	return "email:" + strings.ToLower(strings.TrimSpace(email))
}
