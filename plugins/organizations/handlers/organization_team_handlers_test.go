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

type organizationTeamHandlerFixture struct {
	service *orgtests.MockOrganizationTeamService
}

type organizationTeamHandlerCase struct {
	name            string
	userID          *string
	body            []byte
	organizationID  string
	teamID          string
	prepare         func(*organizationTeamHandlerFixture)
	expectedStatus  int
	expectedMessage string
	checkResponse   func(*testing.T, *models.RequestContext)
}

func newOrganizationTeamHandlerFixture() *organizationTeamHandlerFixture {
	return &organizationTeamHandlerFixture{service: &orgtests.MockOrganizationTeamService{}}
}

func (f *organizationTeamHandlerFixture) newRequest(t *testing.T, method, path string, body []byte, userID *string, organizationID, teamID string) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()

	req, w, reqCtx := internaltests.NewHandlerRequest(t, method, path, body, userID)
	if organizationID != "" {
		req.SetPathValue("organization_id", organizationID)
	}
	if teamID != "" {
		req.SetPathValue("team_id", teamID)
	}
	return req, w, reqCtx
}

func runOrganizationTeamHandlerCases(t *testing.T, method, path string, buildHandler func(*organizationTeamHandlerFixture) http.HandlerFunc, cases []organizationTeamHandlerCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newOrganizationTeamHandlerFixture()
			if tt.prepare != nil {
				tt.prepare(fixture)
			}

			handler := buildHandler(fixture)
			req, w, reqCtx := fixture.newRequest(t, method, path, tt.body, tt.userID, tt.organizationID, tt.teamID)
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

func TestCreateOrganizationTeamHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamHandlerCases(t, http.MethodPost, "/organizations/org-1/teams", func(fixture *organizationTeamHandlerFixture) http.HandlerFunc {
		return (&CreateOrganizationTeamHandler{OrgTeamService: fixture.service}).Handle()
	}, []organizationTeamHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			body:            internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationTeamRequest{Name: "Platform"}),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:            "invalid_json",
			userID:          new("user-1"),
			organizationID:  "org-1",
			body:            []byte("{"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid request body",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationTeamRequest{Name: "Platform"}),
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("CreateTeam", mock.Anything, "user-1", "org-1", mock.Anything).Return((*orgtypes.OrganizationTeam)(nil), errors.New("create failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "create failed",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.CreateOrganizationTeamRequest{Name: "Platform"}),
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("CreateTeam", mock.Anything, "user-1", "org-1", mock.Anything).Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				team := internaltests.DecodeResponseJSON[orgtypes.OrganizationTeam](t, reqCtx)
				assert.Equal(t, "team-1", team.ID)
				assert.Equal(t, "org-1", team.OrganizationID)
				assert.Equal(t, "Platform", team.Name)
			},
		},
	})
}

func TestGetAllOrganizationTeamsHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamHandlerCases(t, http.MethodGet, "/organizations/org-1/teams", func(fixture *organizationTeamHandlerFixture) http.HandlerFunc {
		return (&GetAllOrganizationTeamsHandler{OrgTeamService: fixture.service}).Handle()
	}, []organizationTeamHandlerCase{
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
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("GetAllTeams", mock.Anything, "user-1", "org-1").Return(([]orgtypes.OrganizationTeam)(nil), errors.New("some error")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "some error",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("GetAllTeams", mock.Anything, "user-1", "org-1").Return([]orgtypes.OrganizationTeam{{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				teams := internaltests.DecodeResponseJSON[[]orgtypes.OrganizationTeam](t, reqCtx)
				assert.Len(t, teams, 1)
				assert.Equal(t, "team-1", teams[0].ID)
			},
		},
	})
}

func TestGetOrganizationTeamHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamHandlerCases(t, http.MethodGet, "/organizations/org-1/teams/team-1", func(fixture *organizationTeamHandlerFixture) http.HandlerFunc {
		return (&GetOrganizationTeamHandler{OrgTeamService: fixture.service}).Handle()
	}, []organizationTeamHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			teamID:          "team-1",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "not_found",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("GetTeam", mock.Anything, "user-1", "org-1", "team-1").Return((*orgtypes.OrganizationTeam)(nil), internalerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("GetTeam", mock.Anything, "user-1", "org-1", "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				team := internaltests.DecodeResponseJSON[orgtypes.OrganizationTeam](t, reqCtx)
				assert.Equal(t, "team-1", team.ID)
			},
		},
	})
}

func TestUpdateOrganizationTeamHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamHandlerCases(t, http.MethodPatch, "/organizations/org-1/teams/team-1", func(fixture *organizationTeamHandlerFixture) http.HandlerFunc {
		return (&UpdateOrganizationTeamHandler{OrgTeamService: fixture.service}).Handle()
	}, []organizationTeamHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			teamID:          "team-1",
			body:            internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationTeamRequest{Name: "Platform"}),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:            "invalid_json",
			userID:          new("user-1"),
			organizationID:  "org-1",
			teamID:          "team-1",
			body:            []byte("{"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid request body",
		},
		{
			name:           "service_error",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationTeamRequest{Name: "Platform"}),
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("UpdateTeam", mock.Anything, "user-1", "org-1", "team-1", mock.Anything).Return((*orgtypes.OrganizationTeam)(nil), errors.New("update failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "update failed",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationTeamRequest{Name: "Platform"}),
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("UpdateTeam", mock.Anything, "user-1", "org-1", "team-1", mock.Anything).Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				team := internaltests.DecodeResponseJSON[orgtypes.OrganizationTeam](t, reqCtx)
				assert.Equal(t, "team-1", team.ID)
			},
		},
	})
}

func TestDeleteOrganizationTeamHandler(t *testing.T) {
	t.Parallel()

	runOrganizationTeamHandlerCases(t, http.MethodDelete, "/organizations/org-1/teams/team-1", func(fixture *organizationTeamHandlerFixture) http.HandlerFunc {
		return (&DeleteOrganizationTeamHandler{OrgTeamService: fixture.service}).Handle()
	}, []organizationTeamHandlerCase{
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
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("DeleteTeam", mock.Anything, "user-1", "org-1", "team-1").Return(errors.New("delete failed")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "delete failed",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			teamID:         "team-1",
			prepare: func(fixture *organizationTeamHandlerFixture) {
				fixture.service.On("DeleteTeam", mock.Anything, "user-1", "org-1", "team-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				response := internaltests.DecodeResponseJSON[orgtypes.DeleteOrganizationTeamResponse](t, reqCtx)
				assert.Equal(t, "organization team deleted", response.Message)
			},
		},
	})
}
