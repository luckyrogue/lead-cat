package stub

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestProviderFor(t *testing.T) {
	cal, err := NewProvider().For(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if cal == nil {
		t.Fatal("expected a CalendarService")
	}
}
