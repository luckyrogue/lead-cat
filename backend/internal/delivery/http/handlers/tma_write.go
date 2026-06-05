package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// tmaCreateRequest is the create-meeting payload. No recurrence_until field: the
// Mini App only supports once-only meetings in this slice (recurring is deferred).
type tmaCreateRequest struct {
	Dept         string   `json:"dept"`
	Type         string   `json:"type"`
	Host         string   `json:"host"`
	Date         string   `json:"date"`  // YYYY-MM-DD
	Start        string   `json:"start"` // HH:MM
	End          string   `json:"end"`   // HH:MM
	Recurrence   string   `json:"recurrence"`
	Desc         string   `json:"desc"`
	Participants []string `json:"participants"` // emails
}

// toCreateMeetingInput maps the TMA request to the application input. Pure: host
// falls back to the bot user's name when blank; blank participant emails are dropped.
func toCreateMeetingInput(req tmaCreateRequest, hostFallback string) application.CreateMeetingInput {
	host := strings.TrimSpace(req.Host)
	if host == "" {
		host = hostFallback
	}
	parts := make([]postgres.MeetingParticipant, 0, len(req.Participants))
	for _, e := range req.Participants {
		if e = strings.TrimSpace(e); e != "" {
			parts = append(parts, postgres.MeetingParticipant{Email: e})
		}
	}
	return application.CreateMeetingInput{
		Dept: req.Dept, Type: req.Type, Host: host,
		Date: req.Date, Start: req.Start, End: req.End,
		Recurrence: req.Recurrence, Description: req.Desc,
		Participants: parts,
	}
}

// botUser returns the authed TMA bot_user from locals.
func botUser(c *fiber.Ctx) (postgres.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	return bu, ok
}

// TMACreateMeeting creates a non-recurring meeting for the authed TMA user.
func (a *API) TMACreateMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req tmaCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if rec := strings.TrimSpace(req.Recurrence); rec != "" && rec != string(meeting.Once) {
		return fiber.NewError(fiber.StatusBadRequest, "meetings_recurring_unsupported")
	}
	wsIDs, err := a.App.Store.ListWorkspacesWithGoogle(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if len(wsIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "meetings_not_configured")
	}
	// Use the first Google-configured workspace; multi-workspace targeting is deferred.
	workspaceID := wsIDs[0]
	organizerID, err := a.App.EnsureTMAOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	m, err := a.App.CreateMeeting(c.Context(), workspaceID, organizerID, toCreateMeetingInput(req, bu.FullName))
	if err != nil {
		if errors.Is(err, application.ErrInvalidInput) || errors.Is(err, application.ErrGoogleNotConfigured) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.App.Log.Info("tma_meeting_created",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", m.ID.String()),
		zap.String("workspace_id", workspaceID.String()))
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
}

type tmaConflictDTO struct {
	Email string `json:"email"`
	Name  string `json:"name"`  // application.Conflict.PersonName
	Title string `json:"title"` // application.Conflict.MeetingName
	Start string `json:"start"` // HH:MM Almaty
	End   string `json:"end"`
}

// toConflictDTO renders a conflict's UTC times into Almaty HH:MM. Pure.
func toConflictDTO(c application.Conflict, loc *time.Location) tmaConflictDTO {
	return tmaConflictDTO{
		Email: c.Email,
		Name:  c.PersonName,
		Title: c.MeetingName,
		Start: c.Start.In(loc).Format("15:04"),
		End:   c.End.In(loc).Format("15:04"),
	}
}

type tmaConflictRequest struct {
	Participants []string `json:"participants"`
	Date         string   `json:"date"`  // YYYY-MM-DD
	Start        string   `json:"start"` // HH:MM
	End          string   `json:"end"`   // HH:MM
	ExcludeID    string   `json:"exclude_id"`
}

// TMAConflicts reports cross-participant conflicts for a pending meeting (§4.7).
func (a *API) TMAConflicts(c *fiber.Ctx) error {
	if _, ok := botUser(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req tmaConflictRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
	start, err1 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.Start, loc)
	end, err2 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.End, loc)
	if err1 != nil || err2 != nil || !end.After(start) || len(req.Participants) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid range/participants")
	}
	exclude := uuid.Nil
	if s := strings.TrimSpace(req.ExcludeID); s != "" {
		if id, perr := uuid.Parse(s); perr == nil {
			exclude = id
		}
	}
	conflicts, err := a.App.MeetingConflicts(c.Context(), req.Participants, start, end, exclude)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	out := make([]tmaConflictDTO, 0, len(conflicts))
	for _, cf := range conflicts {
		out = append(out, toConflictDTO(cf, loc))
	}
	return c.JSON(fiber.Map{"conflicts": out})
}
