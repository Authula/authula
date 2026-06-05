package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
)

type CreateOrganizationHandler struct {
	OrgService orgservices.OrganizationService
}

func (h *CreateOrganizationHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		var request types.CreateOrganizationRequest
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

		organization, err := h.OrgService.CreateOrganization(ctx, actor, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.Values[models.ContextAccessControlAssignRole.String()] = &models.AccessControlAssignRoleContext{
			UserID:         actor.ID,
			RoleName:       request.Role,
			AssignerUserID: nil,
		}

		reqCtx.SetJSONResponse(http.StatusCreated, organization)
	}
}

type GetAllOrganizationsHandler struct {
	OrgService orgservices.OrganizationService
}

func (h *GetAllOrganizationsHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor
		organizations, err := h.OrgService.GetAllOrganizations(ctx, actor)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, organizations)
	}
}

type GetOrganizationByIDHandler struct {
	OrgService orgservices.OrganizationService
}

func (h *GetOrganizationByIDHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		organization, err := h.OrgService.GetOrganizationByID(ctx, actor, organizationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, organization)
	}
}

type UpdateOrganizationHandler struct {
	OrgService orgservices.OrganizationService
}

func (h *UpdateOrganizationHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")

		var request types.UpdateOrganizationRequest
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

		organization, err := h.OrgService.UpdateOrganization(ctx, actor, organizationID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, organization)
	}
}

type DeleteOrganizationHandler struct {
	OrgService orgservices.OrganizationService
}

func (h *DeleteOrganizationHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		if err := h.OrgService.DeleteOrganization(ctx, actor, organizationID); err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.DeleteOrganizationResponse{Message: "organization deleted"})
	}
}
