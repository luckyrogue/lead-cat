package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PlatformUser struct {
	ID         uuid.UUID
	AuthSub    string
	Email      string
	TelegramID *int64
	AvatarURL  string
	AuthMethod string
	CreatedAt  time.Time
	Timezone   string
	Language   string
}

type User struct {
	ID         uuid.UUID
	AuthSub    string
	Email      string
	TelegramID *int64
}

type Organization struct {
	ID           uuid.UUID  `json:"id"`
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	NotifyChatID *int64     `json:"notify_chat_id,omitempty"`
	MeetLink     string     `json:"meet_link,omitempty"`
	TZ           string     `json:"tz,omitempty"`
	OwnerUserID  *uuid.UUID `json:"owner_user_id,omitempty"`
}

type Member struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	UserID           *uuid.UUID
	TelegramUsername string
	Role             string
	InvitedEmail     *string
	Email            string
}

type PendingChat struct {
	ID             uuid.UUID
	OrganizationID *uuid.UUID
	TelegramUserID int64
	ChatID         int64
	ChatTitle      string
}

type Employee struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	FullName       string    `json:"full_name"`
	Email          string    `json:"email"`
	Dept           string    `json:"dept"`
	HasTelegram    bool      `json:"has_telegram"`
}

type MeetingParticipant struct {
	EmployeeID *uuid.UUID `json:"employee_id,omitempty"`
	Email      string     `json:"email"`
}

type Meeting struct {
	ID              uuid.UUID            `json:"id"`
	OrganizationID  uuid.UUID            `json:"organization_id"`
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
	RecurrenceDays  []int                `json:"recurrence_days,omitempty"`
	Participants    []MeetingParticipant `json:"participants"`
}

type MeetingWithTZ struct {
	Meeting
	TZ string
}

// MeetingFilter narrows a meetings query. Zero value = no filtering.
type MeetingFilter struct {
	Status    string     // "scheduled" | "cancelled"; "" or "all" = any
	From      *time.Time // inclusive lower bound on starts_at
	To        *time.Time // exclusive upper bound on starts_at
	Dept      string     // case-insensitive contains match when non-empty
	Organizer *uuid.UUID // organizer_user_id when non-nil
}

type BotUser struct {
	ID              uuid.UUID `json:"id"`
	TelegramID      int64     `json:"telegram_id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	ReminderMinutes string    `json:"reminder_minutes"`
	Timezone        string    `json:"timezone"`
	Language        string    `json:"language"`
}

type AuditEntry struct {
	ID              uuid.UUID       `json:"id"`
	ActorUserID     uuid.UUID       `json:"actor_user_id"`
	ActorTelegramID int64           `json:"actor_telegram_id"`
	ActorEmail      string          `json:"actor_email"`
	ActorKind       string          `json:"actor_kind"`
	Action          string          `json:"action"`
	TargetKind      string          `json:"target_kind"`
	TargetID        string          `json:"target_id"`
	Details         json.RawMessage `json:"details"`
	CreatedAt       time.Time       `json:"created_at"`
}

type AuditFilter struct {
	Action     string
	ActorEmail string
	Limit      int
}

type OrganizationInvite struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Email           string
	Role            string
	TokenHash       []byte
	ExpiresAt       time.Time
	AcceptedAt      *time.Time
	CreatedByUserID *uuid.UUID
	CreatedAt       time.Time
}

type MagicLinkToken struct {
	ID         uuid.UUID
	Email      string
	TokenHash  []byte
	Purpose    string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

type WebSession struct {
	ID         uuid.UUID
	TokenHash  []byte
	UserID     uuid.UUID
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	UserAgent  string
	IP         string
}
