package postgres

import "github.com/luckyrogue/lead-cat/internal/application/model"

type (
	PlatformUser       = model.PlatformUser
	User               = model.User
	Organization       = model.Organization
	Member             = model.Member
	PendingChat        = model.PendingChat
	Employee           = model.Employee
	MeetingParticipant = model.MeetingParticipant
	Meeting            = model.Meeting
	MeetingWithTZ      = model.MeetingWithTZ
	BotUser            = model.BotUser
	AuditEntry         = model.AuditEntry
	AuditFilter        = model.AuditFilter
	OrganizationInvite = model.OrganizationInvite
	MagicLinkToken     = model.MagicLinkToken
	WebSession         = model.WebSession
	CalendarConnection = model.CalendarConnection
	CalendarOAuthState = model.CalendarOAuthState
	InviteView         = model.InviteView
)
