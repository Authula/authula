package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgtypes "github.com/Authula/authula/plugins/organizations/types"
)

type organizationMemberHandlerFixture struct {
	service *orgtests.MockOrganizationMemberService
}

type organizationMemberHandlerCase struct {
	name            string
	userID          *string
	body            []byte
	organizationID  string
	memberID        string
	prepare         func(*organizationMemberHandlerFixture)
	expectedStatus  int
	expectedMessage string
	checkResponse   func(*testing.T, *models.RequestContext)
}

func newOrganizationMemberHandlerFixture() *organizationMemberHandlerFixture {
	return &organizationMemberHandlerFixture{service: &orgtests.MockOrganizationMemberService{}}
}

func (f *organizationMemberHandlerFixture) newRequest(t *testing.T, method, path string, body []byte, userID *string, organizationID, memberID string) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()

	req, w, reqCtx := internaltests.NewHandlerRequest(t, method, path, body, userID)
	if organizationID != "" {
		req.SetPathValue("organization_id", organizationID)
	}
	if memberID != "" {
		req.SetPathValue("member_id", memberID)
	}
	return req, w, reqCtx
}

func runOrganizationMemberHandlerCases(t *testing.T, method, path string, buildHandler func(*organizationMemberHandlerFixture) http.HandlerFunc, cases []organizationMemberHandlerCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationMemberHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := buildHandler(fixture)
			req, w, reqCtx := fixture.newRequest(t, method, path, tt.body, tt.userID, tt.organizationID, tt.memberID)
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

func TestAddOrganizationMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodPost, "/organizations/org-1/members", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&AddOrganizationMemberHandler{OrgMemberService: fixture.service}).Handle()
	}, []organizationMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			body:            internaltests.MarshalToJSON(t, orgtypes.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"}),
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
			body:           internaltests.MarshalToJSON(t, orgtypes.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"}),
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("AddMember", mock.Anything, "user-1", "org-1", mock.Anything).Return((*orgtypes.OrganizationMember)(nil), errors.New("add failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "add failed",
		},
		{
			name:           "quota_exceeded",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"}),
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("AddMember", mock.Anything, "user-1", "org-1", mock.Anything).Return((*orgtypes.OrganizationMember)(nil), orgconstants.ErrMembersQuotaExceeded).Once()
			},
			expectedStatus:  http.StatusTooManyRequests,
			expectedMessage: "members quota exceeded",
		},
		{
			name:           "success_with_role_assignment",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"}),
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("AddMember", mock.Anything, "user-1", "org-1", mock.Anything).Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				member := internaltests.DecodeResponseJSON[orgtypes.OrganizationMember](t, reqCtx)
				assert.Equal(t, "mem-1", member.ID)
				assert.Equal(t, "org-1", member.OrganizationID)
				assert.Equal(t, "user-2", member.UserID)
				assignRoleValue, ok := reqCtx.Values[models.ContextAccessControlAssignRole.String()]
				assert.True(t, ok)
				assignRoleCtx, ok := assignRoleValue.(*models.AccessControlAssignRoleContext)
				assert.True(t, ok)
				assert.Equal(t, "user-2", assignRoleCtx.UserID)
				assert.Equal(t, "member", assignRoleCtx.RoleName)
				assert.Equal(t, "user-1", *assignRoleCtx.AssignerUserID)
			},
		},
	})
}

func TestGetAllOrganizationMembersHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodGet, "/organizations/org-1/members", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&GetAllOrganizationMembersHandler{OrgMemberService: fixture.service}).Handle()
	}, []organizationMemberHandlerCase{
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
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("GetAllMembers", mock.Anything, "user-1", "org-1", 1, 10).Return(([]orgtypes.OrganizationMember)(nil), errors.New("some error")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "some error",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("GetAllMembers", mock.Anything, "user-1", "org-1", 1, 10).Return([]orgtypes.OrganizationMember{{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				members := internaltests.DecodeResponseJSON[[]orgtypes.OrganizationMember](t, reqCtx)
				assert.Len(t, members, 1)
				assert.Equal(t, "mem-1", members[0].ID)
			},
		},
	})
}

func TestGetOrganizationMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodGet, "/organizations/org-1/members/mem-1", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&GetOrganizationMemberHandler{OrgMemberService: fixture.service}).Handle()
	}, []organizationMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			memberID:        "mem-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "not_found",
			userID:         new("user-1"),
			organizationID: "org-1",
			memberID:       "mem-1",
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("GetMember", mock.Anything, "user-1", "org-1", "mem-1").Return((*orgtypes.OrganizationMember)(nil), internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			memberID:       "mem-1",
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("GetMember", mock.Anything, "user-1", "org-1", "mem-1").Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				member := internaltests.DecodeResponseJSON[orgtypes.OrganizationMember](t, reqCtx)
				assert.Equal(t, "mem-1", member.ID)
				assert.Equal(t, "user-2", member.UserID)
			},
		},
	})
}

func TestUpdateOrganizationMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodPatch, "/organizations/org-1/members/mem-1", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&UpdateOrganizationMemberHandler{OrgMemberService: fixture.service}).Handle()
	}, []organizationMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			memberID:        "mem-1",
			body:            internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationMemberRequest{Role: "admin"}),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:            "invalid_json",
			userID:          new("user-1"),
			organizationID:  "org-1",
			memberID:        "mem-1",
			body:            []byte("{"),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request body",
		},
		{
			name:           "forbidden",
			userID:         new("user-1"),
			organizationID: "org-1",
			memberID:       "mem-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationMemberRequest{Role: "admin"}),
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("UpdateMember", mock.Anything, "user-1", "org-1", "mem-1", mock.Anything).Return((*orgtypes.OrganizationMember)(nil), internalerrors.ErrForbidden).Once()
			},
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name:           "success_with_role_assignment",
			userID:         new("user-1"),
			organizationID: "org-1",
			memberID:       "mem-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationMemberRequest{Role: "admin"}),
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("UpdateMember", mock.Anything, "user-1", "org-1", "mem-1", mock.Anything).Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "admin"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				member := internaltests.DecodeResponseJSON[orgtypes.OrganizationMember](t, reqCtx)
				assert.Equal(t, "admin", member.Role)
				assignRoleValue, ok := reqCtx.Values[models.ContextAccessControlAssignRole.String()]
				assert.True(t, ok)
				assignRoleCtx, ok := assignRoleValue.(*models.AccessControlAssignRoleContext)
				assert.True(t, ok)
				assert.Equal(t, "user-2", assignRoleCtx.UserID)
				assert.Equal(t, "admin", assignRoleCtx.RoleName)
				assert.Equal(t, "user-1", *assignRoleCtx.AssignerUserID)
			},
		},
	})
}

func TestDeleteOrganizationMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodDelete, "/organizations/org-1/members/mem-1", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&DeleteOrganizationMemberHandler{OrgMemberService: fixture.service}).Handle()
	}, []organizationMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			memberID:        "mem-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			memberID:       "mem-1",
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("RemoveMember", mock.Anything, "user-1", "org-1", "mem-1").Return(errors.New("delete failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "delete failed",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			memberID:       "mem-1",
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("RemoveMember", mock.Anything, "user-1", "org-1", "mem-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				response := internaltests.DecodeResponseJSON[orgtypes.DeleteOrganizationMemberResponse](t, reqCtx)
				assert.Equal(t, "organization member deleted", response.Message)
			},
		},
	})
}
