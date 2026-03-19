package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

func TestGetTOTPURIUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
		setup  func(*testFixture)
		assert func(t *testing.T, uri string, err error, f *testFixture)
	}{
		{
			name:   "success",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret"}, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return("ABCDEFGHIJKLMNOPQRST", nil).Once()
			},
			assert: func(t *testing.T, uri string, err error, f *testFixture) {
				require.NoError(t, err)
				require.True(t, strings.HasPrefix(uri, "otpauth://totp/"))
				require.Contains(t, uri, "issuer=MyApp")
			},
		},
		{
			name:   "returns not enabled",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, uri string, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrTOTPNotEnabled)
				require.Empty(t, uri)
			},
		},
		{
			name:   "returns user not found",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret"}, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, uri string, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrUserNotFound)
				require.Empty(t, uri)
			},
		},
		{
			name:   "returns decrypt error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret"}, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return("", errors.New("decrypt failed")).Once()
			},
			assert: func(t *testing.T, uri string, err error, f *testFixture) {
				require.ErrorContains(t, err, "decrypt failed")
				require.Empty(t, uri)
			},
		},
		{
			name:   "returns repo error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, errors.New("repo failed")).Once()
			},
			assert: func(t *testing.T, uri string, err error, f *testFixture) {
				require.ErrorContains(t, err, "repo failed")
				require.Empty(t, uri)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}

			uc := NewGetTOTPURIUseCase(f.config, f.userSvc, f.tokenSvc, f.totpSvc, f.totpRepo)
			uri, err := uc.GetTOTPURI(context.Background(), tc.userID, "MyApp")
			tc.assert(t, uri, err, f)
		})
	}
}
