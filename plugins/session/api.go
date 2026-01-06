package session

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// SessionAPI defines the programmatic API for the session plugin.
type SessionAPI interface {
	CreateSession(ctx context.Context, userID string) (sessionToken string, err error)
	GetSession(ctx context.Context, token string) (*models.Session, error)
	ValidateSession(ctx context.Context, token string) (valid bool, err error)
	DestroySession(ctx context.Context, sessionID string) error
	DestroyAllUserSessions(ctx context.Context, userID string) error
}

// sessionPluginAPI implements the SessionAPI interface.
type sessionPluginAPI struct {
	plugin *SessionPlugin
}

// newSessionAPI creates a new API instance for the session plugin.
func newSessionAPI(plugin *SessionPlugin) SessionAPI {
	return &sessionPluginAPI{
		plugin: plugin,
	}
}

// CreateSession creates a new session for the given user ID.
func (api *sessionPluginAPI) CreateSession(ctx context.Context, userID string) (sessionToken string, err error) {
	rawToken, err := api.plugin.tokenService.Generate()
	if err != nil {
		return "", err
	}
	hashedToken := api.plugin.tokenService.Hash(rawToken)

	_, err = api.plugin.sessionService.Create(ctx, userID, hashedToken, nil, nil, api.plugin.config.MaxAge)
	if err != nil {
		return "", err
	}

	return rawToken, nil
}

// GetSession retrieves session information by token.
func (api *sessionPluginAPI) GetSession(ctx context.Context, token string) (*models.Session, error) {
	hashed := api.plugin.tokenService.Hash(token)
	return api.plugin.sessionService.GetByToken(ctx, hashed)
}

// ValidateSession checks if a session is valid.
func (api *sessionPluginAPI) ValidateSession(ctx context.Context, token string) (valid bool, err error) {
	hashed := api.plugin.tokenService.Hash(token)
	session, err := api.plugin.sessionService.GetByToken(ctx, hashed)
	if err != nil {
		return false, err
	}

	return session != nil, nil
}

// DestroySession destroys/invalidates a session.
func (api *sessionPluginAPI) DestroySession(ctx context.Context, sessionID string) error {
	return api.plugin.sessionService.Delete(ctx, sessionID)
}

// DestroyAllUserSessions destroys all sessions for a user.
func (api *sessionPluginAPI) DestroyAllUserSessions(ctx context.Context, userID string) error {
	return api.plugin.sessionService.DeleteAllByUserID(ctx, userID)
}
