package model

import (
	"time"

	"github.com/google/uuid"
)

type JoinRequestView struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	OrgName        string    `json:"org_name"`
	Status         string    `json:"status"`
}

type JoinRequestAdminView struct {
	RequestID uuid.UUID `json:"request_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
