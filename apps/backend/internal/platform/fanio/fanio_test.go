package fanio

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestAll_OK(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	if err := All(context.Background(), 2, 4, func(context.Context, int) error {
		n.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 4 {
		t.Fatalf("got %d", n.Load())
	}
}

func TestAll_FirstErrorCancels(t *testing.T) {
	t.Parallel()
	err := All(context.Background(), 2, 3, func(_ context.Context, i int) error {
		if i == 1 {
			return errors.New("boom")
		}
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEachStringBestEffort(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	EachStringBestEffort(context.Background(), 2, []string{"a", "b", "c"}, func(context.Context, string) {
		n.Add(1)
	})
	if n.Load() != 3 {
		t.Fatalf("got %d", n.Load())
	}
}
