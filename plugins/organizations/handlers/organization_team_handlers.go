package handlers

import (
	"net/http"

	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	"github.com/Authula/authula/plugins/organizations/types"
	orgusecases "github.com/Authula/authula/plugins/organizations/usecases"
	"github.com/Authula/authula/util"
)

type CreateOrganizationTeamHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *CreateOrganizationTeamHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")

		var request types.CreateOrganizationTeamRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		team, err := h.UseCases.CreateTeam(ctx, actor, organizationID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, team)
	}
}

type ListAllOrganizationTeamsHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *ListAllOrganizationTeamsHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		paginationParams := pagination.ParseFromRequest(r)
		teams, err := h.UseCases.ListAllTeams(ctx, actor, organizationID, paginationParams)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, teams)
	}
}

type GetOrganizationTeamHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *GetOrganizationTeamHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")
		team, err := h.UseCases.GetTeam(ctx, actor, organizationID, teamID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, team)
	}
}

type UpdateOrganizationTeamHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *UpdateOrganizationTeamHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")

		var request types.UpdateOrganizationTeamRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		team, err := h.UseCases.UpdateTeam(ctx, actor, organizationID, teamID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, team)
	}
}

type DeleteOrganizationTeamHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *DeleteOrganizationTeamHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		teamID := r.PathValue("team_id")
		if err := h.UseCases.DeleteTeam(ctx, actor, organizationID, teamID); err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.DeleteOrganizationTeamResponse{Message: "organization team deleted"})
	}
}
