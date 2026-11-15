package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	rootservices "github.com/Authula/authula/services"
)

func newVerificationServiceForTest() (rootservices.VerificationService, *internaltests.MockVerificationRepository) {
	repo := new(internaltests.MockVerificationRepository)
	svc := coreservices.NewVerificationService(repo, nil, nil, new(internaltests.MockLogger))
	return svc, repo
}

func TestVerificationService_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name             string
		hashedToken      string
		vType            models.VerificationType
		value            string
		expiry           time.Duration
		beforeHooks      []models.ServiceHook[models.Verification]
		afterHooks       []models.ServiceHook[models.Verification]
		nilServiceHooks  bool
		nilVerifications bool
		nilLogger        bool
		repoErr          error
		expectCreate     bool
		wantErr          string
		wantLogs         int
		wantIdentifier   string
	}{
		{
			name:           "creates a verification with generated id and expiry",
			hashedToken:    "hashed-token",
			vType:          models.TypeEmailVerification,
			value:          "user@example.com",
			expiry:         time.Hour,
			expectCreate:   true,
			wantIdentifier: "user@example.com",
		},
		{
			name:           "creates a verification with zero expiry",
			hashedToken:    "hashed-token",
			vType:          models.TypePasswordResetRequest,
			value:          "user@example.com",
			expiry:         0,
			expectCreate:   true,
			wantIdentifier: "user@example.com",
		},
		{
			name:           "creates a verification with a negative expiry in the past",
			hashedToken:    "hashed-token",
			vType:          models.TypePasswordResetRequest,
			value:          "user@example.com",
			expiry:         -time.Hour,
			expectCreate:   true,
			wantIdentifier: "user@example.com",
		},
		{
			name:         "rejects an empty hashed token",
			hashedToken:  "",
			vType:        models.TypeEmailVerification,
			value:        "user@example.com",
			expiry:       time.Hour,
			expectCreate: false,
			wantErr:      "hashedToken cannot be empty",
		},
		{
			name:         "returns the repository error",
			hashedToken:  "hashed-token",
			vType:        models.TypeEmailVerification,
			value:        "user@example.com",
			expiry:       time.Hour,
			repoErr:      errors.New("db down"),
			expectCreate: true,
			wantErr:      "db down",
		},
		{
			name:        "before create hook error blocks creation",
			hashedToken: "hashed-token",
			vType:       models.TypeEmailVerification,
			value:       "user@example.com",
			expiry:      time.Hour,
			beforeHooks: []models.ServiceHook[models.Verification]{
				func(*models.Verification) error { return errors.New("blocked") },
			},
			expectCreate: false,
			wantErr:      "blocked",
		},
		{
			name:        "before create hook mutation reaches the repository",
			hashedToken: "hashed-token",
			vType:       models.TypeEmailVerification,
			value:       "user@example.com",
			expiry:      time.Hour,
			beforeHooks: []models.ServiceHook[models.Verification]{
				func(v *models.Verification) error {
					v.Identifier = "hooked@example.com"
					return nil
				},
			},
			expectCreate:   true,
			wantIdentifier: "hooked@example.com",
		},
		{
			name:        "after create hook error is logged but does not fail",
			hashedToken: "hashed-token",
			vType:       models.TypeEmailVerification,
			value:       "user@example.com",
			expiry:      time.Hour,
			afterHooks: []models.ServiceHook[models.Verification]{
				func(*models.Verification) error { return errors.New("notify failed") },
			},
			expectCreate:   true,
			wantLogs:       1,
			wantIdentifier: "user@example.com",
		},
		{
			name:        "after create hook error does not panic with a nil logger",
			hashedToken: "hashed-token",
			vType:       models.TypeEmailVerification,
			value:       "user@example.com",
			expiry:      time.Hour,
			afterHooks: []models.ServiceHook[models.Verification]{
				func(*models.Verification) error { return errors.New("notify failed") },
			},
			nilLogger:      true,
			expectCreate:   true,
			wantIdentifier: "user@example.com",
		},
		{
			name:            "works without service hooks",
			hashedToken:     "hashed-token",
			vType:           models.TypeEmailVerification,
			value:           "user@example.com",
			expiry:          time.Hour,
			nilServiceHooks: true,
			expectCreate:    true,
			wantIdentifier:  "user@example.com",
		},
		{
			name:             "works without verification hooks configured",
			hashedToken:      "hashed-token",
			vType:            models.TypeEmailVerification,
			value:            "user@example.com",
			expiry:           time.Hour,
			nilVerifications: true,
			expectCreate:     true,
			wantIdentifier:   "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(internaltests.MockVerificationRepository)
			logger := new(internaltests.RecordingLogger)
			stored := &models.Verification{ID: "stored-verification"}
			var passedVerification *models.Verification

			if tt.expectCreate {
				repo.On("Create", mock.Anything, mock.MatchedBy(func(v *models.Verification) bool {
					passedVerification = v
					return true
				})).Return(stored, tt.repoErr).Once()
			}

			var hooks *models.CoreServiceHooksConfig
			switch {
			case tt.nilServiceHooks:
				hooks = nil
			case tt.nilVerifications:
				hooks = new(models.CoreServiceHooksConfig)
			default:
				hooks = newVerificationHooks()
				for _, hook := range tt.beforeHooks {
					hooks.Verifications.RegisterBeforeCreate(hook)
				}
				for _, hook := range tt.afterHooks {
					hooks.Verifications.RegisterAfterCreate(hook)
				}
			}

			var serviceLogger models.Logger = logger
			if tt.nilLogger {
				serviceLogger = nil
			}

			svc := coreservices.NewVerificationService(repo, nil, hooks, serviceLogger)
			created, err := svc.Create(ctx, "user-1", tt.hashedToken, tt.vType, tt.value, tt.expiry)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, created)
				if !tt.expectCreate {
					repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				}
				if !tt.nilLogger {
					assert.Len(t, logger.ErrorCalls, tt.wantLogs)
				}
				repo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, created)
			require.NotNil(t, passedVerification)
			assert.Same(t, stored, created)

			_, uuidErr := uuid.Parse(passedVerification.ID)
			require.NoError(t, uuidErr)
			require.NotNil(t, passedVerification.UserID)
			assert.Equal(t, "user-1", *passedVerification.UserID)
			assert.Equal(t, tt.hashedToken, passedVerification.Token)
			assert.Equal(t, tt.vType, passedVerification.Type)
			assert.Equal(t, tt.wantIdentifier, passedVerification.Identifier)
			assert.WithinDuration(t, time.Now().UTC().Add(tt.expiry), passedVerification.ExpiresAt, 5*time.Second)

			if !tt.nilLogger {
				assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestVerificationService_Create_BeforeHookOrder(t *testing.T) {
	t.Parallel()

	var order []string

	hooks := newVerificationHooks()
	hooks.Verifications.RegisterBeforeCreate(func(*models.Verification) error {
		order = append(order, "first")
		return nil
	})
	hooks.Verifications.RegisterBeforeCreate(func(*models.Verification) error {
		order = append(order, "second")
		return errors.New("stop")
	})
	hooks.Verifications.RegisterBeforeCreate(func(*models.Verification) error {
		order = append(order, "third")
		return nil
	})

	repo := new(internaltests.MockVerificationRepository)
	svc := coreservices.NewVerificationService(repo, nil, hooks, new(internaltests.MockLogger))

	created, err := svc.Create(
		context.Background(),
		"user-1",
		"hashed-token",
		models.TypeEmailVerification,
		"user@example.com",
		time.Hour,
	)

	require.EqualError(t, err, "stop")
	require.Nil(t, created)
	assert.Equal(t, []string{"first", "second"}, order)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestVerificationService_Create_ForwardsContext(t *testing.T) {
	t.Parallel()

	type contextKey string
	const key contextKey = "test-key"

	ctx := context.WithValue(context.Background(), key, "test-value")
	repo := new(internaltests.MockVerificationRepository)
	repo.On("Create", mock.MatchedBy(func(got context.Context) bool {
		return got.Value(key) == "test-value"
	}), mock.Anything).Return(&models.Verification{ID: "verification-1"}, nil).Once()

	svc := coreservices.NewVerificationService(repo, nil, nil, new(internaltests.MockLogger))

	_, err := svc.Create(ctx, "user-1", "hashed-token", models.TypeEmailVerification, "user@example.com", time.Hour)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestVerificationService_GetByToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expected := &models.Verification{ID: "verification-1", Token: "hashed-token"}
	dbErr := errors.New("db down")

	tests := []struct {
		name     string
		verif    *models.Verification
		repoErr  error
		wantErr  error
		wantSame *models.Verification
	}{
		{
			name:     "returns the verification",
			verif:    expected,
			wantSame: expected,
		},
		{
			name:    "propagates the not found sentinel",
			repoErr: coreerrors.ErrVerificationNotFound,
			wantErr: coreerrors.ErrVerificationNotFound,
		},
		{
			name:    "propagates other errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newVerificationServiceForTest()
			repo.On("GetByToken", mock.Anything, "hashed-token").Return(tt.verif, tt.repoErr).Once()

			got, err := svc.GetByToken(ctx, "hashed-token")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Same(t, tt.wantSame, got)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestVerificationService_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name: "deletes the verification",
		},
		{
			name:    "propagates repository errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newVerificationServiceForTest()
			repo.On("Delete", mock.Anything, "verification-1").Return(tt.repoErr).Once()

			err := svc.Delete(ctx, "verification-1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestVerificationService_DeleteByUserIDAndType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		vType   models.VerificationType
		repoErr error
		wantErr error
	}{
		{
			name:  "deletes email verifications for the user",
			vType: models.TypeEmailVerification,
		},
		{
			name:  "deletes password reset requests for the user",
			vType: models.TypePasswordResetRequest,
		},
		{
			name:    "propagates repository errors",
			vType:   models.TypeEmailVerification,
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newVerificationServiceForTest()
			repo.On("DeleteByUserIDAndType", mock.Anything, "user-1", tt.vType).Return(tt.repoErr).Once()

			err := svc.DeleteByUserIDAndType(ctx, "user-1", tt.vType)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestVerificationService_IsExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "returns false when the expiry is in the future",
			expiresAt: time.Now().UTC().Add(time.Hour),
			want:      false,
		},
		{
			name:      "returns true when the expiry is in the past",
			expiresAt: time.Now().UTC().Add(-time.Hour),
			want:      true,
		},
		{
			name:      "returns true when the expiry is just in the past",
			expiresAt: time.Now().UTC().Add(-time.Nanosecond),
			want:      true,
		},
		{
			name:      "returns true for a zero expiry",
			expiresAt: time.Time{},
			want:      true,
		},
		{
			name:      "compares instants regardless of time zone",
			expiresAt: time.Now().In(time.FixedZone("UTC+5", 5*60*60)).Add(time.Hour),
			want:      false,
		},
	}

	svc := coreservices.NewVerificationService(nil, nil, nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := svc.IsExpired(&models.Verification{ExpiresAt: tt.expiresAt})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerificationService_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name: "deletes every expired verification",
		},
		{
			name:    "propagates repository errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newVerificationServiceForTest()
			repo.On("DeleteExpired", mock.Anything).Return(tt.repoErr).Once()

			err := svc.DeleteExpired(ctx)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}
