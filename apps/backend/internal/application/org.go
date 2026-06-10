package application

import (
	"errors"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// ErrLastOwner is returned when an operation would remove the last owner of an
// organisation (demote or remove).
var ErrLastOwner = errors.New("cannot remove or demote the last owner")

// OrgMemberView is a lightweight projection of an organisation member used by
// guard functions to avoid importing the postgres model.
type OrgMemberView struct {
	Role string
}

// rolePrecedence maps role names to numeric precedence. Unknown roles rank 0.
func rolePrecedence(role string) int {
	switch role {
	case "owner":
		return 3
	case "admin":
		return 2
	case "member":
		return 1
	}
	return 0
}

// RoleAtLeast returns true when role is at least as privileged as min.
func RoleAtLeast(role, min string) bool {
	return rolePrecedence(role) >= rolePrecedence(min)
}

// canDemoteOrRemove checks whether the member at idx may be removed or demoted
// without leaving the organisation without an owner. It returns ErrLastOwner
// when the target is the only owner.
func canDemoteOrRemove(members []OrgMemberView, idx int) error {
	if members[idx].Role != "owner" {
		return nil
	}
	for i, m := range members {
		if i != idx && m.Role == "owner" {
			return nil
		}
	}
	return ErrLastOwner
}

// memberViews projects members to OrgMemberView and returns the index of the
// member whose UserID == target (-1 if absent).
func memberViews(members []postgres.Member, target uuid.UUID) ([]OrgMemberView, int) {
	views := make([]OrgMemberView, len(members))
	idx := -1
	for i, m := range members {
		views[i] = OrgMemberView{Role: m.Role}
		if m.UserID != nil && *m.UserID == target {
			idx = i
		}
	}
	return views, idx
}

// slugify converts a human-readable name into a URL-safe slug:
//   - Unicode NFD decomposition + strip combining marks (so "Café"→"cafe")
//   - Lowercase
//   - Keep only [a-z0-9]; collapse any run of other characters to a single "-"
//   - Trim leading/trailing "-"
//
// Non-foldable scripts (Cyrillic, CJK, etc.) produce no latin output and the
// result may be an empty string. The caller is responsible for providing a
// fallback.
func slugify(name string) string {

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	folded, _, err := transform.String(t, name)
	if err != nil {
		folded = name
	}

	lower := strings.ToLower(folded)

	var sb strings.Builder
	inSep := true
	for _, r := range lower {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			sb.WriteRune(r)
			inSep = false
		} else {
			if !inSep {
				sb.WriteRune('-')
				inSep = true
			}
		}
	}

	result := sb.String()
	return strings.TrimRight(result, "-")
}
