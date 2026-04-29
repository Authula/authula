package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
)

type AddOrganizationMemberHandler struct {
	OrgMemberService orgservices.OrganizationMemberService
}

func (h *AddOrganizationMemberHandler) Handle() http.HandlerFunc {
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

		var request types.AddOrganizationMemberRequest
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

		member, err := h.OrgMemberService.AddMember(ctx, userID, organizationID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.Values[models.ContextAccessControlAssignRole.String()] = &models.AccessControlAssignRoleContext{
			UserID:         request.UserID,
			RoleName:       request.Role,
			AssignerUserID: &userID,
		}

		reqCtx.SetJSONResponse(http.StatusCreated, member)
	}
}

type GetAllOrganizationMembersHandler struct {
	OrgMemberService orgservices.OrganizationMemberService
}

func (h *GetAllOrganizationMembersHandler) Handle() http.HandlerFunc {
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
		page := util.GetQueryInt(r, "page", 1)
		limit := util.GetQueryInt(r, "limit", 10)
		members, err := h.OrgMemberService.GetAllMembers(ctx, userID, organizationID, page, limit)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, members)
	}
}

type GetOrganizationMemberHandler struct {
	OrgMemberService orgservices.OrganizationMemberService
}

func (h *GetOrganizationMemberHandler) Handle() http.HandlerFunc {
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
		memberID := r.PathValue("member_id")
		member, err := h.OrgMemberService.GetMember(ctx, userID, organizationID, memberID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, member)
	}
}

type UpdateOrganizationMemberHandler struct {
	OrgMemberService orgservices.OrganizationMemberService
}

func (h *UpdateOrganizationMemberHandler) Handle() http.HandlerFunc {
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
		memberID := r.PathValue("member_id")

		var request types.UpdateOrganizationMemberRequest
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

		member, err := h.OrgMemberService.UpdateMember(ctx, userID, organizationID, memberID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.Values[models.ContextAccessControlAssignRole.String()] = &models.AccessControlAssignRoleContext{
			UserID:         member.UserID,
			RoleName:       request.Role,
			AssignerUserID: &userID,
		}

		reqCtx.SetJSONResponse(http.StatusOK, member)
	}
}

type DeleteOrganizationMemberHandler struct {
	OrgMemberService orgservices.OrganizationMemberService
}

func (h *DeleteOrganizationMemberHandler) Handle() http.HandlerFunc {
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
		memberID := r.PathValue("member_id")
		if err := h.OrgMemberService.RemoveMember(ctx, userID, organizationID, memberID); err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.DeleteOrganizationMemberResponse{Message: "organization member deleted"})
	}
}
