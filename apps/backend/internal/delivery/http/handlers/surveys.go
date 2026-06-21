package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func orgIDFromHeader(c *fiber.Ctx) (uuid.UUID, error) {
	return uuid.Parse(c.Get("X-Org-Id"))
}

func surveyErr(log *zap.Logger, err error) error {
	switch {
	case errors.Is(err, model.ErrInvalidSurvey):
		return fiber.NewError(fiber.StatusBadRequest, "invalid_survey")
	case errors.Is(err, model.ErrSurveyHasResponses):
		return fiber.NewError(fiber.StatusConflict, "survey_has_responses")
	case errors.Is(err, model.ErrForbidden):
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	case model.IsNotFound(err):
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	default:
		return internalAPIError(log, "survey_failed", err)
	}
}

type surveyQuestionBody struct {
	Prompt    string   `json:"prompt"`
	Type      string   `json:"type"`
	Options   []string `json:"options"`
	RatingMax int      `json:"rating_max"`
	Required  bool     `json:"required"`
}

type surveyBody struct {
	Name      string               `json:"name"`
	IsActive  bool                 `json:"is_active"`
	Questions []surveyQuestionBody `json:"questions"`
}

func (b surveyBody) toModel() model.Survey {
	qs := make([]model.SurveyQuestion, len(b.Questions))
	for i, q := range b.Questions {
		ratingMax := q.RatingMax
		if ratingMax == 0 {
			ratingMax = 5
		}
		qs[i] = model.SurveyQuestion{
			OrderIndex: i,
			Prompt:     q.Prompt,
			Type:       model.QuestionType(q.Type),
			Options:    q.Options,
			RatingMax:  ratingMax,
			Required:   q.Required,
		}
	}
	return model.Survey{Name: b.Name, IsActive: b.IsActive, Questions: qs}
}

func (a *API) SurveyList(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	list, err := a.App.ListSurveys(c.UserContext(), orgID)
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.JSON(fiber.Map{"surveys": list})
}

func (a *API) SurveyCreate(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	var body surveyBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	sv, err := a.App.CreateSurvey(c.UserContext(), orgID, body.toModel())
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.Status(fiber.StatusCreated).JSON(sv)
}

func (a *API) SurveyGet(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_id")
	}
	sv, err := a.App.GetSurvey(c.UserContext(), orgID, id)
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.JSON(sv)
}

func (a *API) SurveyUpdate(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_id")
	}
	var body surveyBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	sv, err := a.App.UpdateSurvey(c.UserContext(), orgID, id, body.toModel())
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.JSON(sv)
}

func (a *API) SurveyDelete(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_id")
	}
	if err := a.App.DeleteSurvey(c.UserContext(), orgID, id); err != nil {
		return surveyErr(a.Log, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
