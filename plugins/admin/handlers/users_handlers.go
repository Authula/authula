package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type GetAllUsersHandler struct {
	useCase usecases.UsersUseCase
}

func NewGetAllUsersHandler(useCase usecases.UsersUseCase) *GetAllUsersHandler {
	return &GetAllUsersHandler{useCase: useCase}
}

func (h *GetAllUsersHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())

		cursorValue := strings.TrimSpace(r.URL.Query().Get("cursor"))
		var cursor *string
		if cursorValue != "" {
			cursor = &cursorValue
		}

		limit := 20
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil {
				reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid limit"})
				reqCtx.Handled = true
				return
			}
			limit = value
		}

		page, err := h.useCase.GetAll(r.Context(), cursor, limit)
		if err != nil {
			respondUsersError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.UsersPage{
			Users:      page.Users,
			NextCursor: page.NextCursor,
		})
	}
}

type GetUserByIDHandler struct {
	useCase usecases.UsersUseCase
}

func NewGetUserByIDHandler(useCase usecases.UsersUseCase) *GetUserByIDHandler {
	return &GetUserByIDHandler{useCase: useCase}
}

func (h *GetUserByIDHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		user, err := h.useCase.GetByID(r.Context(), userID)
		if err != nil {
			respondUsersError(reqCtx, err)
			return
		}
		if user == nil {
			reqCtx.SetJSONResponse(http.StatusNotFound, map[string]any{"message": "user not found"})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"user": user})
	}
}

type CreateUserHandler struct {
	useCase usecases.UsersUseCase
}

func NewCreateUserHandler(useCase usecases.UsersUseCase) *CreateUserHandler {
	return &CreateUserHandler{useCase: useCase}
}

func (h *CreateUserHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())

		var payload types.CreateUserRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		user, err := h.useCase.Create(r.Context(), payload)
		if err != nil {
			respondUsersError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, map[string]any{"user": user})
	}
}

type UpdateUserHandler struct {
	useCase usecases.UsersUseCase
}

func NewUpdateUserHandler(useCase usecases.UsersUseCase) *UpdateUserHandler {
	return &UpdateUserHandler{useCase: useCase}
}

func (h *UpdateUserHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		var payload types.UpdateUserRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		user, err := h.useCase.Update(r.Context(), userID, payload)
		if err != nil {
			respondUsersError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"user": user})
	}
}

type DeleteUserHandler struct {
	useCase usecases.UsersUseCase
}

func NewDeleteUserHandler(useCase usecases.UsersUseCase) *DeleteUserHandler {
	return &DeleteUserHandler{useCase: useCase}
}

func (h *DeleteUserHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		if err := h.useCase.Delete(r.Context(), userID); err != nil {
			respondUsersError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "user deleted"})
	}
}

func respondUsersError(reqCtx *models.RequestContext, err error) {
	if reqCtx == nil {
		return
	}

	message := "internal server error"
	if err != nil {
		message = err.Error()
	}

	reqCtx.SetJSONResponse(mapUsersErrorStatus(err), map[string]any{"message": message})
	reqCtx.Handled = true
}

func mapUsersErrorStatus(err error) int {
	return mapAdminErrorStatus(err)
}
