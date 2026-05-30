package postgres

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID
	AuthSub    string
	Email      string
	Phone      string
	TelegramID *int64
}

type Workspace struct {
	ID           uuid.UUID  `json:"id"`
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	NotifyChatID *int64     `json:"notify_chat_id,omitempty"`
	MeetLink     string     `json:"meet_link,omitempty"`
	TZ           string     `json:"tz,omitempty"`
	OwnerUserID  *uuid.UUID `json:"owner_user_id,omitempty"`
	VCSProvider  string     `json:"vcs_provider,omitempty"`
	VCSNamespace string     `json:"vcs_namespace,omitempty"`
	VCSBaseURL   *string    `json:"vcs_base_url,omitempty"`
	HasVCSToken  bool       `json:"has_vcs_token,omitempty"`
}

type Member struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	UserID           *uuid.UUID
	TelegramUsername string
	Role             string
}

type Scenario struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	Name        string          `json:"name"`
	Enabled     bool            `json:"enabled"`
	Definition  json.RawMessage `json:"definition"`
}

type ScenarioRun struct {
	ID          uuid.UUID  `json:"id"`
	ScenarioID  uuid.UUID  `json:"scenario_id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Status      string     `json:"status"`
	Trigger     string     `json:"trigger"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

type PendingChat struct {
	ID             uuid.UUID
	WorkspaceID    *uuid.UUID
	TelegramUserID int64
	ChatID         int64
	ChatTitle      string
}
