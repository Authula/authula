package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
)

type AddOrganizationTeamMemberHandler struct {
	OrgTeamMemberService orgservices.OrganizationTeamMemberService
}

func (h *AddOrganizationTeamMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		userID, ok := models.GetUserIDFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
			reqCtx.Handled = true
			return
		}

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")

		var request types.AddOrganizationTeamMemberRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		teamMember, err := h.OrgTeamMemberService.AddTeamMember(ctx, userID, organizationID, teamID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, teamMember)
	}
}

type GetAllOrganizationTeamMembersHandler struct {
	OrgTeamMemberService orgservices.OrganizationTeamMemberService
}

func (h *GetAllOrganizationTeamMembersHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		userID, ok := models.GetUserIDFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
			reqCtx.Handled = true
			return
		}

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")
		page := util.GetQueryInt(r, "page", 1)
		limit := util.GetQueryInt(r, "limit", 10)
		teamMembers, err := h.OrgTeamMemberService.GetAllTeamMembers(ctx, userID, organizationID, teamID, page, limit)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, teamMembers)
	}
}

type GetOrganizationTeamMemberHandler struct {
	OrgTeamMemberService orgservices.OrganizationTeamMemberService
}

func (h *GetOrganizationTeamMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		userID, ok := models.GetUserIDFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
			reqCtx.Handled = true
			return
		}

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")
		memberID := r.PathValue("member_id")
		teamMember, err := h.OrgTeamMemberService.GetTeamMember(ctx, userID, organizationID, teamID, memberID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, teamMember)
	}
}

type DeleteOrganizationTeamMemberHandler struct {
	OrgTeamMemberService orgservices.OrganizationTeamMemberService
}

func (h *DeleteOrganizationTeamMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		userID, ok := models.GetUserIDFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
			reqCtx.Handled = true
			return
		}

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")
		memberID := r.PathValue("member_id")
		if err := h.OrgTeamMemberService.RemoveTeamMember(ctx, userID, organizationID, teamID, memberID); err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.DeleteOrganizationTeamMemberResponse{Message: "organization team member deleted"})
	}
}
