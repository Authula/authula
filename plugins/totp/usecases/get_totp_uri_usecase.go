package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type GetTOTPURIUseCase struct {
	Config       *types.TOTPPluginConfig
	UserService  rootservices.UserService
	TokenService rootservices.TokenService
	TOTPService  *services.TOTPService
	TOTPRepo     TOTPReadRepository
}

func NewGetTOTPURIUseCase(
	config *types.TOTPPluginConfig,
	userService rootservices.UserService,
	tokenService rootservices.TokenService,
	totpService *services.TOTPService,
	totpRepo TOTPReadRepository,
) *GetTOTPURIUseCase {
	return &GetTOTPURIUseCase{
		Config:       config,
		UserService:  userService,
		TokenService: tokenService,
		TOTPService:  totpService,
		TOTPRepo:     totpRepo,
	}
}

func (uc *GetTOTPURIUseCase) GetTOTPURI(ctx context.Context, userID string, issuer string) (string, error) {
	record, err := uc.TOTPRepo.GetByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", constants.ErrTOTPNotEnabled
	}

	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", constants.ErrUserNotFound
	}

	secret, err := uc.TokenService.Decrypt(record.Secret)
	if err != nil {
		return "", err
	}

	return uc.TOTPService.BuildURI(secret, issuer, user.Email), nil
}
