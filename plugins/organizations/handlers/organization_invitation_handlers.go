package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	"github.com/Authula/authula/plugins/organizations/types"
	orgusecases "github.com/Authula/authula/plugins/organizations/usecases"
	"github.com/Authula/authula/util"
)

type CreateOrganizationInvitationHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *CreateOrganizationInvitationHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")

		var request types.CreateOrganizationInvitationRequest
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

		redirectURL := r.URL.Query().Get("redirect_url")
		invitation, err := h.UseCases.CreateOrganizationInvitation(ctx, actor, organizationID, request, redirectURL)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, invitation)
	}
}

type GetAllOrganizationInvitationsHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *GetAllOrganizationInvitationsHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		invitations, err := h.UseCases.GetAllOrganizationInvitations(ctx, actor, organizationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitations)
	}
}

type GetOrganizationInvitationHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *GetOrganizationInvitationHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		invitationID := r.PathValue("invitation_id")
		invitation, err := h.UseCases.GetOrganizationInvitation(ctx, actor, organizationID, invitationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitation)
	}
}

type RevokeOrganizationInvitationHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *RevokeOrganizationInvitationHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		invitationID := r.PathValue("invitation_id")
		invitation, err := h.UseCases.RevokeOrganizationInvitation(ctx, actor, organizationID, invitationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitation)
	}
}

type AcceptOrganizationInvitationHandler struct {
	UseCases       *orgusecases.UseCases
	TrustedOrigins []string
}

func (h *AcceptOrganizationInvitationHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		invitationID := r.PathValue("invitation_id")
		invitation, err := h.UseCases.AcceptOrganizationInvitation(ctx, actor, organizationID, invitationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.Values[models.ContextAccessControlAssignRole.String()] = &models.AccessControlAssignRoleContext{
			UserID:         actor.ID,
			RoleName:       invitation.Role,
			AssignerUserID: &invitation.InviterID,
		}

		redirectURL := r.URL.Query().Get("redirect_url")
		if redirectURL != "" {
			validatedURL, err := util.IsTrustedCallbackURL(redirectURL, h.TrustedOrigins)
			if err != nil {
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": err.Error()})
				reqCtx.Handled = true
				return
			}
			reqCtx.RedirectURL = validatedURL.String()
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitation)
	}
}

type RejectOrganizationInvitationHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *RejectOrganizationInvitationHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		invitationID := r.PathValue("invitation_id")
		invitation, err := h.UseCases.RejectOrganizationInvitation(ctx, actor, organizationID, invitationID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, invitation)
	}
}
