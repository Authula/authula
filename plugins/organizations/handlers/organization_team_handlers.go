package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
)

type CreateOrganizationTeamHandler struct {
	OrgTeamService orgservices.OrganizationTeamService
}

func (h *CreateOrganizationTeamHandler) Handle() http.HandlerFunc {
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

		var request types.CreateOrganizationTeamRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}
		request.Trim()

		team, err := h.OrgTeamService.CreateTeam(ctx, userID, organizationID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, team)
	}
}

type GetAllOrganizationTeamsHandler struct {
	OrgTeamService orgservices.OrganizationTeamService
}

func (h *GetAllOrganizationTeamsHandler) Handle() http.HandlerFunc {
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
		teams, err := h.OrgTeamService.GetAllTeams(ctx, userID, organizationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, teams)
	}
}

type GetOrganizationTeamHandler struct {
	OrgTeamService orgservices.OrganizationTeamService
}

func (h *GetOrganizationTeamHandler) Handle() http.HandlerFunc {
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
		team, err := h.OrgTeamService.GetTeam(ctx, userID, organizationID, teamID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, team)
	}
}

type UpdateOrganizationTeamHandler struct {
	OrgTeamService orgservices.OrganizationTeamService
}

func (h *UpdateOrganizationTeamHandler) Handle() http.HandlerFunc {
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

		var request types.UpdateOrganizationTeamRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}
		request.Trim()

		team, err := h.OrgTeamService.UpdateTeam(ctx, userID, organizationID, teamID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, team)
	}
}

type DeleteOrganizationTeamHandler struct {
	OrgTeamService orgservices.OrganizationTeamService
}

func (h *DeleteOrganizationTeamHandler) Handle() http.HandlerFunc {
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
		if err := h.OrgTeamService.DeleteTeam(ctx, userID, organizationID, teamID); err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.DeleteOrganizationTeamResponse{Message: "organization team deleted"})
	}
}
