package application

import (
	"errors"

	"github.com/google/uuid"

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
