package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

func verifyPassword(
	ctx context.Context,
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	userID, password string,
) error {
	if password == "" {
		return constants.ErrPasswordRequired
	}
	account, err := accountService.GetByUserIDAndProvider(ctx, userID, models.AuthProviderEmail.String())
	if err != nil || account == nil {
		return constants.ErrAccountNotFound
	}
	if account.Password == nil || !passwordService.Verify(password, *account.Password) {
		return constants.ErrInvalidPassword
	}
	return nil
}
