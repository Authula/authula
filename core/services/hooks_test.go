package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func newUserHooks() *models.CoreServiceHooksConfig {
	return &models.CoreServiceHooksConfig{
		Users: new(models.ServiceHooks[models.User]),
	}
}

func newAccountHooks() *models.CoreServiceHooksConfig {
	return &models.CoreServiceHooksConfig{
		Accounts: new(models.ServiceHooks[models.Account]),
	}
}

func newSessionHooks() *models.CoreServiceHooksConfig {
	return &models.CoreServiceHooksConfig{
		Sessions: new(models.ServiceHooks[models.Session]),
	}
}

func newVerificationHooks() *models.CoreServiceHooksConfig {
	return &models.CoreServiceHooksConfig{
		Verifications: new(models.ServiceHooks[models.Verification]),
	}
}

func TestUserService_Create_Hooks(t *testing.T) {
	tests := []struct {
		name         string
		beforeHooks  []models.ServiceHook[models.User]
		afterHooks   []models.ServiceHook[models.User]
		wantErr      bool
		wantLogs     int
		wantMetadata map[string]any
	}{
		{
			name: "no hooks registered",
		},
		{
			name: "before create hook error blocks creation",
			beforeHooks: []models.ServiceHook[models.User]{
				func(u *models.User) error { return errors.New("blocked") },
			},
			wantErr: true,
		},
		{
			name: "before create hook can mutate user",
			beforeHooks: []models.ServiceHook[models.User]{
				func(u *models.User) error {
					u.Metadata = map[string]any{"tag": "hooked"}
					return nil
				},
			},
			wantMetadata: map[string]any{"tag": "hooked"},
		},
		{
			name: "after create hook error does not block and is logged",
			afterHooks: []models.ServiceHook[models.User]{
				func(u *models.User) error { return errors.New("notify failed") },
			},
			wantLogs: 1,
		},
		{
			name: "all after create hooks run and each error is logged",
			afterHooks: []models.ServiceHook[models.User]{
				func(u *models.User) error { return errors.New("first failed") },
				func(u *models.User) error { return errors.New("second failed") },
			},
			wantLogs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(internaltests.MockUserRepository)
			logger := new(internaltests.RecordingLogger)
			createdUser := &models.User{ID: "user-1", Email: "jane@example.com"}
			var passedUser *models.User

			repo.On("GetByEmail", mock.Anything, mock.Anything).Return(nil, nil)
			if !tt.wantErr {
				repo.On("Create", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
					passedUser = u
					return true
				})).Return(createdUser, nil).Once()
			}

			hooks := newUserHooks()
			for _, hook := range tt.beforeHooks {
				hooks.Users.RegisterBeforeCreate(hook)
			}
			for _, hook := range tt.afterHooks {
				hooks.Users.RegisterAfterCreate(hook)
			}

			svc := coreservices.NewUserService(repo, hooks, logger)
			created, err := svc.Create(ctx, "Jane Doe", "jane@example.com", false, nil, nil)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, created)
				repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, created)
			}
			if tt.wantMetadata != nil {
				assert.Equal(t, tt.wantMetadata, passedUser.Metadata)
			}
			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_Update_Hooks(t *testing.T) {
	tests := []struct {
		name        string
		beforeHooks []models.ServiceHook[models.User]
		afterHooks  []models.ServiceHook[models.User]
		wantErr     bool
		wantLogs    int
		wantName    string
	}{
		{
			name: "no hooks registered",
		},
		{
			name: "before update hook error blocks update",
			beforeHooks: []models.ServiceHook[models.User]{
				func(u *models.User) error { return errors.New("blocked") },
			},
			wantErr: true,
		},
		{
			name: "before update hook can mutate user",
			beforeHooks: []models.ServiceHook[models.User]{
				func(u *models.User) error {
					u.Name = "Mutated"
					return nil
				},
			},
			wantName: "Mutated",
		},
		{
			name: "after update hook error does not block and is logged",
			afterHooks: []models.ServiceHook[models.User]{
				func(u *models.User) error { return errors.New("sync failed") },
			},
			wantLogs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			user := &models.User{ID: "user-1", Name: "Old", Email: "jane@example.com"}
			repo := new(internaltests.MockUserRepository)
			logger := new(internaltests.RecordingLogger)
			var passedUser *models.User

			if !tt.wantErr {
				repo.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
					passedUser = u
					return true
				})).Return(user, nil).Once()
			}

			hooks := newUserHooks()
			for _, hook := range tt.beforeHooks {
				hooks.Users.RegisterBeforeUpdate(hook)
			}
			for _, hook := range tt.afterHooks {
				hooks.Users.RegisterAfterUpdate(hook)
			}

			svc := coreservices.NewUserService(repo, hooks, logger)
			updated, err := svc.Update(ctx, user)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, updated)
				repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updated)
			}
			if tt.wantName != "" {
				assert.Equal(t, tt.wantName, passedUser.Name)
			}
			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_Create_Hooks(t *testing.T) {
	tests := []struct {
		name       string
		afterHooks []models.ServiceHook[models.Session]
		wantLogs   int
	}{
		{
			name: "after create hook error does not block and is logged",
			afterHooks: []models.ServiceHook[models.Session]{
				func(s *models.Session) error { return errors.New("session hook failed") },
			},
			wantLogs: 1,
		},
		{
			name: "all after create hooks run and each error is logged",
			afterHooks: []models.ServiceHook[models.Session]{
				func(s *models.Session) error { return errors.New("first failed") },
				func(s *models.Session) error { return errors.New("second failed") },
			},
			wantLogs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(internaltests.MockSessionRepository)
			logger := new(internaltests.RecordingLogger)
			repo.On("Create", mock.Anything, mock.Anything).
				Return(&models.Session{ID: "session-1"}, nil).
				Once()

			hooks := newSessionHooks()
			for _, hook := range tt.afterHooks {
				hooks.Sessions.RegisterAfterCreate(hook)
			}

			svc := coreservices.NewSessionService(repo, nil, hooks, logger)
			created, err := svc.Create(ctx, "user-1", "hashed-token", nil, nil, time.Hour)

			assert.NoError(t, err)
			assert.NotNil(t, created)
			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
		})
	}
}

