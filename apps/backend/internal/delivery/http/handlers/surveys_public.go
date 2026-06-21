package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type publicQuestion struct {
	ID        string   `json:"id"`
	Prompt    string   `json:"prompt"`
	Type      string   `json:"type"`
	Options   []string `json:"options"`
	RatingMax int      `json:"rating_max"`
	Required  bool     `json:"required"`
}

func (a *API) PublicSurveyGet(c *fiber.Ctx) error {
	token := c.Params("token")
	resp, sv, err := a.App.GetPublicSurvey(c.UserContext(), token)
	switch {
	case errors.Is(err, model.ErrResponseCompleted):
		return fiber.NewError(fiber.StatusConflict, "already_completed")
	case errors.Is(err, model.ErrSurveyClosed):
		return fiber.NewError(fiber.StatusNotFound, "survey_closed")
	case model.IsNotFound(err):
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	case err != nil:
		return internalAPIError(a.Log, "survey_get_failed", err)
	}
	qs := make([]publicQuestion, len(sv.Questions))
	for i, q := range sv.Questions {
		qs[i] = publicQuestion{
			ID: q.ID.String(), Prompt: q.Prompt, Type: string(q.Type),
			Options: q.Options, RatingMax: q.RatingMax, Required: q.Required,
		}
	}
	return c.JSON(fiber.Map{
		"survey_name": sv.Name,
		"questions":   qs,
		"booker_name": resp.BookerName,
	})
}

func (a *API) PublicSurveySubmit(c *fiber.Ctx) error {
	token := c.Params("token")
	var body struct {
		Answers []struct {
			QuestionID string `json:"question_id"`
			Value      any    `json:"value"`
		} `json:"answers"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	answers := make([]model.Answer, 0, len(body.Answers))
	for _, a := range body.Answers {
		id, err := uuid.Parse(a.QuestionID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_question_id")
		}
		answers = append(answers, model.Answer{QuestionID: id, Value: a.Value})
	}
	err := a.App.SubmitSurvey(c.UserContext(), token, answers)
	switch {
	case errors.Is(err, model.ErrResponseCompleted):
		return fiber.NewError(fiber.StatusConflict, "already_completed")
	case errors.Is(err, model.ErrSurveyClosed):
		return fiber.NewError(fiber.StatusNotFound, "survey_closed")
	case errors.Is(err, model.ErrInvalidSurvey):
		return fiber.NewError(fiber.StatusBadRequest, "invalid_answers")
	case model.IsNotFound(err):
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	case err != nil:
		return internalAPIError(a.Log, "survey_submit_failed", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
