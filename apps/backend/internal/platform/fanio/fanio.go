package fanio

import (
	"context"

	"golang.org/x/sync/errgroup"
)

const CalendarLimit = 5

func All(ctx context.Context, limit int, n int, fn func(context.Context, int) error) error {
	if n == 0 {
		return nil
	}
	g, ctx := errgroup.WithContext(ctx)
	if limit > 0 {
		g.SetLimit(limit)
	}
	for i := 0; i < n; i++ {
		i := i
		g.Go(func() error { return fn(ctx, i) })
	}
	return g.Wait()
}

func AllBestEffort(ctx context.Context, limit int, n int, fn func(context.Context, int)) {
	if n == 0 {
		return
	}
	g := &errgroup.Group{}
	if limit > 0 {
		g.SetLimit(limit)
	}
	for i := 0; i < n; i++ {
		i := i
		g.Go(func() error {
			fn(ctx, i)
			return nil
		})
	}
	_ = g.Wait()
}

func EachString(ctx context.Context, limit int, items []string, fn func(context.Context, string) error) error {
	if len(items) == 0 {
		return nil
	}
	g, ctx := errgroup.WithContext(ctx)
	if limit > 0 {
		g.SetLimit(limit)
	}
	for _, item := range items {
		item := item
		g.Go(func() error { return fn(ctx, item) })
	}
	return g.Wait()
}

func EachStringBestEffort(ctx context.Context, limit int, items []string, fn func(context.Context, string)) {
	if len(items) == 0 {
		return
	}
	g := &errgroup.Group{}
	if limit > 0 {
		g.SetLimit(limit)
	}
	for _, item := range items {
		item := item
		g.Go(func() error {
			fn(ctx, item)
			return nil
		})
	}
	_ = g.Wait()
}