func TestVerificationService_Create_Hooks(t *testing.T) {
	tests := []struct {
		name       string
		afterHooks []models.ServiceHook[models.Verification]
		wantLogs   int
	}{
		{
			name: "after create hook error does not block and is logged",
			afterHooks: []models.ServiceHook[models.Verification]{
				func(v *models.Verification) error { return errors.New("verification hook failed") },
			},
			wantLogs: 1,
		},
		{
			name: "all after create hooks run and each error is logged",
			afterHooks: []models.ServiceHook[models.Verification]{
				func(v *models.Verification) error { return errors.New("first failed") },
				func(v *models.Verification) error { return errors.New("second failed") },
			},
			wantLogs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(internaltests.MockVerificationRepository)
			logger := new(internaltests.RecordingLogger)
			repo.On("Create", mock.Anything, mock.Anything).
				Return(&models.Verification{ID: "verification-1"}, nil).
				Once()

			hooks := newVerificationHooks()
			for _, hook := range tt.afterHooks {
				hooks.Verifications.RegisterAfterCreate(hook)
			}

			svc := coreservices.NewVerificationService(repo, nil, hooks, logger)
			created, err := svc.Create(ctx, "user-1", "hashed-token", models.TypeEmailVerification, "value", time.Hour)

			assert.NoError(t, err)
			assert.NotNil(t, created)
			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
		})
	}
}

func TestAccountService_Create_Hooks(t *testing.T) {
	tests := []struct {
		name       string
		afterHooks []models.ServiceHook[models.Account]
		wantLogs   int
	}{
		{
			name: "after create hook error does not block and is logged",
			afterHooks: []models.ServiceHook[models.Account]{
				func(a *models.Account) error { return errors.New("account hook failed") },
			},
			wantLogs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(internaltests.MockAccountRepository)
			logger := new(internaltests.RecordingLogger)
			repo.On("Create", mock.Anything, mock.Anything).
				Return(&models.Account{ID: "account-1"}, nil).
				Once()

			hooks := newAccountHooks()
			for _, hook := range tt.afterHooks {
				hooks.Accounts.RegisterAfterCreate(hook)
			}

			svc := coreservices.NewAccountService(new(models.Config), repo, new(internaltests.MockTokenRepository), hooks, logger)
			created, err := svc.Create(ctx, "user-1", "acc-1", models.AuthProviderEmail.String(), nil)

			assert.NoError(t, err)
			assert.NotNil(t, created)
			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
		})
	}
}

func TestAccountService_CreateOAuth2_UpdateHooks(t *testing.T) {
	tests := []struct {
		name        string
		beforeHooks []models.ServiceHook[models.Account]
		afterHooks  []models.ServiceHook[models.Account]
		wantErr     bool
		wantLogs    int
	}{
		{
			name: "before update hook error blocks update",
			beforeHooks: []models.ServiceHook[models.Account]{
				func(a *models.Account) error { return errors.New("blocked") },
			},
			wantErr: true,
		},
		{
			name: "after update hook error does not block and is logged",
			afterHooks: []models.ServiceHook[models.Account]{
				func(a *models.Account) error { return errors.New("account sync failed") },
			},
			wantLogs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			existing := &models.Account{ID: "acc-1", UserID: "user-1", ProviderID: "google", AccountID: "google-1"}
			repo := new(internaltests.MockAccountRepository)
			tokenRepo := new(internaltests.MockTokenRepository)
			logger := new(internaltests.RecordingLogger)

			tokenRepo.On("Encrypt", mock.Anything).Return("encrypted-token", nil)
			repo.On("GetByProviderAndAccountID", mock.Anything, "google", "google-1").Return(existing, nil)
			if !tt.wantErr {
				repo.On("Update", mock.Anything, mock.Anything).Return(existing, nil).Once()
			}

			hooks := newAccountHooks()
			for _, hook := range tt.beforeHooks {
				hooks.Accounts.RegisterBeforeUpdate(hook)
			}
			for _, hook := range tt.afterHooks {
				hooks.Accounts.RegisterAfterUpdate(hook)
			}

			svc := coreservices.NewAccountService(new(models.Config), repo, tokenRepo, hooks, logger)
			updated, err := svc.CreateOAuth2(ctx, "user-1", "google-1", "google", "access-token", nil, nil, nil, nil)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, updated)
				repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updated)
			}
			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
			tokenRepo.AssertExpectations(t)
		})
	}
}
