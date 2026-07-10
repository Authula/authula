package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	"github.com/Authula/authula/plugins/organizations/types"
	orgusecases "github.com/Authula/authula/plugins/organizations/usecases"
	"github.com/Authula/authula/util"
)

type AddOrganizationTeamMemberHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *AddOrganizationTeamMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

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

		teamMember, err := h.UseCases.AddTeamMember(ctx, actor, organizationID, teamID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, teamMember)
	}
}

type GetAllOrganizationTeamMembersHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *GetAllOrganizationTeamMembersHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")
		page := util.GetQueryInt(r, "page", 1)
		limit := util.GetQueryInt(r, "limit", 10)
		teamMembers, err := h.UseCases.GetAllTeamMembers(ctx, actor, organizationID, teamID, page, limit)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, teamMembers)
	}
}

type GetOrganizationTeamMemberHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *GetOrganizationTeamMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")
		memberID := r.PathValue("member_id")
		teamMember, err := h.UseCases.GetTeamMember(ctx, actor, organizationID, teamID, memberID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, teamMember)
	}
}

type DeleteOrganizationTeamMemberHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *DeleteOrganizationTeamMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")
		memberID := r.PathValue("member_id")
		if err := h.UseCases.RemoveTeamMember(ctx, actor, organizationID, teamID, memberID); err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.DeleteOrganizationTeamMemberResponse{Message: "organization team member deleted"})
	}
}
