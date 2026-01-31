package gobetterauth

import (
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

// CORE

func (auth *Auth) GetUserService() services.UserService {
	userService, ok := auth.ServiceRegistry.Get(models.ServiceUser.String()).(services.UserService)
	if !ok {
		auth.logger.Error("user service not available in service registry")
		return nil
	}
	return userService
}

func (auth *Auth) GetAccountService() services.AccountService {
	accountService, ok := auth.ServiceRegistry.Get(models.ServiceAccount.String()).(services.AccountService)
	if !ok {
		auth.logger.Error("account service not available in service registry")
		return nil
	}
	return accountService
}

func (auth *Auth) GetSessionService() services.SessionService {
	sessionService, ok := auth.ServiceRegistry.Get(models.ServiceSession.String()).(services.SessionService)
	if !ok {
		auth.logger.Error("session service not available in service registry")
		return nil
	}
	return sessionService
}

func (auth *Auth) GetVerificationService() services.VerificationService {
	verificationService, ok := auth.ServiceRegistry.Get(models.ServiceVerification.String()).(services.VerificationService)
	if !ok {
		auth.logger.Error("verification service not available in service registry")
		return nil
	}
	return verificationService
}

func (auth *Auth) GetTokenService() services.TokenService {
	tokenService, ok := auth.ServiceRegistry.Get(models.ServiceToken.String()).(services.TokenService)
	if !ok {
		auth.logger.Error("token service not available in service registry")
		return nil
	}
	return tokenService
}

// EMAIL

func (auth *Auth) GetPasswordService() services.PasswordService {
	passwordService, ok := auth.ServiceRegistry.Get(models.ServicePassword.String()).(services.PasswordService)
	if !ok {
		auth.logger.Error("password service not available in service registry")
		return nil
	}
	return passwordService
}

func (auth *Auth) GetMailerService() services.MailerService {
	mailerService, ok := auth.ServiceRegistry.Get(models.ServiceMailer.String()).(services.MailerService)
	if !ok {
		auth.logger.Error("mailer service not available in service registry")
		return nil
	}
	return mailerService
}

// JWT

func (auth *Auth) GetJWTService() services.JWTService {
	jwtService, ok := auth.ServiceRegistry.Get(models.ServiceJWT.String()).(services.JWTService)
	if !ok {
		auth.logger.Error("jwt service not available in service registry")
		return nil
	}
	return jwtService
}
