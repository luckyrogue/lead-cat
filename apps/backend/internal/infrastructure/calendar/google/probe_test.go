package google

import (
	"context"
	"errors"
	"testing"
)

func TestProbe_InvalidJSON(t *testing.T) {
	_, err := Probe(context.Background(), "{not-json", "admin@example.com", "primary")
	if !errors.Is(err, ErrJSONParse) {
		t.Fatalf("expected ErrJSONParse, got %v", err)
	}
}

func TestProbe_MissingPrivateKey(t *testing.T) {
	_, err := Probe(context.Background(), `{"type":"service_account","client_email":"x@y","token_uri":"https://oauth2.googleapis.com/token"}`, "admin@example.com", "primary")
	if !errors.Is(err, ErrJSONParse) {
		t.Fatalf("expected ErrJSONParse for missing private_key, got %v", err)
	}
}
