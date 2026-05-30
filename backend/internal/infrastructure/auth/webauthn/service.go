package webauthn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
)

type Service struct {
	wa    *webauthn.WebAuthn
	store *postgres.Store
	sess  *auth.SessionStore
}

func NewService(rpID, rpOrigin string, store *postgres.Store, sess *auth.SessionStore) (*Service, error) {
	u, err := url.Parse(rpOrigin)
	if err != nil {
		return nil, err
	}
	if rpID == "" {
		rpID = u.Hostname()
	}
	origin := strings.TrimSuffix(rpOrigin, "/")
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Lead Cat",
		RPID:          rpID,
		RPOrigins:     []string{origin},
	})
	if err != nil {
		return nil, err
	}
	return &Service{wa: w, store: store, sess: sess}, nil
}

type waUser struct {
	user postgres.User
	pks  []postgres.Passkey
}

func (u *waUser) WebAuthnID() []byte {
	return u.user.ID[:]
}

func (u *waUser) WebAuthnName() string {
	if u.user.Email != "" {
		return u.user.Email
	}
	if u.user.Phone != "" {
		return u.user.Phone
	}
	return u.user.AuthSub
}

func (u *waUser) WebAuthnDisplayName() string {
	return u.WebAuthnName()
}

func (u *waUser) WebAuthnCredentials() []webauthn.Credential {
	out := make([]webauthn.Credential, len(u.pks))
	for i, p := range u.pks {
		out[i] = webauthn.Credential{
			ID:              p.CredentialID,
			PublicKey:       p.PublicKey,
			AttestationType: "none",
			Authenticator: webauthn.Authenticator{
				SignCount: p.SignCount,
			},
		}
	}
	return out
}

func (s *Service) loadUser(ctx context.Context, userID uuid.UUID) (*waUser, error) {
	u, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	pks, err := s.store.ListPasskeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &waUser{user: u, pks: pks}, nil
}

func sessionKey(kind, id string) string {
	return "webauthn:" + kind + ":" + id
}

func (s *Service) BeginRegistration(ctx context.Context, userID uuid.UUID) (any, string, error) {
	u, err := s.loadUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	opts, session, err := s.wa.BeginRegistration(u)
	if err != nil {
		return nil, "", err
	}
	sid := uuid.New().String()
	if err := s.sess.Set(ctx, sessionKey("reg", sid), session, 5*time.Minute); err != nil {
		return nil, "", err
	}
	return opts, sid, nil
}

func (s *Service) FinishRegistration(ctx context.Context, userID uuid.UUID, sid string, body json.RawMessage, deviceName string) error {
	var session webauthn.SessionData
	ok, err := s.sess.Get(ctx, sessionKey("reg", sid), &session)
	if err != nil || !ok {
		return fmt.Errorf("registration session expired")
	}
	u, err := s.loadUser(ctx, userID)
	if err != nil {
		return err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(body)
	if err != nil {
		return err
	}
	cred, err := s.wa.CreateCredential(u, session, parsed)
	if err != nil {
		return err
	}
	if deviceName == "" {
		deviceName = "Passkey"
	}
	return s.store.SavePasskey(ctx, userID, cred.ID, cred.PublicKey, cred.Authenticator.SignCount, deviceName)
}

func (s *Service) BeginLogin(ctx context.Context) (any, string, error) {
	opts, session, err := s.wa.BeginDiscoverableLogin()
	if err != nil {
		return nil, "", err
	}
	sid := uuid.New().String()
	if err := s.sess.Set(ctx, sessionKey("login", sid), session, 5*time.Minute); err != nil {
		return nil, "", err
	}
	return opts, sid, nil
}

func (s *Service) FinishLogin(ctx context.Context, sid string, body json.RawMessage) (postgres.User, error) {
	var session webauthn.SessionData
	ok, err := s.sess.Get(ctx, sessionKey("login", sid), &session)
	if err != nil || !ok {
		return postgres.User{}, fmt.Errorf("login session expired")
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(body)
	if err != nil {
		return postgres.User{}, err
	}
	var foundUser postgres.User
	cred, err := s.wa.ValidateDiscoverableLogin(func(rawID, userHandle []byte) (webauthn.User, error) {
		pk, u, err := s.store.GetPasskeyByCredentialID(ctx, rawID)
		if err != nil {
			return nil, err
		}
		pks, err := s.store.ListPasskeys(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		foundUser = u
		_ = pk
		return &waUser{user: u, pks: pks}, nil
	}, session, parsed)
	if err != nil {
		return postgres.User{}, err
	}
	_ = s.store.UpdatePasskeySignCount(ctx, cred.ID, cred.Authenticator.SignCount)
	_ = s.sess.Delete(ctx, sessionKey("login", sid))
	return foundUser, nil
}
