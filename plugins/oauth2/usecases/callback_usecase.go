package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"golang.org/x/oauth2"

	coreservices "github.com/Authula/authula/core/services"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
	rootservices "github.com/Authula/authula/services"
)

const maxUserNameLength = 100

type callbackUseCase struct {
	GlobalConfig     *models.Config
	ProviderRegistry *services.ProviderRegistry
	Logger           models.Logger
	HMACKey          []byte
	UserService      rootservices.UserService
	AccountService   rootservices.AccountService
	SessionService   rootservices.SessionService
	TokenService     rootservices.TokenService
}

func NewCallbackUseCase(
	globalConfig *models.Config,
	registry *services.ProviderRegistry,
	logger models.Logger,
	hmacKey []byte,
	userService rootservices.UserService,
	accountService rootservices.AccountService,
	sessionService rootservices.SessionService,
	tokenService rootservices.TokenService,
) *callbackUseCase {
	return &callbackUseCase{
		GlobalConfig:     globalConfig,
		ProviderRegistry: registry,
		Logger:           logger,
		HMACKey:          hmacKey,
		UserService:      userService,
		AccountService:   accountService,
		SessionService:   sessionService,
		TokenService:     tokenService,
	}
}

func (uc *callbackUseCase) Callback(ctx context.Context, req *types.CallbackRequest, ipAddress *string, userAgent *string) (*types.CallbackResult, error) {
	if req.Error != "" {
		uc.Logger.Warn("oauth provider returned an error", "provider", req.ProviderID, "error", req.Error)
		return nil, constants.ErrAccessDenied
	}

	providerID := strings.ToLower(strings.TrimSpace(req.ProviderID))
	oauthProvider, exists := uc.ProviderRegistry.Get(providerID)
	if !exists {
		return nil, constants.ErrProviderNotFound
	}

	state, err := services.ParseState(req.StateCookie, uc.HMACKey, constants.FlowCookieTTL)
	if err != nil {
		uc.Logger.Warn("invalid oauth state", "provider", providerID, "error", err)
		return nil, constants.ErrInvalidState
	}
	if state.Provider != providerID {
		return nil, constants.ErrStateProviderMismatch
	}
	if state.State != req.State {
		return nil, constants.ErrInvalidState
	}

	switch state.Purpose {
	case services.StatePurposeLink:
		if req.ActorID == "" || state.UserID == "" || state.UserID != req.ActorID {
			return nil, constants.ErrUnauthorized
		}
	case services.StatePurposeLogin:
		if req.ActorID != "" {
			// A signed-in browser must restart the flow as an explicit link.
			return nil, constants.ErrUnauthorized
		}
	}

	if req.Code == "" {
		return nil, constants.ErrInvalidCode
	}

	token, err := uc.exchange(ctx, oauthProvider, req)
	if err != nil {
		if errors.Is(err, constants.ErrPKCEVerifierMissing) {
			return nil, err
		}
		uc.Logger.Error("failed to exchange oauth code", "provider", providerID, "error", err)
		return nil, fmt.Errorf("%w: %v", constants.ErrProviderFailure, err)
	}

	userInfo, err := oauthProvider.GetUserInfo(ctx, token)
	if err != nil {
		uc.Logger.Error("failed to get oauth user info", "provider", providerID, "error", err)
		return nil, fmt.Errorf("%w: %v", constants.ErrProviderFailure, err)
	}

	uc.applyUserPayload(userInfo, req.User)

	if err := validateUserInfo(userInfo); err != nil {
		return nil, err
	}

	user, existingAccount, created, err := uc.resolveUser(ctx, providerID, userInfo, state, req)
	if err != nil {
		return nil, err
	}

	if err := uc.persistAccount(ctx, providerID, user, userInfo, token, existingAccount, created); err != nil {
		if created {
			uc.deleteOrphanUser(ctx, user.ID)
		}
		return nil, err
	}

	if state.Purpose == services.StatePurposeLink && userInfo.EmailVerified && user.Email == userInfo.Email && !user.EmailVerified {
		user.EmailVerified = true
		updated, updateErr := uc.UserService.Update(ctx, user)
		if updateErr != nil {
			user.EmailVerified = false
			uc.Logger.Error("failed to mark user email verified after link", "error", updateErr)
		} else {
			user = updated
		}
	}

	result := &types.CallbackResult{User: user}

	if state.Purpose == services.StatePurposeLink {
		session, err := uc.currentSession(ctx, req)
		if err != nil {
			return nil, err
		}
		result.Session = session
		result.ReusedSession = true
	} else {
		sessionToken, session, err := uc.createSession(ctx, user.ID, ipAddress, userAgent)
		if err != nil {
			return nil, err
		}
		result.Session = session
		result.SessionToken = sessionToken
	}

	result.RedirectTo = uc.redirectTo(req)

	return result, nil
}

