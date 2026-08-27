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

type AddOrganizationMemberHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *AddOrganizationMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")

		var request types.AddOrganizationMemberRequest
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

		member, err := h.UseCases.AddMember(ctx, actor, organizationID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.Values[models.ContextAccessControlAssignRole.String()] = &models.AccessControlAssignRoleContext{
			UserID:         request.UserID,
			RoleName:       request.Role,
			AssignerUserID: &actor.ID,
		}

		reqCtx.SetJSONResponse(http.StatusCreated, member)
	}
}

type ListAllOrganizationMembersHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *ListAllOrganizationMembersHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		paginationParams := pagination.ParseFromRequest(r)
		members, err := h.UseCases.ListAllMembers(ctx, actor, organizationID, paginationParams)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, members)
	}
}

type GetOrganizationMemberHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *GetOrganizationMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		memberID := r.PathValue("member_id")
		member, err := h.UseCases.GetMember(ctx, actor, organizationID, memberID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, member)
	}
}

type GetOrganizationMemberByUserIDHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *GetOrganizationMemberByUserIDHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		userID := r.PathValue("user_id")
		member, err := h.UseCases.GetMemberByUserID(ctx, actor, organizationID, userID)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, member)
	}
}

type UpdateOrganizationMemberHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *UpdateOrganizationMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		memberID := r.PathValue("member_id")

		var request types.UpdateOrganizationMemberRequest
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

		member, err := h.UseCases.UpdateMember(ctx, actor, organizationID, memberID, request)
		if err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.Values[models.ContextAccessControlAssignRole.String()] = &models.AccessControlAssignRoleContext{
			UserID:         member.UserID,
			RoleName:       request.Role,
			AssignerUserID: &actor.ID,
		}

		reqCtx.SetJSONResponse(http.StatusOK, member)
	}
}

type DeleteOrganizationMemberHandler struct {
	UseCases *orgusecases.UseCases
}

func (h *DeleteOrganizationMemberHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		actor := reqCtx.Actor

		organizationID := r.PathValue("organization_id")
		memberID := r.PathValue("member_id")
		if err := h.UseCases.RemoveMember(ctx, actor, organizationID, memberID); err != nil {
			orgconstants.HandleError(err, reqCtx)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, types.DeleteOrganizationMemberResponse{Message: "organization member deleted"})
	}
}
