package models

import (
	"context"
	"net/http"
)

type UserService interface {
	CreateUser(user *User) error
	GetUserByID(id string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(user *User) error
}

type AccountService interface {
	CreateAccount(account *Account) error
	GetAccountByUserID(userID string) (*Account, error)
	GetAccountByProviderAndAccountID(provider ProviderType, accountID string) (*Account, error)
	UpdateAccount(account *Account) error
}

type SessionService interface {
	CreateSession(userID string, token string) (*Session, error)
	GetSessionByUserID(userID string) (*Session, error)
	GetSessionByToken(token string) (*Session, error)
	DeleteSessionByID(ID string) error
}

type VerificationService interface {
	CreateVerification(verif *Verification) error
	GetVerificationByToken(token string) (*Verification, error)
	DeleteVerification(id string) error
	IsExpired(verif *Verification) bool
}

type PasswordService interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password string, hash string) (bool, error)
}

type TokenService interface {
	GenerateToken() (string, error)
	HashToken(token string) string
	GenerateEncryptedToken() (string, error)
	EncryptToken(token string) (string, error)
	DecryptToken(encryptedToken string) (string, error)
}

type RateLimitService interface {
	Allow(ctx context.Context, key string, req *http.Request) (bool, error)
	GetClientIP(req *http.Request) string
	BuildKey(key string) string
}

type MailerService interface {
	Send(ctx context.Context, to string, subject string, body string, htmlBody string) error
}

type EventEmitter interface {
	OnUserSignedUp(user User)
	OnUserLoggedIn(user User)
	OnEmailVerified(user User)
	OnPasswordChanged(user User)
	OnEmailChanged(user User)
}

// AuthServices groups all service interfaces related to authentication
type AuthServices struct {
	Users         UserService
	Accounts      AccountService
	Sessions      SessionService
	Verifications VerificationService
	Passwords     PasswordService
	Tokens        TokenService
	RateLimits    RateLimitService
	Mailers       MailerService
}

// AuthApi defines the interface for the authentication API
type AuthApi interface {
	Services() *AuthServices
	SignUpWithEmailAndPassword(ctx context.Context, name string, email string, password string, callbackURL *string) (*SignUpResult, error)
	SignInWithEmailAndPassword(ctx context.Context, email string, password string, callbackURL *string) (*SignInResult, error)
	SignOut(ctx context.Context, sessionToken string) error
	VerifyEmail(ctx context.Context, rawToken string) (*VerifyEmailResult, error)
	SendEmailVerification(ctx context.Context, userID string, callbackURL *string) error
	ResetPassword(ctx context.Context, email string, callbackURL *string) error
	ChangePassword(ctx context.Context, rawToken string, newPassword string) error
	EmailChange(ctx context.Context, userID string, newEmail string, callbackURL *string) error
	GetMe(ctx context.Context, userID string) (*MeResult, error)
	PrepareOAuth2Login(ctx context.Context, providerName string) (*OAuth2LoginResult, error)
	SignInWithOAuth2(ctx context.Context, providerName string, code string, state string, verifier *string) (*SignInResult, error)
}

type ApiMiddleware struct {
	AdminAuth     func() func(http.Handler) http.Handler
	Auth          func() func(http.Handler) http.Handler
	OptionalAuth  func() func(http.Handler) http.Handler
	CorsAuth      func() func(http.Handler) http.Handler
	CSRF          func() func(http.Handler) http.Handler
	RateLimit     func() func(http.Handler) http.Handler
	EndpointHooks func() func(http.Handler) http.Handler
}
