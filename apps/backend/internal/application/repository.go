package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type Repository interface {
	Ping(ctx context.Context) error

	GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error)
	GetUserTelegramID(ctx context.Context, userID uuid.UUID) (int64, bool, error)
	UpsertUserIdentity(ctx context.Context, authSub, email string) (model.User, error)
	UpsertWebIdentity(ctx context.Context, email, name, avatarURL, authMethod string) (model.PlatformUser, error)
	GetPlatformUserByID(ctx context.Context, id uuid.UUID) (model.PlatformUser, bool, error)
	GetPlatformUserLanguageByEmail(ctx context.Context, email string) (string, bool, error)
	GetPlatformUserIDByTelegramID(ctx context.Context, telegramID int64) (uuid.UUID, bool, error)
	LinkTelegram(ctx context.Context, userID uuid.UUID, telegramID int64) error
	LinkMemberUserIDsByTelegram(ctx context.Context, userID uuid.UUID, telegramUsername string) error

	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (model.BotUser, error)
	SetReminderMinutes(ctx context.Context, telegramID int64, csv string) error
	SetBotUserPrefs(ctx context.Context, telegramID int64, timezone, language string) error
	SetPlatformUserPrefs(ctx context.Context, userID uuid.UUID, timezone, language string) error

	GetOrganization(ctx context.Context, id uuid.UUID) (model.Organization, error)
	CreateOrganization(ctx context.Context, name, slug string, ownerUserID uuid.UUID) (model.Organization, error)
	EnsureDefaultOrganizationID(ctx context.Context, defaultTZ, defaultMeetLink string, ownerUserID uuid.UUID) (uuid.UUID, error)
	UpdateOrganization(ctx context.Context, id uuid.UUID, meetLink, tz string) error
	ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]model.Organization, error)
	ListOrganizationsWithGoogle(ctx context.Context) ([]uuid.UUID, error)
	LinkChat(ctx context.Context, organizationID uuid.UUID, chatID int64) error
	GetGoogleConfig(ctx context.Context, id uuid.UUID) (encJSON []byte, subject, calendarID string, err error)
	SetGoogleConfig(ctx context.Context, id uuid.UUID, encJSON []byte, subject, calendarID string) error

	ListMembers(ctx context.Context, organizationID uuid.UUID) ([]model.Member, error)
	ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]model.Member, error)
	AddMember(ctx context.Context, organizationID uuid.UUID, username, role string) (model.Member, error)
	DeleteMember(ctx context.Context, memberID uuid.UUID) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error

	CreateInvite(ctx context.Context, orgID uuid.UUID, email, role string, tokenHash []byte, expiresAt time.Time, createdBy uuid.UUID) (model.OrganizationInvite, error)
	ListInvites(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationInvite, error)
	DeleteInvite(ctx context.Context, orgID, inviteID uuid.UUID) error
	AcceptInvitesForEmail(ctx context.Context, email string, userID uuid.UUID) (int, error)
	ListPendingInvitesForEmail(ctx context.Context, email string) ([]model.InviteView, error)
	AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error
	DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error

	ListEmployees(ctx context.Context, organizationID uuid.UUID) ([]model.Employee, error)
	SearchEmployeesGlobal(ctx context.Context, query string) ([]model.Employee, error)

	GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (model.Meeting, error)
	CreateMeeting(ctx context.Context, m model.Meeting) (model.Meeting, error)
	UpdateMeeting(ctx context.Context, organizationID, id uuid.UUID, m model.Meeting) error
	UpdateMeetingsTx(ctx context.Context, organizationID uuid.UUID, ms []model.Meeting) error
	CancelMeeting(ctx context.Context, organizationID, id uuid.UUID) error
	ListMeetings(ctx context.Context, organizationID uuid.UUID) ([]model.Meeting, error)
	ListMeetingsByOrganizer(ctx context.Context, organizationID, organizerID uuid.UUID) ([]model.Meeting, error)
	ListMeetingsFiltered(ctx context.Context, organizationID uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error)
	ListMeetingsByOrganizerTelegram(ctx context.Context, telegramID int64) ([]model.MeetingWithTZ, error)
	ListMeetingsOverlapping(ctx context.Context, emails []string, from, to time.Time) ([]model.Meeting, error)
	ListScheduleForEmail(ctx context.Context, email string, from, to time.Time) ([]model.Meeting, error)

	CreateMeetingSeries(ctx context.Context, ms []model.Meeting, ps []model.MeetingParticipant) ([]model.Meeting, error)
	ListSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID, fromStart time.Time) ([]model.Meeting, error)
	ListSeriesAllOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID) ([]model.Meeting, error)
	ListSeriesOccurrenceStarts(ctx context.Context, organizationID, seriesID uuid.UUID) ([]time.Time, error)
	CancelSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID, fromStart time.Time) (int, error)
	CancelAllSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID) (int, error)
	SetSeriesRecurrenceUntil(ctx context.Context, organizationID, seriesID uuid.UUID, until time.Time) error

	AddParticipants(ctx context.Context, meetingID uuid.UUID, ps []model.MeetingParticipant) error
	RemoveParticipant(ctx context.Context, meetingID uuid.UUID, email string) error
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error)

	InsertAuditEntry(ctx context.Context, e model.AuditEntry) error
	ListAuditEntries(ctx context.Context, f model.AuditFilter) ([]model.AuditEntry, error)

	InsertMagicLink(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) error
	ConsumeMagicLink(ctx context.Context, tokenHash []byte, now time.Time) (string, bool, error)

	CreateWebSession(ctx context.Context, tokenHash []byte, userID uuid.UUID, expiresAt time.Time, ua, ip string) (model.WebSession, error)
	ResolveWebSession(ctx context.Context, tokenHash []byte, now time.Time) (model.WebSession, bool, error)
	TouchWebSession(ctx context.Context, id uuid.UUID, lastSeen, expiresAt time.Time) error
	RevokeWebSession(ctx context.Context, tokenHash []byte, now time.Time) error

	CreateCalendarOAuthState(ctx context.Context, st model.CalendarOAuthState) error
	ConsumeCalendarOAuthState(ctx context.Context, state string) (model.CalendarOAuthState, error)
	UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error
	ListCalendarConnections(ctx context.Context, email string) ([]model.CalendarConnection, error)
	DeleteCalendarConnection(ctx context.Context, email, provider string) error

	GetOrganizationBySlug(ctx context.Context, slug string) (model.Organization, error)
	GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error)
	CreateJoinRequest(ctx context.Context, orgID, userID uuid.UUID) error
	ListJoinRequestsForUser(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error)
	ListPendingJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error)
	AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error
	DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error

	CreateBookingEventType(ctx context.Context, et model.BookingEventType) (model.BookingEventType, error)
	GetBookingEventType(ctx context.Context, id uuid.UUID) (model.BookingEventType, error)
	GetBookingEventTypeBySlug(ctx context.Context, slug string) (model.BookingEventType, error)
	ListBookingEventTypesForUser(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error)
	UpdateBookingEventType(ctx context.Context, et model.BookingEventType) error
	DeleteBookingEventType(ctx context.Context, id uuid.UUID) error

	WithHostBookingLock(ctx context.Context, hostUserID uuid.UUID, start time.Time, fn func(ctx context.Context) error) error
}