func (uc *callbackUseCase) exchange(ctx context.Context, provider types.OAuth2Provider, req *types.CallbackRequest) (*oauth2.Token, error) {
	options := make([]oauth2.AuthCodeOption, 0, 1)
	if provider.RequiresPKCE() {
		if req.VerifierCookie == "" {
			return nil, constants.ErrPKCEVerifierMissing
		}
		verifier, err := services.ValidateCookie(req.VerifierCookie, uc.HMACKey, constants.FlowCookieTTL)
		if err != nil || verifier == "" {
			return nil, constants.ErrPKCEVerifierMissing
		}
		options = append(options, oauth2.VerifierOption(verifier))
	}

	token, err := provider.Exchange(ctx, req.Code, options...)
	if err != nil {
		return nil, err
	}
	if token == nil || strings.TrimSpace(token.AccessToken) == "" {
		return nil, fmt.Errorf("empty token response")
	}
	return token, nil
}

func validateUserInfo(info *types.UserInfo) error {
	if info == nil {
		return constants.ErrProviderFailure
	}
	if strings.TrimSpace(info.ProviderAccountID) == "" || info.ProviderAccountID == "<nil>" || info.ProviderAccountID == "null" {
		return constants.ErrProviderAccountIDRequired
	}
	if strings.TrimSpace(info.Email) == "" {
		return constants.ErrEmailRequired
	}
	return nil
}

func (uc *callbackUseCase) resolveUser(ctx context.Context, providerID string, userInfo *types.UserInfo, state *services.StatePayload, req *types.CallbackRequest) (*models.User, *models.Account, bool, error) {
	existingAccount, err := uc.AccountService.GetByProviderAndAccountID(ctx, providerID, userInfo.ProviderAccountID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to check existing account: %w", err)
	}

	if existingAccount != nil {
		if req.ActorID != "" && existingAccount.UserID != req.ActorID {
			return nil, nil, false, constants.ErrAccountLinkedToAnotherUser
		}

		user, err := uc.UserService.GetByID(ctx, existingAccount.UserID)
		if err != nil {
			return nil, nil, false, fmt.Errorf("failed to load account owner: %w", err)
		}
		if user == nil {
			return nil, nil, false, constants.ErrAccountConflict
		}
		return user, existingAccount, false, nil
	}

	if state.Purpose == services.StatePurposeLink {
		user, err := uc.resolveLinkUser(ctx, providerID, req.ActorID)
		return user, nil, false, err
	}

	user, err := uc.UserService.GetByEmail(ctx, userInfo.Email)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to look up user by email: %w", err)
	}
	if user != nil {
		if !userInfo.EmailVerified {
			return nil, nil, false, constants.ErrEmailNotVerified
		}
		return user, nil, false, nil
	}

	created, err := uc.UserService.Create(ctx, fallbackName(userInfo), userInfo.Email, userInfo.EmailVerified, imagePointer(userInfo.Picture), nil)
	if err != nil {
		if errors.Is(err, coreservices.ErrEmailInUse) {
			return nil, nil, false, constants.ErrEmailInUse
		}
		uc.Logger.Error("failed to create oauth user", "error", err)
		return nil, nil, false, fmt.Errorf("failed to create user: %w", err)
	}
	return created, nil, true, nil
}

func (uc *callbackUseCase) resolveLinkUser(ctx context.Context, providerID, actorID string) (*models.User, error) {
	user, err := uc.UserService.GetByID(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("failed to load signed-in user: %w", err)
	}
	if user == nil {
		return nil, constants.ErrUnauthorized
	}

	linked, err := uc.AccountService.GetByUserIDAndProvider(ctx, actorID, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check linked accounts: %w", err)
	}
	if linked != nil {
		return nil, constants.ErrAccountConflict
	}

	return user, nil
}

func (uc *callbackUseCase) persistAccount(ctx context.Context, providerID string, user *models.User, userInfo *types.UserInfo, token *oauth2.Token, existingAccount *models.Account, createdUser bool) error {
	expiry := tokenExpiry(token)
	refreshToken := optionalToken(token.RefreshToken)
	scope := tokenScope(token)

	if existingAccount != nil {
		if existingAccount.UserID != user.ID {
			return constants.ErrAccountLinkedToAnotherUser
		}
		if _, err := uc.AccountService.UpdateOAuth2Tokens(ctx, existingAccount.ID, token.AccessToken, refreshToken, expiry, nil, scope); err != nil {
			if errors.Is(err, coreservices.ErrTokenExpired) {
				return constants.ErrTokenExpired
			}
			return fmt.Errorf("failed to update oauth tokens: %w", err)
		}
		return nil
	}

	_, err := uc.AccountService.CreateOAuth2(ctx, user.ID, userInfo.ProviderAccountID, providerID, token.AccessToken, refreshToken, expiry, nil, scope)
	if err == nil {
		return nil
	}

	// A unique constraint may have been hit by a concurrent callback. Re-resolve the owner.
	racedAccount, lookupErr := uc.AccountService.GetByProviderAndAccountID(ctx, providerID, userInfo.ProviderAccountID)
	if lookupErr == nil && racedAccount != nil {
		if racedAccount.UserID != user.ID {
			return constants.ErrAccountLinkedToAnotherUser
		}
		if _, updateErr := uc.AccountService.UpdateOAuth2Tokens(ctx, racedAccount.ID, token.AccessToken, refreshToken, expiry, nil, scope); updateErr == nil {
			return nil
		}
	}

	if errors.Is(err, coreservices.ErrAccountLinkedToAnotherUser) {
		return constants.ErrAccountLinkedToAnotherUser
	}
	if errors.Is(err, coreservices.ErrTokenExpired) {
		return constants.ErrTokenExpired
	}

	uc.Logger.Error("failed to create oauth account", "provider", providerID, "error", err)
	if createdUser {
		return fmt.Errorf("%w: %v", constants.ErrAccountConflict, err)
	}
	return fmt.Errorf("failed to link oauth account: %w", err)
}

