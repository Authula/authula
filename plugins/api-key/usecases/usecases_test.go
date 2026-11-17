package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	apiconstants "github.com/Authula/authula/plugins/api-key/constants"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
)

type useCasesFixture struct {
	service    *apiKeyTests.MockApiKeyService
	authorizer *internaltests.MockAuthorizer
	useCases   *UseCases
}

func newUseCasesFixture() *useCasesFixture {
	service := &apiKeyTests.MockApiKeyService{}
	authorizer := &internaltests.MockAuthorizer{}
	return &useCasesFixture{
		service:    service,
		authorizer: authorizer,
		useCases:   NewUseCases(service, authorizer),
	}
}

func (f *useCasesFixture) assertExpectations(t *testing.T) {
	t.Helper()
	f.service.AssertExpectations(t)
	f.authorizer.AssertExpectations(t)
}

func useCaseActor() *models.Actor {
	return &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{"*"}}
}

func TestUseCasesCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		req       types.CreateApiKeyRequest
		setup     func(*useCasesFixture)
		wantErr   error
		wantNil   bool
		wantValue string
	}{
		{
			name: "user_owner_skips_authorization",
			req:  types.CreateApiKeyRequest{OwnerType: types.OwnerTypeUser, OwnerID: "user-1"},
			setup: func(f *useCasesFixture) {
				f.service.On("Create", mock.Anything, useCaseActor(), mock.Anything).Return(&types.CreateApiKeyResponse{RawApiKey: "raw"}, nil).Once()
			},
			wantValue: "raw",
		},
		{
			name: "service_error_is_returned",
			req:  types.CreateApiKeyRequest{OwnerType: types.OwnerTypeUser, OwnerID: "user-1"},
			setup: func(f *useCasesFixture) {
				f.service.On("Create", mock.Anything, useCaseActor(), mock.Anything).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
			wantNil: true,
		},
		{
			name: "org_owner_authorization_failure_short_circuits",
			req:  types.CreateApiKeyRequest{OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"},
			setup: func(f *useCasesFixture) {
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyCreate).Return(coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
			wantNil: true,
		},
		{
			name: "org_owner_authorized_delegates_to_service",
			req:  types.CreateApiKeyRequest{OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"},
			setup: func(f *useCasesFixture) {
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyCreate).Return(nil).Once()
				f.service.On("Create", mock.Anything, useCaseActor(), mock.Anything).Return(&types.CreateApiKeyResponse{RawApiKey: "org-raw"}, nil).Once()
			},
			wantValue: "org-raw",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newUseCasesFixture()
			tc.setup(fixture)

			resp, err := fixture.useCases.Create(context.Background(), useCaseActor(), tc.req)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			if tc.wantNil {
				assert.Nil(t, resp)
			} else {
				require.NotNil(t, resp)
				assert.Equal(t, tc.wantValue, resp.RawApiKey)
			}
			fixture.assertExpectations(t)
		})
	}
}

func TestUseCasesGetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*useCasesFixture)
		wantErr error
		wantNil bool
	}{
		{
			name: "service_error",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
			wantNil: true,
		},
		{
			name: "missing_key_returns_nil_without_authorization",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
			},
			wantNil: true,
		},
		{
			name: "user_key_needs_no_scope",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser}, nil).Once()
			},
		},
		{
			name: "org_key_authorization_failure",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization}, nil).Once()
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyRead).Return(coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
			wantNil: true,
		},
		{
			name: "org_key_authorized",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization}, nil).Once()
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyRead).Return(nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newUseCasesFixture()
			tc.setup(fixture)

			apiKey, err := fixture.useCases.GetByID(context.Background(), useCaseActor(), "api-key-1")

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			if tc.wantNil {
				assert.Nil(t, apiKey)
			} else {
				require.NotNil(t, apiKey)
			}
			fixture.assertExpectations(t)
		})
	}
}

