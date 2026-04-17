package handlers

import (
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

type organizationInvitationHandlerFixture struct {
	service *orgtests.MockOrganizationInvitationService
}

type organizationInvitationHandlerCase struct {
	name            string
	userID          *string
	body            []byte
	organizationID  string
	invitationID    string
	prepare         func(*organizationInvitationHandlerFixture)
	expectedStatus  int
	expectedMessage string
	checkResponse   func(*testing.T, *models.RequestContext)
}

func newOrganizationInvitationHandlerFixture() *organizationInvitationHandlerFixture {
	return &organizationInvitationHandlerFixture{service: &orgtests.MockOrganizationInvitationService{}}
}

func (f *organizationInvitationHandlerFixture) newRequest(t *testing.T, method, path string, body []byte, userID *string, organizationID, invitationID string) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()

	req, w, reqCtx := internaltests.NewHandlerRequest(t, method, path, body, userID)
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
			handler.ServeHTTP(w, req)

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

func TestCreateOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodPost, "/organizations/org-1/invitations", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&CreateOrganizationInvitationHandler{OrgInvitationService: fixture.service}).Handle()
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
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request body",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"}),
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("CreateOrganizationInvitation", mock.Anything, "user-1", "org-1", mock.Anything).Return((*orgtypes.OrganizationInvitation)(nil), errors.New("create failed")).Once()
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
				fixture.service.On("CreateOrganizationInvitation", mock.Anything, "user-1", "org-1", mock.Anything).Return((*orgtypes.OrganizationInvitation)(nil), orgconstants.ErrInvitationsQuotaExceeded).Once()
			},
			expectedStatus:  http.StatusTooManyRequests,
			expectedMessage: orgconstants.ErrInvitationsQuotaExceeded.Error(),
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationInvitationRequest{Email: "user@example.com", Role: "member"}),
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("CreateOrganizationInvitation", mock.Anything, "user-1", "org-1", mock.Anything).Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusPending}, nil).Once()
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

func TestGetAllOrganizationInvitationsHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodGet, "/organizations/org-1/invitations", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&GetAllOrganizationInvitationsHandler{OrgInvitationService: fixture.service}).Handle()
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
				fixture.service.On("GetAllOrganizationInvitations", mock.Anything, "user-1", "org-1").Return(([]orgtypes.OrganizationInvitation)(nil), errors.New("some error")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "some error",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(fixture *organizationInvitationHandlerFixture) {
				fixture.service.On("GetAllOrganizationInvitations", mock.Anything, "user-1", "org-1").Return([]orgtypes.OrganizationInvitation{{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusPending}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				invitations := internaltests.DecodeResponseJSON[[]orgtypes.OrganizationInvitation](t, reqCtx)
				require.Len(t, invitations, 1)
				assert.Equal(t, "inv-1", invitations[0].ID)
				assert.Equal(t, "org-1", invitations[0].OrganizationID)
			},
		},
	})
}

func TestGetOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodGet, "/organizations/org-1/invitations/inv-1", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&GetOrganizationInvitationHandler{OrgInvitationService: fixture.service}).Handle()
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
				fixture.service.On("GetOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return((*orgtypes.OrganizationInvitation)(nil), internalerrors.ErrNotFound).Once()
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
				fixture.service.On("GetOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(&orgtypes.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1", Email: "user@example.com", Role: "member", Status: orgtypes.OrganizationInvitationStatusPending}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				invitation := internaltests.DecodeResponseJSON[orgtypes.OrganizationInvitation](t, reqCtx)
				assert.Equal(t, "inv-1", invitation.ID)
				assert.Equal(t, "org-1", invitation.OrganizationID)
			},
		},
	})
}

func TestRevokeOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodPatch, "/organizations/org-1/invitations/inv-1", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&RevokeOrganizationInvitationHandler{OrgInvitationService: fixture.service}).Handle()
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
		return (&AcceptOrganizationInvitationHandler{OrgInvitationService: fixture.service}).Handle()
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
				assignRoleValue, ok := reqCtx.Values[models.ContextAccessControlAssignRole.String()]
				require.True(t, ok)
				assignRoleCtx, ok := assignRoleValue.(*models.AccessControlAssignRoleContext)
				require.True(t, ok)
				assert.Equal(t, "user-1", assignRoleCtx.UserID)
				assert.Equal(t, "member", assignRoleCtx.RoleName)
				assert.Equal(t, "user-2", *assignRoleCtx.AssignerUserID)
			},
		},
	})
}

func TestRejectOrganizationInvitationHandler(t *testing.T) {
	t.Parallel()

	runOrganizationInvitationHandlerCases(t, http.MethodPost, "/organizations/org-1/invitations/inv-1/reject", func(fixture *organizationInvitationHandlerFixture) http.HandlerFunc {
		return (&RejectOrganizationInvitationHandler{OrgInvitationService: fixture.service}).Handle()
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
