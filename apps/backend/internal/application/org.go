package application

import (
	"errors"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

var ErrLastOwner = errors.New("cannot remove or demote the last owner")

type OrgMemberView struct {
	Role string
}

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

func RoleAtLeast(role, min string) bool {
	return rolePrecedence(role) >= rolePrecedence(min)
}

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

func memberViews(members []model.Member, target uuid.UUID) ([]OrgMemberView, int) {
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