func (uc *callbackUseCase) currentSession(ctx context.Context, req *types.CallbackRequest) (*models.Session, error) {
	if req.SessionID == "" {
		return nil, constants.ErrUnauthorized
	}

	session, err := uc.SessionService.GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load current session: %w", err)
	}
	if session == nil || session.UserID != req.ActorID || !session.ExpiresAt.After(time.Now()) {
		return nil, constants.ErrSessionNotFound
	}
	return session, nil
}

func (uc *callbackUseCase) createSession(ctx context.Context, userID string, ipAddress, userAgent *string) (string, *models.Session, error) {
	sessionToken, err := uc.TokenService.Generate()
	if err != nil {
		uc.Logger.Error("failed to generate session token", "error", err)
		return "", nil, err
	}

	hashedSessionToken := uc.TokenService.Hash(sessionToken)
	session, err := uc.SessionService.Create(
		ctx,
		userID,
		hashedSessionToken,
		ipAddress,
		userAgent,
		uc.GlobalConfig.Session.ExpiresIn,
	)
	if err != nil {
		uc.Logger.Error("failed to create session", "error", err)
		return "", nil, err
	}

	return sessionToken, session, nil
}

func (uc *callbackUseCase) deleteOrphanUser(ctx context.Context, userID string) {
	if err := uc.UserService.Delete(ctx, userID); err != nil {
		uc.Logger.Error("failed to delete orphan oauth user", "user_id", userID, "error", err)
	}
}

func (uc *callbackUseCase) redirectTo(req *types.CallbackRequest) string {
	if req.RedirectCookie == "" {
		return ""
	}

	raw, err := services.ValidateCookie(req.RedirectCookie, uc.HMACKey, constants.FlowCookieTTL)
	if err != nil {
		return ""
	}

	validated, err := services.ValidateRedirectTo(raw, uc.GlobalConfig.Security.TrustedOrigins)
	if err != nil {
		return ""
	}
	return validated
}

func tokenExpiry(token *oauth2.Token) *time.Time {
	if token.Expiry.IsZero() {
		return nil
	}
	expiry := token.Expiry
	return &expiry
}

func optionalToken(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

// tokenScope returns the scope granted by the provider, if present.
func tokenScope(token *oauth2.Token) *string {
	if token == nil {
		return nil
	}

	var scope string
	switch value := token.Extra("scope").(type) {
	case string:
		scope = value
	case []string:
		scope = strings.Join(value, " ")
	case []any:
		parts := make([]string, 0, len(value))
		for _, part := range value {
			if str, ok := part.(string); ok {
				parts = append(parts, str)
			}
		}
		scope = strings.Join(parts, " ")
	}

	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil
	}
	return &scope
}

func imagePointer(image string) *string {
	if strings.TrimSpace(image) == "" {
		return nil
	}
	return &image
}

func fallbackName(info *types.UserInfo) string {
	if name := strings.TrimSpace(info.Name); name != "" {
		return name
	}
	if local, _, found := strings.Cut(info.Email, "@"); found && local != "" {
		return local
	}
	return "User"
}

// applyUserPayload merges the user payload some providers post alongside the
// callback, which they send on first consent only. Only the name is taken: the
// payload is submitted by the browser and is not signed by the provider, so it
// must never decide which account the login resolves to. A malformed payload is
// logged rather than fatal, since the token already identifies the account.
func (uc *callbackUseCase) applyUserPayload(userInfo *types.UserInfo, payload string) {
	if payload == "" || userInfo == nil {
		return
	}

	var user struct {
		Name struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"name"`
	}

	if err := json.Unmarshal([]byte(payload), &user); err != nil {
		uc.Logger.Warn("failed to parse provider user payload", "error", err)
		return
	}

	if userInfo.Name == "" {
		userInfo.Name = sanitizeName(user.Name.FirstName + " " + user.Name.LastName)
	}
}

// sanitizeName drops control characters, collapses whitespace and caps the
// length. Whitespace is mapped to a space before control characters are
// removed so tab- and newline-separated words do not get glued together.
func sanitizeName(name string) string {
	printable := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)

	cleaned := strings.Join(strings.Fields(printable), " ")

	runes := []rune(cleaned)
	if len(runes) > maxUserNameLength {
		return string(runes[:maxUserNameLength])
	}

	return cleaned
}
