package apikey

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
)

func TestAPI_DelegatesToService(t *testing.T) {
	t.Parallel()

	actor := &models.Actor{ID: "user-1", Type: models.ActorUser}
	now := time.Now().UTC()
	createReq := types.CreateApiKeyRequest{Name: "key", OwnerType: types.OwnerTypeUser, OwnerID: "user-1"}
	getAllReq := types.GetApiKeysRequest{Page: 1, Limit: 10}
	updateReq := types.UpdateApiKeyData{Name: new("updated")}
	verifyReq := types.VerifyApiKeyRequest{Key: "raw-key"}

	tests := []struct {
		name    string
		setup   func(*apiKeyTests.MockApiKeyService)
		call    func(*API) error
		wantErr error
	}{
		{
			name: "create_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Create", mock.Anything, actor, createReq).Return(&types.CreateApiKeyResponse{RawApiKey: "raw"}, nil).Once()
			},
			call: func(a *API) error {
				resp, err := a.Create(context.Background(), actor, createReq)
				if err == nil && (resp == nil || resp.RawApiKey != "raw") {
					return errors.New("unexpected response")
				}
				return err
			},
		},
		{
			name: "create_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Create", mock.Anything, actor, createReq).Return(nil, errors.New("boom")).Once()
			},
			call: func(a *API) error {
				_, err := a.Create(context.Background(), actor, createReq)
				return err
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "get_by_id_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetByID", mock.Anything, actor, "api-key-1").Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			},
			call: func(a *API) error {
				apiKey, err := a.GetByID(context.Background(), actor, "api-key-1")
				if err == nil && (apiKey == nil || apiKey.ID != "api-key-1") {
					return errors.New("unexpected api key")
				}
				return err
			},
		},
		{
			name: "get_by_id_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetByID", mock.Anything, actor, "api-key-1").Return(nil, errors.New("boom")).Once()
			},
			call: func(a *API) error {
				_, err := a.GetByID(context.Background(), actor, "api-key-1")
				return err
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "get_all_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetAll", mock.Anything, actor, getAllReq).Return(&types.GetAllApiKeysResponse{Total: 3}, nil).Once()
			},
			call: func(a *API) error {
				resp, err := a.GetAll(context.Background(), actor, getAllReq)
				if err == nil && (resp == nil || resp.Total != 3) {
					return errors.New("unexpected response")
				}
				return err
			},
		},
		{
			name: "get_all_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetAll", mock.Anything, actor, getAllReq).Return(nil, errors.New("boom")).Once()
			},
			call: func(a *API) error {
				_, err := a.GetAll(context.Background(), actor, getAllReq)
				return err
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "update_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Update", mock.Anything, actor, "api-key-1", updateReq).Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			},
			call: func(a *API) error {
				apiKey, err := a.Update(context.Background(), actor, "api-key-1", updateReq)
				if err == nil && (apiKey == nil || apiKey.ID != "api-key-1") {
					return errors.New("unexpected api key")
				}
				return err
			},
		},
		{
			name: "update_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Update", mock.Anything, actor, "api-key-1", updateReq).Return(nil, errors.New("boom")).Once()
			},
			call: func(a *API) error {
				_, err := a.Update(context.Background(), actor, "api-key-1", updateReq)
				return err
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "delete_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Delete", mock.Anything, actor, "api-key-1").Return(nil).Once()
			},
			call: func(a *API) error {
				return a.Delete(context.Background(), actor, "api-key-1")
			},
		},
		{
			name: "delete_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Delete", mock.Anything, actor, "api-key-1").Return(errors.New("boom")).Once()
			},
			call: func(a *API) error {
				return a.Delete(context.Background(), actor, "api-key-1")
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "delete_expired_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("DeleteExpired", mock.Anything).Return(nil).Once()
			},
			call: func(a *API) error {
				return a.DeleteExpired(context.Background())
			},
		},
		{
			name: "delete_expired_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("DeleteExpired", mock.Anything).Return(errors.New("boom")).Once()
			},
			call: func(a *API) error {
				return a.DeleteExpired(context.Background())
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "delete_all_by_owner_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("DeleteAllByOwner", mock.Anything, actor, types.OwnerTypeUser, "user-1").Return(nil).Once()
			},
			call: func(a *API) error {
				return a.DeleteAllByOwner(context.Background(), actor, types.OwnerTypeUser, "user-1")
			},
		},
		{
			name: "delete_all_by_owner_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("DeleteAllByOwner", mock.Anything, actor, types.OwnerTypeUser, "user-1").Return(errors.New("boom")).Once()
			},
			call: func(a *API) error {
				return a.DeleteAllByOwner(context.Background(), actor, types.OwnerTypeUser, "user-1")
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "record_last_request_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("RecordLastRequest", mock.Anything, "api-key-1", now).Return(&types.ApiKey{ID: "api-key-1", LastRequestedAt: &now}, nil).Once()
			},
			call: func(a *API) error {
				apiKey, err := a.RecordLastRequest(context.Background(), "api-key-1", now)
				if err == nil && (apiKey == nil || apiKey.LastRequestedAt == nil) {
					return errors.New("unexpected api key")
				}
				return err
			},
		},
		{
			name: "record_last_request_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("RecordLastRequest", mock.Anything, "api-key-1", now).Return(nil, errors.New("boom")).Once()
			},
			call: func(a *API) error {
				_, err := a.RecordLastRequest(context.Background(), "api-key-1", now)
				return err
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "verify_success",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Verify", mock.Anything, verifyReq).Return(&types.VerifyApiKeyResult{Valid: true}, nil).Once()
			},
			call: func(a *API) error {
				result, err := a.Verify(context.Background(), verifyReq)
				if err == nil && (result == nil || !result.Valid) {
					return errors.New("unexpected result")
				}
				return err
			},
		},
		{
			name: "verify_error",
			setup: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Verify", mock.Anything, verifyReq).Return(nil, errors.New("boom")).Once()
			},
			call: func(a *API) error {
				_, err := a.Verify(context.Background(), verifyReq)
				return err
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &apiKeyTests.MockApiKeyService{}
			tc.setup(service)
			api := NewAPI(service)

			err := tc.call(api)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			service.AssertExpectations(t)
		})
	}
}

func TestNewAPI(t *testing.T) {
	t.Parallel()

	service := &apiKeyTests.MockApiKeyService{}

	api := NewAPI(service)

	require.NotNil(t, api)
	assert.Same(t, service, api.service)
}
