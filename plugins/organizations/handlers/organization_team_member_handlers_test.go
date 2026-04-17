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
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgtypes "github.com/Authula/authula/plugins/organizations/types"
)

type organizationTeamMemberHandlerFixture struct {
	service *orgtests.MockOrganizationTeamMemberService
}

type organizationTeamMemberHandlerCase struct {
	name            string
	userID          *string
	body            []byte
	organizationID  string
	teamID          string
	memberID        string
	prepare         func(*organizationTeamMemberHandlerFixture)
	expectedStatus  int
	expectedMessage string
	checkResponse   func(*testing.T, *models.RequestContext)
}

func newOrganizationTeamMemberHandlerFixture() *organizationTeamMemberHandlerFixture {
	return &organizationTeamMemberHandlerFixture{service: &orgtests.MockOrganizationTeamMemberService{}}
}

func (f *organizationTeamMemberHandlerFixture) newRequest(t *testing.T, method, path string, body []byte, userID *string, organizationID, teamID, memberID string) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()

	req, w, reqCtx := internaltests.NewHandlerRequest(t, method, path, body, userID)
	if organizationID != "" {
		req.SetPathValue("organization_id", organizationID)
	}
	if teamID != "" {
		req.SetPathValue("team_id", teamID)
	}
	if memberID != "" {
		req.SetPathValue("member_id", memberID)
	}
	return req, w, reqCtx
}

func runOrganizationTeamMemberHandlerCases(t *testing.T, method, path string, buildHandler func(*organizationTeamMemberHandlerFixture) http.HandlerFunc, cases []organizationTeamMemberHandlerCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationTeamMemberHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := buildHandler(fixture)
			req, w, reqCtx := fixture.newRequest(t, method, path, tt.body, tt.userID, tt.organizationID, tt.teamID, tt.memberID)
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

func TestAddOrganizationTeamMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamMemberHandlerCases(t, http.MethodPost, "/organizations/org-1/teams/team-1/members", func(fixture *organizationTeamMemberHandlerFixture) http.HandlerFunc {
		return (&AddOrganizationTeamMemberHandler{OrgTeamMemberService: fixture.service}).Handle()
	}, []organizationTeamMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			teamID:          "team-1",
			body:            internaltests.MarshalToJSON(t, orgtypes.AddOrganizationTeamMemberRequest{MemberID: "mem-2"}),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:            "invalid_json",
			userID:          new("user-1"),
			organizationID:  "org-1",
			teamID:          "team-1",
			body:            []byte("{"),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request body",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.AddOrganizationTeamMemberRequest{MemberID: "mem-2"}),
			prepare: func(fixture *organizationTeamMemberHandlerFixture) {
				fixture.service.On("AddTeamMember", mock.Anything, "user-1", "org-1", "team-1", mock.Anything).Return((*orgtypes.OrganizationTeamMember)(nil), errors.New("add failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "add failed",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.AddOrganizationTeamMemberRequest{MemberID: "mem-2"}),
			prepare: func(fixture *organizationTeamMemberHandlerFixture) {
				fixture.service.On("AddTeamMember", mock.Anything, "user-1", "org-1", "team-1", mock.Anything).Return(&orgtypes.OrganizationTeamMember{ID: "team-mem-1", TeamID: "team-1", MemberID: "mem-2"}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				teamMember := internaltests.DecodeResponseJSON[orgtypes.OrganizationTeamMember](t, reqCtx)
				assert.Equal(t, "team-mem-1", teamMember.ID)
				assert.Equal(t, "team-1", teamMember.TeamID)
				assert.Equal(t, "mem-2", teamMember.MemberID)
			},
		},
	})
}

func TestGetAllOrganizationTeamMembersHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamMemberHandlerCases(t, http.MethodGet, "/organizations/org-1/teams/team-1/members", func(fixture *organizationTeamMemberHandlerFixture) http.HandlerFunc {
		return (&GetAllOrganizationTeamMembersHandler{OrgTeamMemberService: fixture.service}).Handle()
	}, []organizationTeamMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			teamID:          "team-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			prepare: func(fixture *organizationTeamMemberHandlerFixture) {
				fixture.service.On("GetAllTeamMembers", mock.Anything, "user-1", "org-1", "team-1", 1, 10).Return(([]orgtypes.OrganizationTeamMember)(nil), errors.New("some error")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "some error",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			prepare: func(fixture *organizationTeamMemberHandlerFixture) {
				fixture.service.On("GetAllTeamMembers", mock.Anything, "user-1", "org-1", "team-1", 1, 10).Return([]orgtypes.OrganizationTeamMember{{ID: "team-mem-1", TeamID: "team-1", MemberID: "mem-2"}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				teamMembers := internaltests.DecodeResponseJSON[[]orgtypes.OrganizationTeamMember](t, reqCtx)
				assert.Len(t, teamMembers, 1)
				assert.Equal(t, "team-mem-1", teamMembers[0].ID)
			},
		},
	})
}

func TestGetOrganizationTeamMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamMemberHandlerCases(t, http.MethodGet, "/organizations/org-1/teams/team-1/members/mem-1", func(fixture *organizationTeamMemberHandlerFixture) http.HandlerFunc {
		return (&GetOrganizationTeamMemberHandler{OrgTeamMemberService: fixture.service}).Handle()
	}, []organizationTeamMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			teamID:          "team-1",
			memberID:        "mem-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "not_found",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "mem-1",
			prepare: func(fixture *organizationTeamMemberHandlerFixture) {
				fixture.service.On("GetTeamMember", mock.Anything, "user-1", "org-1", "team-1", "mem-1").Return((*orgtypes.OrganizationTeamMember)(nil), internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "mem-1",
			prepare: func(fixture *organizationTeamMemberHandlerFixture) {
				fixture.service.On("GetTeamMember", mock.Anything, "user-1", "org-1", "team-1", "mem-1").Return(&orgtypes.OrganizationTeamMember{ID: "team-mem-1", TeamID: "team-1", MemberID: "mem-2"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				teamMember := internaltests.DecodeResponseJSON[orgtypes.OrganizationTeamMember](t, reqCtx)
				assert.Equal(t, "team-mem-1", teamMember.ID)
			},
		},
	})
}

func TestDeleteOrganizationTeamMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamMemberHandlerCases(t, http.MethodDelete, "/organizations/org-1/teams/team-1/members/mem-1", func(fixture *organizationTeamMemberHandlerFixture) http.HandlerFunc {
		return (&DeleteOrganizationTeamMemberHandler{OrgTeamMemberService: fixture.service}).Handle()
	}, []organizationTeamMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			teamID:          "team-1",
			memberID:        "mem-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "mem-1",
			prepare: func(fixture *organizationTeamMemberHandlerFixture) {
				fixture.service.On("RemoveTeamMember", mock.Anything, "user-1", "org-1", "team-1", "mem-1").Return(errors.New("delete failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "delete failed",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "mem-1",
			prepare: func(fixture *organizationTeamMemberHandlerFixture) {
				fixture.service.On("RemoveTeamMember", mock.Anything, "user-1", "org-1", "team-1", "mem-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				response := internaltests.DecodeResponseJSON[orgtypes.DeleteOrganizationTeamMemberResponse](t, reqCtx)
				assert.Equal(t, "organization team member deleted", response.Message)
			},
		},
	})
}
