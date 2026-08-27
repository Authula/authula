package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
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
	targetUserID    string
	prepare         func(*organizationMemberHandlerFixture)
	expectedStatus  int
	expectedCode    string
	expectedMessage string
	checkResponse   func(*testing.T, *models.RequestContext)
}

func newOrganizationMemberHandlerFixture() *organizationMemberHandlerFixture {
	return &organizationMemberHandlerFixture{service: &orgtests.MockOrganizationMemberService{}}
}

func (f *organizationMemberHandlerFixture) newRequest(t *testing.T, method, path string, body []byte, userID *string, organizationID, memberID string) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()

	var actor *models.Actor
	if userID != nil {
		actor = orgtests.Actor(*userID)
	}
	req, w, reqCtx := internaltests.NewHandlerRequestWithActor(t, method, path, body, actor)
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
			if tt.targetUserID != "" {
				req.SetPathValue("user_id", tt.targetUserID)
			}
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
		})
	}
}

func TestAddOrganizationMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodPost, "/organizations/org-1/members", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&AddOrganizationMemberHandler{UseCases: newMemberUseCases(fixture.service)}).Handle()
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
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "unexpected EOF",
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
			expectedStatus:  http.StatusConflict,
			expectedCode:    orgconstants.CodeMembersQuotaExceeded,
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
			},
		},
	})
}

func TestListAllOrganizationMembersHandler(t *testing.T) {
	t.Parallel()

	defaultParams := pagination.Params{Page: pagination.DefaultPage, Limit: pagination.DefaultLimit}

	runOrganizationMemberHandlerCases(t, http.MethodGet, "/organizations/org-1/members", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&ListAllOrganizationMembersHandler{UseCases: newMemberUseCases(fixture.service)}).Handle()
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
				fixture.service.On("ListAllMembers", mock.Anything, "user-1", "org-1", defaultParams).
					Return((*orgtypes.ListOrganizationMembersResponse)(nil), errors.New("some error")).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "some error",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("ListAllMembers", mock.Anything, "user-1", "org-1", defaultParams).
					Return(&orgtypes.ListOrganizationMembersResponse{
						Data:       []orgtypes.OrganizationMemberResponse{{ID: "mem-1", OrganizationID: "org-1", Role: "member"}},
						Pagination: pagination.New(1, 10, 25),
					}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				resp := internaltests.DecodeResponseJSON[orgtypes.ListOrganizationMembersResponse](t, reqCtx)
				assert.Len(t, resp.Data, 1)
				assert.Equal(t, "mem-1", resp.Data[0].ID)
				assert.Equal(t, pagination.Pagination{Page: 1, Limit: 10, Total: 25, TotalPages: 3, HasMore: true}, resp.Pagination)
			},
		},
	})
}

func TestListAllOrganizationMembersHandlerParsesPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		query          string
		expectedParams pagination.Params
	}{
		{name: "no query string uses the defaults", query: "", expectedParams: pagination.Params{Page: 1, Limit: 10}},
		{name: "explicit values are forwarded", query: "?page=3&limit=50", expectedParams: pagination.Params{Page: 3, Limit: 50}},
		{name: "unparseable values fall back to the defaults", query: "?page=abc&limit=xyz", expectedParams: pagination.Params{Page: 1, Limit: 10}},
		{name: "an absurd limit is forwarded for the service to clamp", query: "?limit=100000", expectedParams: pagination.Params{Page: 1, Limit: 100000}},
		{name: "a negative limit is forwarded for the service to clamp", query: "?limit=-1", expectedParams: pagination.Params{Page: 1, Limit: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runOrganizationMemberHandlerCases(t, http.MethodGet, "/organizations/org-1/members"+tt.query, func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
				return (&ListAllOrganizationMembersHandler{UseCases: newMemberUseCases(fixture.service)}).Handle()
			}, []organizationMemberHandlerCase{
				{
					name:           "forwards_parsed_params",
					userID:         new("user-1"),
					organizationID: "org-1",
					prepare: func(fixture *organizationMemberHandlerFixture) {
						fixture.service.On("ListAllMembers", mock.Anything, "user-1", "org-1", tt.expectedParams).
							Return(&orgtypes.ListOrganizationMembersResponse{
								Data:       []orgtypes.OrganizationMemberResponse{},
								Pagination: pagination.New(1, 10, 0),
							}, nil).Once()
					},
					expectedStatus: http.StatusOK,
				},
			})
		})
	}
}

func TestGetOrganizationMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodGet, "/organizations/org-1/members/mem-1", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&GetOrganizationMemberHandler{UseCases: newMemberUseCases(fixture.service)}).Handle()
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
				fixture.service.On("GetMember", mock.Anything, "user-1", "org-1", "mem-1").Return((*orgtypes.OrganizationMemberResponse)(nil), coreerrors.ErrNotFound).Once()
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
				fixture.service.On("GetMember", mock.Anything, "user-1", "org-1", "mem-1").Return(&orgtypes.OrganizationMemberResponse{ID: "mem-1", OrganizationID: "org-1", Role: "member"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				member := internaltests.DecodeResponseJSON[orgtypes.OrganizationMemberResponse](t, reqCtx)
				assert.Equal(t, "mem-1", member.ID)
			},
		},
	})
}

func TestGetOrganizationMemberByUserIDHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodGet, "/organizations/org-1/members/by-user/user-2", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&GetOrganizationMemberByUserIDHandler{UseCases: newMemberUseCases(fixture.service)}).Handle()
	}, []organizationMemberHandlerCase{
		{
			name:            "missing_user",
			organizationID:  "org-1",
			targetUserID:    "user-2",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Unauthorized",
		},
		{
			name:           "not_found",
			userID:         new("user-1"),
			organizationID: "org-1",
			targetUserID:   "user-2",
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("GetMemberByUserID", mock.Anything, "user-1", "org-1", "user-2").Return((*orgtypes.OrganizationMemberResponse)(nil), coreerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:           "success",
			userID:         new("user-1"),
			organizationID: "org-1",
			targetUserID:   "user-2",
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("GetMemberByUserID", mock.Anything, "user-1", "org-1", "user-2").Return(&orgtypes.OrganizationMemberResponse{ID: "mem-1", OrganizationID: "org-1", Role: "member"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				member := internaltests.DecodeResponseJSON[orgtypes.OrganizationMemberResponse](t, reqCtx)
				assert.Equal(t, "mem-1", member.ID)
			},
		},
	})
}

func TestUpdateOrganizationMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodPatch, "/organizations/org-1/members/mem-1", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&UpdateOrganizationMemberHandler{UseCases: newMemberUseCases(fixture.service)}).Handle()
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
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "unexpected EOF",
		},
		{
			name:           "forbidden",
			userID:         new("user-1"),
			organizationID: "org-1",
			memberID:       "mem-1",
			body:           internaltests.MarshalToJSON(t, orgtypes.UpdateOrganizationMemberRequest{Role: "admin"}),
			prepare: func(fixture *organizationMemberHandlerFixture) {
				fixture.service.On("UpdateMember", mock.Anything, "user-1", "org-1", "mem-1", mock.Anything).Return((*orgtypes.OrganizationMember)(nil), coreerrors.ErrForbidden).Once()
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
			},
		},
	})
}

func TestDeleteOrganizationMemberHandler(t *testing.T) {
	t.Parallel()

	runOrganizationMemberHandlerCases(t, http.MethodDelete, "/organizations/org-1/members/mem-1", func(fixture *organizationMemberHandlerFixture) http.HandlerFunc {
		return (&DeleteOrganizationMemberHandler{UseCases: newMemberUseCases(fixture.service)}).Handle()
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
