package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

var errUnimplemented = errors.New("unimplemented")

type unimplementedRepo struct{}

func (unimplementedRepo) Ping(context.Context) error { return errUnimplemented }
func (unimplementedRepo) GetUserByID(context.Context, uuid.UUID) (model.User, error) {
	return model.User{}, errUnimplemented
}
func (unimplementedRepo) GetUserTelegramID(context.Context, uuid.UUID) (int64, bool, error) {
	return 0, false, errUnimplemented
}
func (unimplementedRepo) UpsertUserIdentity(context.Context, string, string) (model.User, error) {
	return model.User{}, errUnimplemented
}
func (unimplementedRepo) UpsertWebIdentity(context.Context, string, string, string, string) (model.PlatformUser, error) {
	return model.PlatformUser{}, errUnimplemented
}
func (unimplementedRepo) GetPlatformUserByID(context.Context, uuid.UUID) (model.PlatformUser, bool, error) {
	return model.PlatformUser{}, false, errUnimplemented
}
func (unimplementedRepo) GetPlatformUserIDByTelegramID(context.Context, int64) (uuid.UUID, bool, error) {
	return uuid.Nil, false, errUnimplemented
}
func (unimplementedRepo) LinkTelegram(context.Context, uuid.UUID, int64) error { return errUnimplemented }
func (unimplementedRepo) LinkMemberUserIDsByTelegram(context.Context, uuid.UUID, string) error {
	return errUnimplemented
}
func (unimplementedRepo) GetBotUserByTelegramID(context.Context, int64) (model.BotUser, error) {
	return model.BotUser{}, errUnimplemented
}
func (unimplementedRepo) SetReminderMinutes(context.Context, int64, string) error { return errUnimplemented }
func (unimplementedRepo) GetOrganization(context.Context, uuid.UUID) (model.Organization, error) {
	return model.Organization{}, errUnimplemented
}
func (unimplementedRepo) CreateOrganization(context.Context, string, string, uuid.UUID) (model.Organization, error) {
	return model.Organization{}, errUnimplemented
}
func (unimplementedRepo) EnsureDefaultOrganizationID(context.Context, string, string, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, errUnimplemented
}
func (unimplementedRepo) UpdateOrganization(context.Context, uuid.UUID, string, string) error {
	return errUnimplemented
}
func (unimplementedRepo) ListOrganizationsForUser(context.Context, uuid.UUID) ([]model.Organization, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListOrganizationsWithGoogle(context.Context) ([]uuid.UUID, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) LinkChat(context.Context, uuid.UUID, int64) error { return errUnimplemented }
func (unimplementedRepo) GetGoogleConfig(context.Context, uuid.UUID) ([]byte, string, string, error) {
	return nil, "", "", errUnimplemented
}
func (unimplementedRepo) SetGoogleConfig(context.Context, uuid.UUID, []byte, string, string) error {
	return errUnimplemented
}
func (unimplementedRepo) ListMembers(context.Context, uuid.UUID) ([]model.Member, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListOrgMembers(context.Context, uuid.UUID) ([]model.Member, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) AddMember(context.Context, uuid.UUID, string, string) (model.Member, error) {
	return model.Member{}, errUnimplemented
}
func (unimplementedRepo) DeleteMember(context.Context, uuid.UUID) error { return errUnimplemented }
func (unimplementedRepo) RemoveMember(context.Context, uuid.UUID, uuid.UUID) error {
	return errUnimplemented
}
func (unimplementedRepo) UpdateMemberRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	return errUnimplemented
}
func (unimplementedRepo) CreateInvite(context.Context, uuid.UUID, string, string, []byte, time.Time, uuid.UUID) (model.OrganizationInvite, error) {
	return model.OrganizationInvite{}, errUnimplemented
}
func (unimplementedRepo) ListInvites(context.Context, uuid.UUID) ([]model.OrganizationInvite, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) DeleteInvite(context.Context, uuid.UUID, uuid.UUID) error { return errUnimplemented }
func (unimplementedRepo) AcceptInvitesForEmail(context.Context, string, uuid.UUID) (int, error) {
	return 0, errUnimplemented
}
func (unimplementedRepo) ListEmployees(context.Context, uuid.UUID) ([]model.Employee, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) SearchEmployeesGlobal(context.Context, string) ([]model.Employee, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) GetMeeting(context.Context, uuid.UUID, uuid.UUID) (model.Meeting, error) {
	return model.Meeting{}, errUnimplemented
}
func (unimplementedRepo) CreateMeeting(context.Context, model.Meeting) (model.Meeting, error) {
	return model.Meeting{}, errUnimplemented
}
func (unimplementedRepo) UpdateMeeting(context.Context, uuid.UUID, uuid.UUID, model.Meeting) error {
	return errUnimplemented
}
func (unimplementedRepo) UpdateMeetingsTx(context.Context, uuid.UUID, []model.Meeting) error {
	return errUnimplemented
}
func (unimplementedRepo) CancelMeeting(context.Context, uuid.UUID, uuid.UUID) error { return errUnimplemented }
func (unimplementedRepo) ListMeetings(context.Context, uuid.UUID) ([]model.Meeting, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListMeetingsByOrganizer(context.Context, uuid.UUID, uuid.UUID) ([]model.Meeting, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListMeetingsFiltered(context.Context, uuid.UUID, model.MeetingFilter) ([]model.Meeting, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListMeetingsByOrganizerTelegram(context.Context, int64) ([]model.MeetingWithTZ, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListMeetingsOverlapping(context.Context, []string, time.Time, time.Time) ([]model.Meeting, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListScheduleForEmail(context.Context, string, time.Time, time.Time) ([]model.Meeting, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) CreateMeetingSeries(context.Context, []model.Meeting, []model.MeetingParticipant) ([]model.Meeting, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListSeriesOccurrences(context.Context, uuid.UUID, uuid.UUID, time.Time) ([]model.Meeting, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) ListSeriesAllOccurrences(context.Context, uuid.UUID, uuid.UUID) ([]model.Meeting, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) CancelSeriesOccurrences(context.Context, uuid.UUID, uuid.UUID, time.Time) (int, error) {
	return 0, errUnimplemented
}
func (unimplementedRepo) CancelAllSeriesOccurrences(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	return 0, errUnimplemented
}
func (unimplementedRepo) AddParticipants(context.Context, uuid.UUID, []model.MeetingParticipant) error {
	return errUnimplemented
}
func (unimplementedRepo) RemoveParticipant(context.Context, uuid.UUID, string) error { return errUnimplemented }
func (unimplementedRepo) ListParticipants(context.Context, uuid.UUID) ([]model.MeetingParticipant, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) InsertAuditEntry(context.Context, model.AuditEntry) error { return errUnimplemented }
func (unimplementedRepo) ListAuditEntries(context.Context, model.AuditFilter) ([]model.AuditEntry, error) {
	return nil, errUnimplemented
}
func (unimplementedRepo) InsertMagicLink(context.Context, string, []byte, time.Time) error {
	return errUnimplemented
}
func (unimplementedRepo) ConsumeMagicLink(context.Context, []byte, time.Time) (string, bool, error) {
	return "", false, errUnimplemented
}
func (unimplementedRepo) CreateWebSession(context.Context, []byte, uuid.UUID, time.Time, string, string) (model.WebSession, error) {
	return model.WebSession{}, errUnimplemented
}
func (unimplementedRepo) ResolveWebSession(context.Context, []byte, time.Time) (model.WebSession, bool, error) {
	return model.WebSession{}, false, errUnimplemented
}
func (unimplementedRepo) TouchWebSession(context.Context, uuid.UUID, time.Time, time.Time) error {
	return errUnimplemented
}
func (unimplementedRepo) RevokeWebSession(context.Context, []byte, time.Time) error { return errUnimplemented }
