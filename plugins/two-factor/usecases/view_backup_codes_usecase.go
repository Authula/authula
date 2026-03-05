package usecases

import (
	"context"
	"encoding/json"

	twofactor "github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type viewBackupCodesUseCase struct {
	TokenService rootservices.TokenService
	Repo         *twofactor.TwoFactorRepository
}

func NewViewBackupCodesUseCase(
	tokenService rootservices.TokenService,
	repo *twofactor.TwoFactorRepository,
) ViewBackupCodesUseCase {
	return &viewBackupCodesUseCase{
		TokenService: tokenService,
		Repo:         repo,
	}
}

func (uc *viewBackupCodesUseCase) View(ctx context.Context, userID string) ([]string, error) {
	// Get two-factor record
	record, err := uc.Repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, constants.ErrTwoFactorNotEnabled
	}

	// Decrypt backup codes
	decryptedJSON, err := uc.TokenService.Decrypt(record.BackupCodes)
	if err != nil {
		return nil, err
	}

	var codes []string
	if err := json.Unmarshal([]byte(decryptedJSON), &codes); err != nil {
		return nil, err
	}

	return codes, nil
}
