package services

import (
	"context"
	"strings"
	"time"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

type ImpersonationService struct {
	impersonationRepo repositories.ImpersonationRepository
	sessionStateRepo  repositories.SessionStateRepository
	sessionService    rootservices.SessionService
	tokenService      rootservices.TokenService
	sessionExpiresIn  time.Duration
	maxExpiresIn      time.Duration
}

func NewImpersonationService(
	impersonationRepo repositories.ImpersonationRepository,
	sessionStateRepo repositories.SessionStateRepository,
	sessionService rootservices.SessionService,
	tokenService rootservices.TokenService,
	sessionExpiresIn time.Duration,
	maxExpiresIn time.Duration,
) *ImpersonationService {
	if maxExpiresIn <= 0 {
		maxExpiresIn = 15 * time.Minute
	}
	if sessionExpiresIn <= 0 {
		sessionExpiresIn = maxExpiresIn
	}

	return &ImpersonationService{
		impersonationRepo: impersonationRepo,
		sessionStateRepo:  sessionStateRepo,
		sessionService:    sessionService,
		tokenService:      tokenService,
		sessionExpiresIn:  sessionExpiresIn,
		maxExpiresIn:      maxExpiresIn,
	}
}

func (s *ImpersonationService) GetAllImpersonations(ctx context.Context, actor *models.Actor) ([]types.Impersonation, error) {
	return s.impersonationRepo.GetAllImpersonations(ctx)
}

func (s *ImpersonationService) GetImpersonationByID(ctx context.Context, actor *models.Actor, impersonationID string) (*types.Impersonation, error) {
	impersonationID = strings.TrimSpace(impersonationID)
	if impersonationID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	row, err := s.impersonationRepo.GetImpersonationByID(ctx, impersonationID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, coreerrors.ErrNotFound
	}

	return row, nil
}

func (s *ImpersonationService) StartImpersonation(
	ctx context.Context,
	actor *models.Actor,
	actorUserID string,
	actorSessionID *string,
	ipAddress *string,
	userAgent *string,
	req types.StartImpersonationRequest,
	impersonatorScopes []string,
	originalCookieValue string,
	originalCookieMaxAge int,
) (*types.StartImpersonationResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	targetUserID := strings.TrimSpace(req.TargetUserID)
	reason := strings.TrimSpace(req.Reason)

	if actorUserID == "" {
		return nil, coreerrors.ErrBadRequest
	}
	if targetUserID == "" {
		return nil, coreerrors.ErrBadRequest
	}
	if actorUserID == targetUserID {
		return nil, coreerrors.ErrBadRequest
	}
	if reason == "" {
		return nil, coreerrors.ErrBadRequest
	}

	actorExists, err := s.impersonationRepo.UserExists(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	if !actorExists {
		return nil, coreerrors.ErrNotFound
	}

	targetExists, err := s.impersonationRepo.UserExists(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if !targetExists {
		return nil, coreerrors.ErrNotFound
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.maxExpiresIn)
	maxDuration := s.maxExpiresIn
	if req.ExpiresInSeconds != nil {
		if *req.ExpiresInSeconds <= 0 {
			return nil, coreerrors.ErrBadRequest
		}
		requestedDuration := time.Duration(*req.ExpiresInSeconds) * time.Second
		if requestedDuration > s.maxExpiresIn {
			return nil, coreerrors.ErrBadRequest
		}
		maxDuration = requestedDuration
		expiresAt = now.Add(requestedDuration)
	}

	var impersonationSessionID *string
	var rawSessionToken *string
	if s.tokenService != nil && s.sessionService != nil {
		rawToken, err := s.tokenService.Generate()
		if err != nil {
			return nil, err
		}

		hashedToken := s.tokenService.Hash(rawToken)

		createdSession, err := s.sessionService.Create(
			ctx,
			targetUserID,
			hashedToken,
			ipAddress,
			userAgent,
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

	if err := s.impersonationRepo.CreateImpersonation(ctx, impersonation); err != nil {
		return nil, err
	}

	if impersonationSessionID != nil {
		state := &types.AdminSessionState{
			SessionID:              *impersonationSessionID,
			ImpersonatorUserID:     &actorUserID,
			ImpersonatorSessionID:  actorSessionID,
			ImpersonationReason:    &reason,
			ImpersonationExpiresAt: &expiresAt,
		}
		if err := s.sessionStateRepo.Upsert(ctx, state); err != nil {
			return nil, err
		}
	}

	// Compute .original cookie value and max age
	var originalCookieToken string
	impersonationMaxAge := 0
	if originalCookieValue != "" {
		originalCookieToken = originalCookieValue
		impersonationMaxAge = int(time.Until(expiresAt).Seconds())
		if originalCookieMaxAge > 0 && originalCookieMaxAge < impersonationMaxAge {
			impersonationMaxAge = originalCookieMaxAge
		}
		if impersonationMaxAge < 0 {
			impersonationMaxAge = 0
		}
	}

	return &types.StartImpersonationResult{
		Impersonation:        impersonation,
		SessionID:            impersonationSessionID,
		SessionToken:         rawSessionToken,
		ImpersonatorUserID:   actorUserID,
		ImpersonatorScopes:   impersonatorScopes,
		OriginalCookieToken:  originalCookieToken,
		OriginalCookieMaxAge: impersonationMaxAge,
		TargetUserID:         targetUserID,
	}, nil
}

func (s *ImpersonationService) ValidateImpersonationCookie(ctx context.Context, originalCookieValue string) (*models.Session, error) {
	hashedOriginal := s.tokenService.Hash(originalCookieValue)
	originalSession, err := s.sessionService.GetByToken(ctx, hashedOriginal)
	if err != nil || originalSession == nil {
		return nil, coreerrors.ErrForbidden
	}
	return originalSession, nil
}

func (s *ImpersonationService) StopImpersonation(ctx context.Context, actor *models.Actor, actorUserID string, request types.StopImpersonationRequest) error {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return coreerrors.ErrBadRequest
	}

	var target *types.Impersonation
	var err error
	if request.ImpersonationID != nil && strings.TrimSpace(*request.ImpersonationID) != "" {
		target, err = s.impersonationRepo.GetActiveImpersonationByID(ctx, strings.TrimSpace(*request.ImpersonationID))
		if err != nil {
			return err
		}
		if target == nil {
			return coreerrors.ErrNotFound
		}
	} else {
		target, err = s.impersonationRepo.GetLatestActiveImpersonationByActor(ctx, actorUserID)
		if err != nil {
			return err
		}
		if target == nil {
			return coreerrors.ErrNotFound
		}
	}

	if target.ActorUserID != actorUserID {
		return coreerrors.ErrForbidden
	}

	if target.ImpersonationSessionID != nil && s.sessionStateRepo != nil {
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
		if err := s.sessionStateRepo.Upsert(ctx, state); err != nil {
			return err
		}
	}

	if target.ImpersonationSessionID != nil {
		if err := s.sessionService.Delete(ctx, *target.ImpersonationSessionID); err != nil {
			return err
		}
	}

	endedBy := actorUserID
	return s.impersonationRepo.EndImpersonation(ctx, target.ID, &endedBy)
}
