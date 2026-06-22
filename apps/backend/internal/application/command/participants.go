package command

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func normalizeEmail(s string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(s))
	if err != nil {
		return "", err
	}
	return strings.ToLower(addr.Address), nil
}

func (c *Meetings) AddParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error {
	m, _, err := c.loadForParticipantOp(ctx, organizationID, meetingID, userID)
	if err != nil {
		return err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return fmt.Errorf("%w: bad email: %v", ErrInvalidInput, err)
	}
	parts, err := c.Store.ListParticipants(ctx, meetingID)
	if err != nil {
		return err
	}
	for _, p := range parts {
		if p.Email == email {
			return fmt.Errorf("%w: already a participant", ErrInvalidInput)
		}
	}
	if err := c.Store.AddParticipants(ctx, meetingID, []model.MeetingParticipant{{Email: email}}); err != nil {
		return err
	}
	if err := c.syncAttendees(ctx, organizationID, m.GoogleEventID, meetingID, m.OrganizerUserID); err != nil {
		return err
	}
	if c.Queue != nil {
		if err := c.Queue.EnqueueParticipantAdded(ctx, organizationID, meetingID, email); err != nil && c.Log != nil {
			c.Log.Warn("enqueue participant added", zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return nil
}

func (c *Meetings) RemoveParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error {
	m, _, err := c.loadForParticipantOp(ctx, organizationID, meetingID, userID)
	if err != nil {
		return err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return fmt.Errorf("%w: bad email: %v", ErrInvalidInput, err)
	}
	if err := c.Store.RemoveParticipant(ctx, meetingID, email); err != nil {
		return err
	}
	if err := c.syncAttendees(ctx, organizationID, m.GoogleEventID, meetingID, m.OrganizerUserID); err != nil {
		return err
	}
	if c.Queue != nil {
		if err := c.Queue.EnqueueParticipantRemoved(ctx, organizationID, meetingID, email); err != nil && c.Log != nil {
			c.Log.Warn("enqueue participant removed", zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return nil
}

func (c *Meetings) loadForParticipantOp(ctx context.Context, organizationID, meetingID, userID uuid.UUID) (model.Meeting, model.Organization, error) {
	m, err := c.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return model.Meeting{}, model.Organization{}, err
	}
	org, err := c.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return model.Meeting{}, model.Organization{}, err
	}
	if !ownerOrOrganizer(org, m.OrganizerUserID, userID) {
		return model.Meeting{}, model.Organization{}, ErrForbidden
	}
	if m.Status != "scheduled" {
		return model.Meeting{}, model.Organization{}, model.ErrMeetingNotEditable
	}
	return m, org, nil
}

func (c *Meetings) syncAttendees(ctx context.Context, organizationID uuid.UUID, googleEventID string, meetingID uuid.UUID, organizerUserID *uuid.UUID) error {
	if googleEventID == "" {
		return nil
	}
	parts, err := c.Store.ListParticipants(ctx, meetingID)
	if err != nil {
		return err
	}
	var emails []string
	for _, p := range parts {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	calSvc, err := c.Calendar.For(ctx, organizationID, c.organizerEmail(ctx, organizerUserID))
	if err != nil {
		return err
	}
	if err := calSvc.UpdateAttendees(ctx, googleEventID, emails); err != nil {
		return fmt.Errorf("calendar: %w", err)
	}
	return nil
}
