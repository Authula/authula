package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type impersonationUseCase struct {
	service             *services.ImpersonationService
	sessionStateService *services.SessionStateService
	sessionService      rootservices.SessionService
	tokenService        rootservices.TokenService
	sessionExpiresIn    time.Duration
	maxExpires          time.Duration
}

func NewImpersonationUseCase(
	service *services.ImpersonationService,
	sessionStateService *services.SessionStateService,
	sessionService rootservices.SessionService,
	tokenService rootservices.TokenService,
	sessionExpiresIn time.Duration,
	maxExpires time.Duration,
) ImpersonationUseCase {
	if maxExpires <= 0 {
		maxExpires = 15 * time.Minute
	}
	if sessionExpiresIn <= 0 {
		sessionExpiresIn = maxExpires
	}

	return &impersonationUseCase{
		service:             service,
		sessionStateService: sessionStateService,
		sessionService:      sessionService,
		tokenService:        tokenService,
		sessionExpiresIn:    sessionExpiresIn,
		maxExpires:          maxExpires,
	}
}

func (u *impersonationUseCase) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	targetUserID := strings.TrimSpace(req.TargetUserID)
	reason := strings.TrimSpace(req.Reason)

	if actorUserID == "" {
		return nil, errors.New("actor user id is required")
	}
	if targetUserID == "" {
		return nil, errors.New("target_user_id is required")
	}
	if actorUserID == targetUserID {
		return nil, errors.New("cannot impersonate yourself")
	}
	if reason == "" {
		return nil, errors.New("reason is required")
	}

	actorExists, err := u.service.UserExists(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	if !actorExists {
		return nil, errors.New("actor user not found")
	}

	targetExists, err := u.service.UserExists(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if !targetExists {
		return nil, errors.New("target user not found")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(u.maxExpires)
	maxDuration := u.maxExpires
	if req.ExpiresInSeconds != nil {
		if *req.ExpiresInSeconds <= 0 {
			return nil, errors.New("expires_in_seconds must be greater than zero")
		}
		requestedDuration := time.Duration(*req.ExpiresInSeconds) * time.Second
		if requestedDuration > u.maxExpires {
			return nil, errors.New("requested impersonation duration exceeds configured maximum")
		}
		maxDuration = requestedDuration
		expiresAt = now.Add(requestedDuration)
	}

	var impersonationSessionID *string
	var rawSessionToken *string
	if u.sessionService != nil && u.tokenService != nil {
		rawToken, err := u.tokenService.Generate()
		if err != nil {
			return nil, err
		}

		hashedToken := u.tokenService.Hash(rawToken)

		createdSession, err := u.sessionService.Create(
			ctx,
			targetUserID,
			hashedToken,
			nil,
			nil,
			maxDuration,
		)
		if err != nil {
			return nil, err
		}

		impersonationSessionID = &createdSession.ID
		rawSessionToken = &rawToken
	}

	impersonation := &types.Impersonation{
		ID:                     util.GenerateUUID(),
		ActorUserID:            actorUserID,
		TargetUserID:           targetUserID,
		ActorSessionID:         actorSessionID,
		ImpersonationSessionID: impersonationSessionID,
		Reason:                 reason,
		StartedAt:              now,
		ExpiresAt:              expiresAt,
	}

	if err := u.service.CreateImpersonation(ctx, impersonation); err != nil {
		return nil, err
	}

	if impersonationSessionID != nil && u.sessionStateService != nil {
		state := &types.AdminSessionState{
			SessionID:              *impersonationSessionID,
			ImpersonatorUserID:     &actorUserID,
			ImpersonationReason:    &reason,
			ImpersonationExpiresAt: &expiresAt,
		}
		if err := u.sessionStateService.Upsert(ctx, state); err != nil {
			return nil, err
		}
	}

	return &types.StartImpersonationResult{
		Impersonation: impersonation,
		SessionID:     impersonationSessionID,
		SessionToken:  rawSessionToken,
	}, nil
}

func (u *impersonationUseCase) StopImpersonation(ctx context.Context, actorUserID string, request types.StopImpersonationRequest) error {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return errors.New("actor user id is required")
	}

	var target *types.Impersonation
	var err error
	if request.ImpersonationID != nil && strings.TrimSpace(*request.ImpersonationID) != "" {
		target, err = u.service.GetActiveImpersonationByID(ctx, strings.TrimSpace(*request.ImpersonationID))
		if err != nil {
			return err
		}
		if target == nil {
			return errors.New("active impersonation not found")
		}
	} else {
		target, err = u.service.GetLatestActiveImpersonationByActor(ctx, actorUserID)
		if err != nil {
			return err
		}
		if target == nil {
			return errors.New("no active impersonation found")
		}
	}

	if target.ActorUserID != actorUserID {
		return errors.New("you can only stop your own impersonation sessions")
	}

	if target.ImpersonationSessionID != nil && u.sessionStateService != nil {
		now := time.Now().UTC()
		reason := "impersonation ended"
		state := &types.AdminSessionState{
			SessionID:              *target.ImpersonationSessionID,
			RevokedAt:              &now,
			RevokedReason:          &reason,
			RevokedByUserID:        &actorUserID,
			ImpersonatorUserID:     &target.ActorUserID,
			ImpersonationReason:    &target.Reason,
			ImpersonationExpiresAt: &target.ExpiresAt,
		}
		if err := u.sessionStateService.Upsert(ctx, state); err != nil {
			return err
		}
	}

	if target.ImpersonationSessionID != nil && u.sessionService != nil {
		if err := u.sessionService.Delete(ctx, *target.ImpersonationSessionID); err != nil {
			return err
		}
	}

	endedBy := actorUserID
	return u.service.EndImpersonation(ctx, target.ID, &endedBy)
}

func (u *impersonationUseCase) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return u.service.GetImpersonations(ctx)
}

func (u *impersonationUseCase) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	impersonationID = strings.TrimSpace(impersonationID)
	if impersonationID == "" {
		return nil, errors.New("impersonation_id is required")
	}

	row, err := u.service.GetImpersonationByID(ctx, impersonationID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errors.New("impersonation not found")
	}

	return row, nil
}
