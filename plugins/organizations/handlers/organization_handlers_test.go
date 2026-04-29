package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgtypes "github.com/Authula/authula/plugins/organizations/types"
)

type organizationHandlerFixture struct {
	service *orgtests.MockOrganizationService
}

type organizationHandlerCase struct {
	name            string
	userID          *string
	body            []byte
	organizationID  string
	prepare         func(*organizationHandlerFixture)
	expectedStatus  int
	expectedMessage string
	checkResponse   func(t *testing.T, reqCtx *models.RequestContext)
}

func newOrganizationHandlerFixture() *organizationHandlerFixture {
	return &organizationHandlerFixture{service: &orgtests.MockOrganizationService{}}
}

func (f *organizationHandlerFixture) newRequest(t *testing.T, method, path string, body []byte, userID *string, organizationID string) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	req, w, reqCtx := internaltests.NewHandlerRequest(t, method, path, body, userID)
	if organizationID != "" {
		req.SetPathValue("organization_id", organizationID)
	}
	return req, w, reqCtx
}

func TestCreateOrganizationHandler(t *testing.T) {
	t.Parallel()

	tests := []organizationHandlerCase{
		{
			name:            "missing_user",
			body:            internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"}),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:            "missing_role",
			userID:          new("user-1"),
			body:            internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationRequest{Name: "Acme Inc"}),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "unprocessable entity",
		},
		{
			name:            "invalid_json",
			userID:          new("user-1"),
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request body",
		},
		{
			name:   "unprocessable_entity",
			userID: new("user-1"),
			body:   internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("CreateOrganization", mock.Anything, "user-1", mock.MatchedBy(func(request orgtypes.CreateOrganizationRequest) bool {
					return request.Name == "Acme Inc" && request.Role == "member"
				})).Return((*orgtypes.Organization)(nil), internalerrors.ErrUnprocessableEntity).Once()
			},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "unprocessable entity",
		},
		{
			name:   "quota_exceeded",
			userID: new("user-1"),
			body:   internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationRequest{Name: "Acme Inc", Role: "member"}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("CreateOrganization", mock.Anything, "user-1", mock.MatchedBy(func(request orgtypes.CreateOrganizationRequest) bool {
					return request.Name == "Acme Inc" && request.Role == "member"
				})).Return((*orgtypes.Organization)(nil), orgconstants.ErrOrganizationsQuotaExceeded).Once()
			},
			expectedStatus:  http.StatusTooManyRequests,
			expectedMessage: "organizations quota exceeded",
		},
		{
			name:   "repo_error",
			userID: new("user-1"),
			body: internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationRequest{
				Name: "Acme Inc",
				Role: "member",
			}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("CreateOrganization", mock.Anything, "user-1", mock.MatchedBy(func(request orgtypes.CreateOrganizationRequest) bool {
					return request.Name == "Acme Inc" && request.Role == "member"
				})).Return((*orgtypes.Organization)(nil), errors.New("create failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "create failed",
		},
		{
			name:   "success",
			userID: new("user-1"),
			body: internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationRequest{
				Name: "Acme Inc",
				Role: "member",
			}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("CreateOrganization", mock.Anything, "user-1", mock.MatchedBy(func(request orgtypes.CreateOrganizationRequest) bool {
					return request.Name == "Acme Inc" && request.Role == "member"
				})).Return(&orgtypes.Organization{
					ID:      "org-1",
					OwnerID: "user-1",
					Name:    "Acme Inc",
					Slug:    "acme-inc",
				}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				org := internaltests.DecodeResponseJSON[orgtypes.Organization](t, reqCtx)
				assert.Equal(t, "org-1", org.ID)
				assert.Equal(t, "user-1", org.OwnerID)
				assert.Equal(t, "Acme Inc", org.Name)
				assert.Equal(t, "acme-inc", org.Slug)
				assignRoleValue, ok := reqCtx.Values[models.ContextAccessControlAssignRole.String()]
				require.True(t, ok)
				assignRoleCtx, ok := assignRoleValue.(*models.AccessControlAssignRoleContext)
				require.True(t, ok)
				assert.Equal(t, "user-1", assignRoleCtx.UserID)
				assert.Equal(t, "member", assignRoleCtx.RoleName)
				assert.Nil(t, assignRoleCtx.AssignerUserID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := &CreateOrganizationHandler{OrgService: fixture.service}
			req, w, reqCtx := fixture.newRequest(t, http.MethodPost, "/organizations", tt.body, tt.userID, "")
			handler.Handle().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			fixture.service.AssertExpectations(t)
		})
	}
}

func TestGetAllOrganizationsHandler(t *testing.T) {
	t.Parallel()

	tests := []organizationHandlerCase{
		{
			name:            "missing_user",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:   "service_error",
			userID: new("user-1"),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("GetAllOrganizations", mock.Anything, "user-1").Return(([]orgtypes.Organization)(nil), errors.New("some error")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "some error",
		},
		{
			name:   "success",
			userID: new("user-1"),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("GetAllOrganizations", mock.Anything, "user-1").Return([]orgtypes.Organization{{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc"}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				organizations := internaltests.DecodeResponseJSON[[]orgtypes.Organization](t, reqCtx)
				require.Len(t, organizations, 1)
				assert.Equal(t, "org-1", organizations[0].ID)
				assert.Equal(t, "user-1", organizations[0].OwnerID)
				assert.Equal(t, "Acme Inc", organizations[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := &GetAllOrganizationsHandler{OrgService: fixture.service}
			req, w, reqCtx := fixture.newRequest(t, http.MethodGet, "/organizations", nil, tt.userID, "")
			handler.Handle().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			fixture.service.AssertExpectations(t)
		})
	}
}

func TestGetOrganizationByIDHandler(t *testing.T) {
	t.Parallel()

	tests := []organizationHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "not_found",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("GetOrganizationByID", mock.Anything, "user-1", "org-1").Return((*orgtypes.Organization)(nil), internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:           "forbidden",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("GetOrganizationByID", mock.Anything, "user-1", "org-1").Return((*orgtypes.Organization)(nil), internalerrors.ErrForbidden).Once()
			},
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name:           "member_success",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("GetOrganizationByID", mock.Anything, "user-1", "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "owner-2", Name: "Acme Inc", Slug: "acme-inc"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				org := internaltests.DecodeResponseJSON[orgtypes.Organization](t, reqCtx)
				assert.Equal(t, "org-1", org.ID)
				assert.Equal(t, "owner-2", org.OwnerID)
				assert.Equal(t, "Acme Inc", org.Name)
				assert.Equal(t, "acme-inc", org.Slug)
			},
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("GetOrganizationByID", mock.Anything, "user-1", "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				org := internaltests.DecodeResponseJSON[orgtypes.Organization](t, reqCtx)
				assert.Equal(t, "org-1", org.ID)
				assert.Equal(t, "user-1", org.OwnerID)
				assert.Equal(t, "Acme Inc", org.Name)
				assert.Equal(t, "acme-inc", org.Slug)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := &GetOrganizationByIDHandler{OrgService: fixture.service}
			req, w, reqCtx := fixture.newRequest(t, http.MethodGet, "/organizations/test-id", nil, tt.userID, tt.organizationID)
			handler.Handle().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			fixture.service.AssertExpectations(t)
		})
	}
}

func TestUpdateOrganizationHandler(t *testing.T) {
	t.Parallel()

	tests := []organizationHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			body:            internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationRequest{Name: "Acme Platform"}),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:            "invalid_json",
			userID:          new("user-1"),
			organizationID:  "org-1",
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request body",
		},
		{
			name:           "unprocessable_entity",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationRequest{Name: ""}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("UpdateOrganization", mock.Anything, "user-1", "org-1", mock.Anything).Return((*orgtypes.Organization)(nil), internalerrors.ErrUnprocessableEntity).Once()
			},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "unprocessable entity",
		},
		{
			name:           "not_found",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationRequest{Name: "Acme Platform"}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("UpdateOrganization", mock.Anything, "user-1", "org-1", mock.MatchedBy(func(request orgtypes.UpdateOrganizationRequest) bool {
					return request.Name == "Acme Platform"
				})).Return((*orgtypes.Organization)(nil), internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:           "forbidden",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationRequest{Name: "Acme Platform"}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("UpdateOrganization", mock.Anything, "user-1", "org-1", mock.Anything).Return((*orgtypes.Organization)(nil), internalerrors.ErrForbidden).Once()
			},
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name:           "member_success",
			userID:         new("user-1"),
			organizationID: "org-1",
			body: internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationRequest{
				Name:     "Acme Platform",
				Logo:     new("http://some/url/logo.svg"),
				Metadata: json.RawMessage(`{"tier":"pro"}`),
			}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("UpdateOrganization", mock.Anything, "user-1", "org-1", mock.MatchedBy(func(request orgtypes.UpdateOrganizationRequest) bool {
					return request.Name == "Acme Platform"
				})).Return(&orgtypes.Organization{
					ID:       "org-1",
					OwnerID:  "owner-2",
					Name:     "Acme Platform",
					Slug:     "acme-inc",
					Logo:     new("http://some/url/logo.svg"),
					Metadata: json.RawMessage(`{"tier":"pro"}`),
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				org := internaltests.DecodeResponseJSON[orgtypes.Organization](t, reqCtx)
				assert.Equal(t, "org-1", org.ID)
				assert.Equal(t, "owner-2", org.OwnerID)
				assert.Equal(t, "Acme Platform", org.Name)
				assert.Equal(t, "acme-inc", org.Slug)
				require.NotNil(t, org.Logo)
				assert.Equal(t, "http://some/url/logo.svg", *org.Logo)
				assert.NotEmpty(t, string(org.Metadata))
			},
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			body: internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationRequest{
				Name:     "Acme Platform",
				Logo:     new("http://some/url/logo.svg"),
				Metadata: json.RawMessage(`{"tier":"pro"}`),
			}),
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("UpdateOrganization", mock.Anything, "user-1", "org-1", mock.MatchedBy(func(request orgtypes.UpdateOrganizationRequest) bool {
					return request.Name == "Acme Platform"
				})).Return(&orgtypes.Organization{
					ID:       "org-1",
					OwnerID:  "user-1",
					Name:     "Acme Platform",
					Slug:     "acme-inc",
					Logo:     new("http://some/url/logo.svg"),
					Metadata: json.RawMessage(`{"tier":"pro"}`),
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				org := internaltests.DecodeResponseJSON[orgtypes.Organization](t, reqCtx)
				assert.Equal(t, "org-1", org.ID)
				assert.Equal(t, "user-1", org.OwnerID)
				assert.Equal(t, "Acme Platform", org.Name)
				assert.Equal(t, "acme-inc", org.Slug)
				require.NotNil(t, org.Logo)
				assert.Equal(t, "http://some/url/logo.svg", *org.Logo)
				assert.JSONEq(t, "{\"tier\":\"pro\"}", string(org.Metadata))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := &UpdateOrganizationHandler{OrgService: fixture.service}
			req, w, reqCtx := fixture.newRequest(t, http.MethodPatch, "/organizations/test-id", tt.body, tt.userID, tt.organizationID)
			handler.Handle().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			fixture.service.AssertExpectations(t)
		})
	}
}

func TestDeleteOrganizationHandler(t *testing.T) {
	t.Parallel()

	tests := []organizationHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "not_found",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("DeleteOrganization", mock.Anything, "user-1", "org-1").Return(internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:           "forbidden",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("DeleteOrganization", mock.Anything, "user-1", "org-1").Return(internalerrors.ErrForbidden).Once()
			},
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(f *organizationHandlerFixture) {
				f.service.On("DeleteOrganization", mock.Anything, "user-1", "org-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				response := internaltests.DecodeResponseJSON[orgtypes.DeleteOrganizationResponse](t, reqCtx)
				assert.Equal(t, "organization deleted", response.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := &DeleteOrganizationHandler{OrgService: fixture.service}
			req, w, reqCtx := fixture.newRequest(t, http.MethodDelete, "/organizations/test-id", nil, tt.userID, tt.organizationID)
			handler.Handle().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			fixture.service.AssertExpectations(t)
		})
	}
}