func TestUseCasesGetAll(t *testing.T) {
	t.Parallel()

	orgType := types.OwnerTypeOrganization
	userType := types.OwnerTypeUser

	tests := []struct {
		name    string
		req     types.GetApiKeysRequest
		setup   func(*useCasesFixture)
		wantErr error
	}{
		{
			name: "nil_owner_type_skips_authorization",
			req:  types.GetApiKeysRequest{},
			setup: func(f *useCasesFixture) {
				f.service.On("GetAll", mock.Anything, useCaseActor(), mock.Anything).Return(&types.GetAllApiKeysResponse{Total: 1}, nil).Once()
			},
		},
		{
			name: "user_owner_type_skips_authorization",
			req:  types.GetApiKeysRequest{OwnerType: &userType},
			setup: func(f *useCasesFixture) {
				f.service.On("GetAll", mock.Anything, useCaseActor(), mock.Anything).Return(&types.GetAllApiKeysResponse{Total: 1}, nil).Once()
			},
		},
		{
			name: "org_owner_type_authorization_failure",
			req:  types.GetApiKeysRequest{OwnerType: &orgType},
			setup: func(f *useCasesFixture) {
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyList).Return(coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "org_owner_type_authorized",
			req:  types.GetApiKeysRequest{OwnerType: &orgType},
			setup: func(f *useCasesFixture) {
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyList).Return(nil).Once()
				f.service.On("GetAll", mock.Anything, useCaseActor(), mock.Anything).Return(&types.GetAllApiKeysResponse{Total: 2}, nil).Once()
			},
		},
		{
			name: "service_error",
			req:  types.GetApiKeysRequest{},
			setup: func(f *useCasesFixture) {
				f.service.On("GetAll", mock.Anything, useCaseActor(), mock.Anything).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newUseCasesFixture()
			tc.setup(fixture)

			resp, err := fixture.useCases.GetAll(context.Background(), useCaseActor(), tc.req)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
			fixture.assertExpectations(t)
		})
	}
}

func TestUseCasesUpdate(t *testing.T) {
	t.Parallel()

	req := types.UpdateApiKeyData{Name: new("updated")}

	tests := []struct {
		name    string
		setup   func(*useCasesFixture)
		wantErr error
		wantNil bool
	}{
		{
			name: "lookup_error",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
			wantNil: true,
		},
		{
			name: "missing_key_returns_not_found",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
			wantNil: true,
		},
		{
			name: "org_key_authorization_failure",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization}, nil).Once()
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyUpdate).Return(coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
			wantNil: true,
		},
		{
			name: "org_key_authorized_delegates",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization}, nil).Once()
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyUpdate).Return(nil).Once()
				f.service.On("Update", mock.Anything, useCaseActor(), "api-key-1", req).Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			},
		},
		{
			name: "user_key_delegates_and_propagates_error",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser}, nil).Once()
				f.service.On("Update", mock.Anything, useCaseActor(), "api-key-1", req).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newUseCasesFixture()
			tc.setup(fixture)

			apiKey, err := fixture.useCases.Update(context.Background(), useCaseActor(), "api-key-1", req)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			if tc.wantNil {
				assert.Nil(t, apiKey)
			} else {
				require.NotNil(t, apiKey)
			}
			fixture.assertExpectations(t)
		})
	}
}

func TestUseCasesDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*useCasesFixture)
		wantErr error
	}{
		{
			name: "lookup_error",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "missing_key_is_a_noop",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
			},
		},
		{
			name: "org_key_authorization_failure",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization}, nil).Once()
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyDelete).Return(coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "org_key_authorized_delegates",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization}, nil).Once()
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyDelete).Return(nil).Once()
				f.service.On("Delete", mock.Anything, useCaseActor(), "api-key-1").Return(nil).Once()
			},
		},
		{
			name: "user_key_delegates_and_propagates_error",
			setup: func(f *useCasesFixture) {
				f.service.On("GetByID", mock.Anything, useCaseActor(), "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser}, nil).Once()
				f.service.On("Delete", mock.Anything, useCaseActor(), "api-key-1").Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newUseCasesFixture()
			tc.setup(fixture)

			err := fixture.useCases.Delete(context.Background(), useCaseActor(), "api-key-1")

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			fixture.assertExpectations(t)
		})
	}
}

