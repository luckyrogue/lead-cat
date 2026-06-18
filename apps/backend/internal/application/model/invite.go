package model

import "github.com/google/uuid"

type InviteView struct {
	InviteID       uuid.UUID `json:"invite_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	OrgName        string    `json:"org_name"`
	Role           string    `json:"role"`
}
