package handlers

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func TestMapMeetingWriteError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   error
		code int
		msg  string
	}{
		{"forbidden", application.ErrForbidden, fiber.StatusForbidden, "forbidden"},
		{"invalid input", application.ErrInvalidInput, fiber.StatusBadRequest, application.ErrInvalidInput.Error()},
		{"google missing", application.ErrGoogleNotConfigured, fiber.StatusBadRequest, application.ErrGoogleNotConfigured.Error()},
		{"not editable", model.ErrMeetingNotEditable, fiber.StatusConflict, "meeting_not_editable"},
		{"internal", errors.New("boom"), fiber.StatusInternalServerError, "internal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapMeetingWriteError(tc.in)
			fe, ok := err.(*fiber.Error)
			if !ok {
				t.Fatalf("want *fiber.Error, got %T", err)
			}
			if fe.Code != tc.code || fe.Message != tc.msg {
				t.Fatalf("got code=%d msg=%q want code=%d msg=%q", fe.Code, fe.Message, tc.code, tc.msg)
			}
		})
	}
}

func TestWebUser(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		got, ok := webUser(c)
		if ok {
			t.Fatalf("unexpected web_user: %+v", got)
		}
		c.Locals("web_user", model.PlatformUser{Email: "u@x.io"})
		got, ok = webUser(c)
		if !ok || got.Email != "u@x.io" {
			t.Fatalf("got %+v ok=%v", got, ok)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