func TestUseCasesDeleteAllByOwner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ownerType string
		ownerID   string
		setup     func(*useCasesFixture)
		wantErr   error
	}{
		{
			name:      "user_owner_delegates_without_authorization",
			ownerType: types.OwnerTypeUser,
			ownerID:   "user-1",
			setup: func(f *useCasesFixture) {
				f.service.On("DeleteAllByOwner", mock.Anything, useCaseActor(), types.OwnerTypeUser, "user-1").Return(nil).Once()
			},
		},
		{
			name:      "org_owner_authorization_failure",
			ownerType: types.OwnerTypeOrganization,
			ownerID:   "org-1",
			setup: func(f *useCasesFixture) {
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyDelete).Return(coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:      "org_owner_authorized_delegates",
			ownerType: types.OwnerTypeOrganization,
			ownerID:   "org-1",
			setup: func(f *useCasesFixture) {
				f.authorizer.On("AuthorizeScope", mock.Anything, useCaseActor(), apiconstants.OrgApiKeyDelete).Return(nil).Once()
				f.service.On("DeleteAllByOwner", mock.Anything, useCaseActor(), types.OwnerTypeOrganization, "org-1").Return(nil).Once()
			},
		},
		{
			name:      "service_error_propagated",
			ownerType: types.OwnerTypeUser,
			ownerID:   "user-1",
			setup: func(f *useCasesFixture) {
				f.service.On("DeleteAllByOwner", mock.Anything, useCaseActor(), types.OwnerTypeUser, "user-1").Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newUseCasesFixture()
			tc.setup(fixture)

			err := fixture.useCases.DeleteAllByOwner(context.Background(), useCaseActor(), tc.ownerType, tc.ownerID)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			fixture.assertExpectations(t)
		})
	}
}

func TestUseCasesDelegatingMethods(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	verifyReq := types.VerifyApiKeyRequest{Key: "raw-key"}

	tests := []struct {
		name    string
		call    func(*UseCases) error
		setup   func(*useCasesFixture)
		wantErr error
	}{
		{
			name: "delete_expired_success",
			call: func(u *UseCases) error {
				return u.DeleteExpired(context.Background())
			},
			setup: func(f *useCasesFixture) {
				f.service.On("DeleteExpired", mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "delete_expired_error",
			call: func(u *UseCases) error {
				return u.DeleteExpired(context.Background())
			},
			setup: func(f *useCasesFixture) {
				f.service.On("DeleteExpired", mock.Anything).Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "record_last_request_success",
			call: func(u *UseCases) error {
				apiKey, err := u.RecordLastRequest(context.Background(), "api-key-1", now)
				if err != nil {
					return err
				}
				if apiKey == nil || apiKey.LastRequestedAt == nil || !apiKey.LastRequestedAt.Equal(now) {
					return errors.New("unexpected api key")
				}
				return nil
			},
			setup: func(f *useCasesFixture) {
				f.service.On("RecordLastRequest", mock.Anything, "api-key-1", now).Return(&types.ApiKey{ID: "api-key-1", LastRequestedAt: &now}, nil).Once()
			},
		},
		{
			name: "record_last_request_error",
			call: func(u *UseCases) error {
				_, err := u.RecordLastRequest(context.Background(), "api-key-1", now)
				return err
			},
			setup: func(f *useCasesFixture) {
				f.service.On("RecordLastRequest", mock.Anything, "api-key-1", now).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "verify_success",
			call: func(u *UseCases) error {
				result, err := u.Verify(context.Background(), verifyReq)
				if err != nil {
					return err
				}
				if result == nil || !result.Valid {
					return errors.New("expected valid result")
				}
				return nil
			},
			setup: func(f *useCasesFixture) {
				f.service.On("Verify", mock.Anything, verifyReq).Return(&types.VerifyApiKeyResult{Valid: true, ApiKey: &types.ApiKey{ID: "api-key-1"}}, nil).Once()
			},
		},
		{
			name: "verify_error",
			call: func(u *UseCases) error {
				_, err := u.Verify(context.Background(), verifyReq)
				return err
			},
			setup: func(f *useCasesFixture) {
				f.service.On("Verify", mock.Anything, verifyReq).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newUseCasesFixture()
			tc.setup(fixture)

			err := tc.call(fixture.useCases)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			fixture.assertExpectations(t)
		})
	}
}
