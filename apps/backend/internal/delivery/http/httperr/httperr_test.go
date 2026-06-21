package httperr_test

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/delivery/http/httperr"
)

func TestPublicMessage_5xxGeneric(t *testing.T) {
	err := fiber.NewError(fiber.StatusInternalServerError, "calendar: secret detail")
	if got := httperr.PublicMessage(err); got != "internal_error" {
		t.Fatalf("got %q", got)
	}
}

func TestPublicMessage_4xxPreservesCode(t *testing.T) {
	err := fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	if got := httperr.PublicMessage(err); got != "invalid_body" {
		t.Fatalf("got %q", got)
	}
}

func TestPublicMessage_Unknown(t *testing.T) {
	if got := httperr.PublicMessage(errors.New("db")); got != "internal_error" {
		t.Fatalf("got %q", got)
	}
}

