package examples

import (
	"net/http"
	"strings"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
)

type ITodoService interface {
	Create(ctx models.RequestContext, req *CreateTodoRequest) (*Todo, error)
	MarkAsComplete(ctx models.RequestContext, req *MarkTodoCompleteRequest) (*Todo, error)
}

type CreateTodoRequest struct {
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	SomeOtherField *string `json:"some_other_field,omitempty"`
}

func (r *CreateTodoRequest) Trim() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	if r.SomeOtherField != nil {
		*r.SomeOtherField = strings.TrimSpace(*r.SomeOtherField)
	}
}

type CreateTodoResponse struct {
	Todo *Todo `json:"todo"`
}

// CreateTodoHandler handles POST /todos
type CreateTodoHandler struct {
	todoService ITodoService
}

func NewCreateTodoHandler(todoService ITodoService) *CreateTodoHandler {
	return &CreateTodoHandler{todoService: todoService}
}

func (h *CreateTodoHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
}

// MarkTodoCompleteHandler handles PUT /todos/{id}/complete

type MarkTodoCompleteRequest struct {
	TodoID string `json:"todo_id"`
}

func (r *MarkTodoCompleteRequest) Trim() {
	r.TodoID = strings.TrimSpace(r.TodoID)
}

type MarkTodoCompleteResponse struct {
	Todo *Todo `json:"todo"`
}
type MarkTodoCompleteHandler struct {
	todoService ITodoService
}

func NewMarkTodoCompleteHandler(todoService ITodoService) *MarkTodoCompleteHandler {
	return &MarkTodoCompleteHandler{todoService: todoService}
}

func (h *MarkTodoCompleteHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		todoID := r.PathValue("id")

		completedTodo, err := h.todoService.MarkAsComplete(ctx, &MarkTodoCompleteRequest{TodoID: todoID})
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &MarkTodoCompleteResponse{
			Todo: completedTodo,
		})
	}
}
