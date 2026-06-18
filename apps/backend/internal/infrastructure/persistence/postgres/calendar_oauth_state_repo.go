package postgres

import (
	"context"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateCalendarOAuthState(ctx context.Context, st model.CalendarOAuthState) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO calendar_oauth_states (state, email, provider, verifier, expires_at)
		VALUES ($1,$2,$3,$4,$5)`,
		st.State, st.Email, st.Provider, st.Verifier, st.ExpiresAt)
	return err
}

func (s *Store) ConsumeCalendarOAuthState(ctx context.Context, state string) (model.CalendarOAuthState, error) {
	var st model.CalendarOAuthState
	err := s.pool.QueryRow(ctx, `
		DELETE FROM calendar_oauth_states
		WHERE state = $1 AND expires_at > now()
		RETURNING state, email, provider, verifier, expires_at`, state).
		Scan(&st.State, &st.Email, &st.Provider, &st.Verifier, &st.ExpiresAt)
	if err != nil {
		return model.CalendarOAuthState{}, err
	}
	return st, nil
}
