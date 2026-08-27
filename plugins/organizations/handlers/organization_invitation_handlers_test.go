package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgtypes "github.com/Authula/authula/plugins/organizations/types"
)

type organizationInvitationHandlerFixture struct {
	service *orgtests.MockOrganizationInvitationService
	orgSvc  *orgtests.MockOrganizationService
}

type organizationInvitationHandlerCase struct {
	name            string
	userID          *string
	body            []byte
	organizationID  string
	invitationID    string
	prepare         func(*organizationInvitationHandlerFixture)
	expectedStatus  int
	expectedCode    string
	expectedMessage string
	checkResponse   func(*testing.T, *models.RequestContext)
}

func newOrganizationInvitationHandlerFixture() *organizationInvitationHandlerFixture {
	return &organizationInvitationHandlerFixture{
		service: &orgtests.MockOrganizationInvitationService{},
		orgSvc:  &orgtests.MockOrganizationService{},
	}
}

func (f *organizationInvitationHandlerFixture) newRequest(t *testing.T, method, path string, body []byte, userID *string, organizationID, invitationID string) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()

	var actor *models.Actor
	if userID != nil {
		actor = orgtests.Actor(*userID)
	}
	req, w, reqCtx := internaltests.NewHandlerRequestWithActor(t, method, path, body, actor)
	if organizationID != "" {
		req.SetPathValue("organization_id", organizationID)
	}
	if invitationID != "" {
		req.SetPathValue("invitation_id", invitationID)
	}
	return req, w, reqCtx
}

func runOrganizationInvitationHandlerCases(t *testing.T, method, path string, buildHandler func(*organizationInvitationHandlerFixture) http.HandlerFunc, cases []organizationInvitationHandlerCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationInvitationHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := buildHandler(fixture)
			req, w, reqCtx := fixture.newRequest(t, method, path, tt.body, tt.userID, tt.organizationID, tt.invitationID)
			if tt.name == "missing_user" {
				reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
				reqCtx.Handled = true
			} else {
				handler.ServeHTTP(w, req)
			}

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.expectedMessage != "" {
				if tt.expectedCode != "" {
					internaltests.AssertErrorResponse(t, reqCtx, tt.expectedStatus, tt.expectedCode, tt.expectedMessage)
				} else {
					internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
				}
			}
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			fixture.service.AssertExpectations(t)
			fixture.orgSvc.AssertExpectations(t)
		})
	}
}

func TestCreateOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodPost, "/organizations/org-1/invitations", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&CreateOrganizationInvitationHandler{UseCases: newInvitationUseCases(fixture.orgSvc, fixture.service)}).Handle()
	}, []organizationInvitationHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			body:            internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"}),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:            "invalid_json",
			userID:          new("user-1"),
			organizationID:  "org-1",
			body:            []byte("{"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "unexpected EOF",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"}),
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("CreateOrganizationInvitation", mock.Anything, "user-1", "org-1", mock.Anything, mock.Anything).Return((*orgtypes.OrganizationInvitation)(nil), errors.New("create failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "create failed",
		},
		{
			name:           "quota exceeded",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"}),
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("CreateOrganizationInvitation", mock.Anything, "user-1", "org-1", mock.Anything, mock.Anything).Return((*orgtypes.OrganizationInvitation)(nil), orgconstants.ErrInvitationsQuotaExceeded).Once()
			},
			expectedStatus:  http.StatusConflict,
			expectedCode:    orgconstants.CodeInvitationsQuotaExceeded,
			expectedMessage: orgconstants.ErrInvitationsQuotaExceeded.Error(),
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"}),
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("CreateOrganizationInvitation", mock.Anything, "user-1", "org-1", mock.Anything, mock.Anything).Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusPending}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				invitation := internaltests.DecodeResponseJSON[orgtypes.OrganizationInvitation](t, reqCtx)
				assert.Equal(t, "inv-1", invitation.ID)
				assert.Equal(t, "org-1", invitation.OrganizationID)
				assert.Equal(t, "user@example.com", invitation.Email)
				assert.Equal(t, "member", invitation.Role)
			},
		},
	})
}

func TestListAllOrganizationInvitationsHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodGet, "/organizations/org-1/invitations", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&ListAllOrganizationInvitationsHandler{UseCases: newInvitationUseCases(fixture.orgSvc, fixture.service)}).Handle()
	}, []organizationInvitationHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("ListAllOrganizationInvitationsByOrgIDWithOrg", mock.Anything, "org-1", pagination.Params{Page: pagination.DefaultPage, Limit: pagination.DefaultLimit}).
					Return((*orgtypes.ListOrganizationInvitationsResponse)(nil), errors.New("some error")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "some error",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("ListAllOrganizationInvitationsByOrgIDWithOrg", mock.Anything, "org-1", pagination.Params{Page: pagination.DefaultPage, Limit: pagination.DefaultLimit}).
					Return(&orgtypes.ListOrganizationInvitationsResponse{
						Data: []orgtypes.GetOrganizationInvitationResponse{
							{
								Invitation:   &orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusPending},
								Organization: orgtypes.OrganizationSummary{ID: "org-1", Name: "Acme Corp", Slug: "acme"},
							},
						},
						Pagination: pagination.New(1, 10, 1),
					}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				resp := internaltests.DecodeResponseJSON[orgtypes.ListOrganizationInvitationsResponse](t, reqCtx)
				require.Len(t, resp.Data, 1)
				assert.Equal(t, "inv-1", resp.Data[0].Invitation.ID)
				assert.Equal(t, "org-1", resp.Data[0].Invitation.OrganizationID)
				assert.Equal(t, "Acme Corp", resp.Data[0].Organization.Name)
				assert.Equal(t, pagination.Pagination{Page: 1, Limit: 10, Total: 1, TotalPages: 1, HasMore: false}, resp.Pagination)
			},
		},
	})
}

func TestGetOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodGet, "/organizations/org-1/invitations/inv-1", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&GetOrganizationInvitationHandler{UseCases: newInvitationUseCases(fixture.orgSvc, fixture.service)}).Handle()
	}, []organizationInvitationHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			invitationID:    "inv-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "not_found",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return((*orgtypes.GetOrganizationInvitationResponse)(nil), coreerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(&orgtypes.GetOrganizationInvitationResponse{
					Invitation:   &orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusPending},
					Organization: orgtypes.OrganizationSummary{ID: "org-1", Name: "Acme Corp", Slug: "acme"},
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				resp := internaltests.DecodeResponseJSON[orgtypes.GetOrganizationInvitationResponse](t, reqCtx)
				require.NotNil(t, resp)
				assert.Equal(t, "inv-1", resp.Invitation.ID)
				assert.Equal(t, "org-1", resp.Invitation.OrganizationID)
				assert.Equal(t, "org-1", resp.Organization.ID)
				assert.Equal(t, "Acme Corp", resp.Organization.Name)
				assert.Equal(t, "acme", resp.Organization.Slug)
			},
		},
	})
}

func TestRevokeOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodPatch, "/organizations/org-1/invitations/inv-1", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&RevokeOrganizationInvitationHandler{UseCases: newInvitationUseCases(fixture.orgSvc, fixture.service)}).Handle()
	}, []organizationInvitationHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			invitationID:    "inv-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("RevokeOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return((*orgtypes.OrganizationInvitation)(nil), errors.New("revocation failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "revocation failed",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("RevokeOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusRevoked}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				invitation := internaltests.DecodeResponseJSON[orgtypes.OrganizationInvitation](t, reqCtx)
				assert.Equal(t, "inv-1", invitation.ID)
				assert.Equal(t, orgtypes.OrganizationInvitationStatusRevoked, invitation.Status)
			},
		},
	})
}

func TestAcceptOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodPost, "/organizations/org-1/invitations/inv-1/accept", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&AcceptOrganizationInvitationHandler{UseCases: newInvitationUseCases(fixture.orgSvc, fixture.service)}).Handle()
	}, []organizationInvitationHandlerCase{
		{
			name:           "redirect_url",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("AcceptOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusAccepted}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				assert.Equal(t, "", reqCtx.RedirectURL)
			},
		},
		{
			name:           "json_response",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("AcceptOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusAccepted}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				invitation := internaltests.DecodeResponseJSON[orgtypes.OrganizationInvitation](t, reqCtx)
				assert.Equal(t, "inv-1", invitation.ID)
				assert.Equal(t, orgtypes.OrganizationInvitationStatusAccepted, invitation.Status)
			},
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("AcceptOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return((*orgtypes.OrganizationInvitation)(nil), errors.New("accept failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "accept failed",
		},
		{
			name:           "success_with_role_assignment",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("AcceptOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusAccepted, InviterID: "user-2"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				invitation := internaltests.DecodeResponseJSON[orgtypes.OrganizationInvitation](t, reqCtx)
				assert.Equal(t, "inv-1", invitation.ID)
				assert.Equal(t, orgtypes.OrganizationInvitationStatusAccepted, invitation.Status)
			},
		},
	})
}

func TestRejectOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodPost, "/organizations/org-1/invitations/inv-1/reject", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&RejectOrganizationInvitationHandler{UseCases: newInvitationUseCases(fixture.orgSvc, fixture.service)}).Handle()
	}, []organizationInvitationHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			invitationID:    "inv-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("RejectOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return((*orgtypes.OrganizationInvitation)(nil), errors.New("reject failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "reject failed",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			invitationID:   "inv-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("RejectOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusRejected}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				invitation := internaltests.DecodeResponseJSON[orgtypes.OrganizationInvitation](t, reqCtx)
				assert.Equal(t, "inv-1", invitation.ID)
				assert.Equal(t, orgtypes.OrganizationInvitationStatusRejected, invitation.Status)
			},
		},
	})
}
