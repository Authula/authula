package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
)

type CreateOrganizationInvitationHandler struct {
	OrgInvitationService orgservices.OrganizationInvitationService
}

func (h *CreateOrganizationInvitationHandler) Handle() http.HandlerFunc {
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

		var request types.CreateOrganizationInvitationRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}
		request.Trim()

		invitation, err := h.OrgInvitationService.CreateOrganizationInvitation(ctx, userID, organizationID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, invitation)
	}
}

type GetAllOrganizationInvitationsHandler struct {
	OrgInvitationService orgservices.OrganizationInvitationService
}

func (h *GetAllOrganizationInvitationsHandler) Handle() http.HandlerFunc {
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
		invitations, err := h.OrgInvitationService.GetAllOrganizationInvitations(ctx, userID, organizationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitations)
	}
}

type GetOrganizationInvitationHandler struct {
	OrgInvitationService orgservices.OrganizationInvitationService
}

func (h *GetOrganizationInvitationHandler) Handle() http.HandlerFunc {
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
		invitationID := r.PathValue("invitation_id")
		invitation, err := h.OrgInvitationService.GetOrganizationInvitation(ctx, userID, organizationID, invitationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitation)
	}
}

type RevokeOrganizationInvitationHandler struct {
	OrgInvitationService orgservices.OrganizationInvitationService
}

func (h *RevokeOrganizationInvitationHandler) Handle() http.HandlerFunc {
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
		invitationID := r.PathValue("invitation_id")
		invitation, err := h.OrgInvitationService.RevokeOrganizationInvitation(ctx, userID, organizationID, invitationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitation)
	}
}

type AcceptOrganizationInvitationHandler struct {
	OrgInvitationService orgservices.OrganizationInvitationService
}

func (h *AcceptOrganizationInvitationHandler) Handle() http.HandlerFunc {
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
		invitationID := r.PathValue("invitation_id")
		invitation, err := h.OrgInvitationService.AcceptOrganizationInvitation(ctx, userID, organizationID, invitationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.Values[models.ContextAccessControlAssignRole.String()] = &models.AccessControlAssignRoleContext{
			UserID:         userID,
			RoleName:       invitation.Role,
			AssignerUserID: &invitation.InviterID,
		}

		var request types.AcceptOrganizationInvitationRequest
		request.RedirectURL = r.URL.Query().Get("redirect_url")
		request.Trim()
		if request.RedirectURL != "" {
			reqCtx.RedirectURL = request.RedirectURL
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitation)
	}
}

type RejectOrganizationInvitationHandler struct {
	OrgInvitationService orgservices.OrganizationInvitationService
}

func (h *RejectOrganizationInvitationHandler) Handle() http.HandlerFunc {
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
		invitationID := r.PathValue("invitation_id")
		invitation, err := h.OrgInvitationService.RejectOrganizationInvitation(ctx, userID, organizationID, invitationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitation)
	}
}
