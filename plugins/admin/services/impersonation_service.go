package services

import (
	"context"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	adminconstants "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
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

func (s *ImpersonationService) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	return s.impersonationRepo.GetAllImpersonations(ctx)
}

func (s *ImpersonationService) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	impersonationID = strings.TrimSpace(impersonationID)
	if impersonationID == "" {
		return nil, adminconstants.ErrBadRequest
	}

	row, err := s.impersonationRepo.GetImpersonationByID(ctx, impersonationID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, adminconstants.ErrNotFound
	}

	return row, nil
}

func (s *ImpersonationService) StartImpersonation(
	ctx context.Context,
	actorUserID string,
	actorSessionID *string,
	ipAddress *string,
	userAgent *string,
	req types.StartImpersonationRequest,
) (*types.StartImpersonationResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	targetUserID := strings.TrimSpace(req.TargetUserID)
	reason := strings.TrimSpace(req.Reason)

	if actorUserID == "" {
		return nil, adminconstants.ErrBadRequest
	}
	if targetUserID == "" {
		return nil, adminconstants.ErrBadRequest
	}
	if actorUserID == targetUserID {
		return nil, adminconstants.ErrBadRequest
	}
	if reason == "" {
		return nil, adminconstants.ErrBadRequest
	}

	actorExists, err := s.impersonationRepo.UserExists(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	if !actorExists {
		return nil, adminconstants.ErrNotFound
	}

	targetExists, err := s.impersonationRepo.UserExists(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if !targetExists {
		return nil, adminconstants.ErrNotFound
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.maxExpiresIn)
	maxDuration := s.maxExpiresIn
	if req.ExpiresInSeconds != nil {
		if *req.ExpiresInSeconds <= 0 {
			return nil, adminconstants.ErrBadRequest
		}
		requestedDuration := time.Duration(*req.ExpiresInSeconds) * time.Second
		if requestedDuration > s.maxExpiresIn {
			return nil, adminconstants.ErrBadRequest
		}
		maxDuration = requestedDuration
		expiresAt = now.Add(requestedDuration)
	}

	var impersonationSessionID *string
	var rawSessionToken *string
	if s.sessionService != nil && s.tokenService != nil {
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

	if impersonationSessionID != nil && s.sessionStateRepo != nil {
		state := &types.AdminSessionState{
			SessionID:              *impersonationSessionID,
			ImpersonatorUserID:     &actorUserID,
			ImpersonationReason:    &reason,
			ImpersonationExpiresAt: &expiresAt,
		}
		if err := s.sessionStateRepo.Upsert(ctx, state); err != nil {
			return nil, err
		}
	}

	return &types.StartImpersonationResult{
		Impersonation: impersonation,
		SessionID:     impersonationSessionID,
		SessionToken:  rawSessionToken,
	}, nil
}

func (s *ImpersonationService) StopImpersonation(ctx context.Context, actorUserID string, request types.StopImpersonationRequest) error {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return adminconstants.ErrBadRequest
	}

	var target *types.Impersonation
	var err error
	if request.ImpersonationID != nil && strings.TrimSpace(*request.ImpersonationID) != "" {
		target, err = s.impersonationRepo.GetActiveImpersonationByID(ctx, strings.TrimSpace(*request.ImpersonationID))
		if err != nil {
			return err
		}
		if target == nil {
			return adminconstants.ErrNotFound
		}
	} else {
		target, err = s.impersonationRepo.GetLatestActiveImpersonationByActor(ctx, actorUserID)
		if err != nil {
			return err
		}
		if target == nil {
			return adminconstants.ErrNotFound
		}
	}

	if target.ActorUserID != actorUserID {
		return adminconstants.ErrForbidden
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
