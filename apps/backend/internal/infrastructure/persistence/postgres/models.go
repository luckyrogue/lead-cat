package postgres

import "github.com/luckyrogue/lead-cat/internal/application/model"

// Shared data structs live in the application/model leaf package so that the
// application and delivery layers can depend on them without importing this
// persistence package (clean-arch: deps point inward). These aliases keep the
// repository code in this package terse.
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
)
