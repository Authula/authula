package examples

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// CreateTodoHandler handles POST /todos
type CreateTodoHandler struct {
	todoService *TodoService
}

func NewCreateTodoHandler(todoService *TodoService) *CreateTodoHandler {
	return &CreateTodoHandler{todoService: todoService}
}

func (h *CreateTodoHandler) Handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	reqCtx, _ := models.GetRequestContext(ctx)

	var payload CreateTodoRequest
	if err := util.ParseJSON(r, &payload); err != nil {
		reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
		reqCtx.Handled = true
		return
	}

	todoCreated, err := h.todoService.Create(ctx, &payload)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "invalid request body"})
		reqCtx.Handled = true
		return
	}

	reqCtx.SetJSONResponse(http.StatusCreated, &CreateTodoResponse{
		Todo: todoCreated,
	})
}

// MarkTodoCompleteHandler handles PUT /todos/{id}/complete
type MarkTodoCompleteHandler struct {
	todoService *TodoService
}

func NewMarkTodoCompleteHandler(todoService *TodoService) *MarkTodoCompleteHandler {
	return &MarkTodoCompleteHandler{todoService: todoService}
}

func (h *MarkTodoCompleteHandler) Handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	reqCtx, _ := models.GetRequestContext(ctx)
	todoID := r.PathValue("id")

	resp, err := h.todoService.MarkAsComplete(ctx, &MarkTodoCompleteRequest{TodoID: todoID})
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
		reqCtx.Handled = true
		return nil
	}

	reqCtx.SetJSONResponse(http.StatusOK, resp)
	return nil
}
