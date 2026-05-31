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

type Employee struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	Dept        string    `json:"dept"`
	HasTelegram bool      `json:"has_telegram"`
}

type MeetingParticipant struct {
	EmployeeID *uuid.UUID `json:"employee_id,omitempty"`
	Email      string     `json:"email"`
}

type Meeting struct {
	ID              uuid.UUID            `json:"id"`
	WorkspaceID     uuid.UUID            `json:"workspace_id"`
	OrganizerUserID *uuid.UUID           `json:"organizer_user_id,omitempty"`
	Dept            string               `json:"dept"`
	Type            string               `json:"type"`
	Host            string               `json:"host"`
	StartsAt        time.Time            `json:"starts_at"`
	EndsAt          time.Time            `json:"ends_at"`
	Recurrence      string               `json:"recurrence"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	GoogleEventID   string               `json:"google_event_id"`
	MeetLink        string               `json:"meet_link"`
	Status          string               `json:"status"`
	SeriesID        *uuid.UUID           `json:"series_id,omitempty"`
	RecurrenceUntil *time.Time           `json:"recurrence_until,omitempty"`
	Participants    []MeetingParticipant `json:"participants"`
}

type BotUser struct {
	ID              uuid.UUID `json:"id"`
	TelegramID      int64     `json:"telegram_id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	ReminderMinutes string    `json:"reminder_minutes"`
}
