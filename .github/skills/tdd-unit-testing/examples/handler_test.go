package todos_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type Todo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// MockTodoUseCase simulates the use-case in examples.
type MockTodoUseCase struct {
	mock.Mock
}

// GetByID returns the given todo by ID.
func (m *MockTodoUseCase) GetByID(ctx context.Context, id string) (Todo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Todo), args.Error(1)
}

// GetTodoHandler is a minimal HTTP handler showing path‑value access.
type GetTodoHandler struct {
	UseCase *MockTodoUseCase
}

func NewGetTodoHandler(useCase *MockTodoUseCase) *GetTodoHandler {
	return &GetTodoHandler{UseCase: useCase}
}

func (h *GetTodoHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		todoID := r.PathValue("todo_id")

		todo, err := h.UseCase.GetByID(ctx, todoID)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}
		if todo == nil {
			reqCtx.SetJSONResponse(http.StatusNotFound, map[string]any{"message": "todo not found"})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"todo": todo})
	}
}

func TestGetTodoHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		todoID         string
		setup          func(*MockTodoUseCase)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "success",
			todoID: "1",
			setup: func(m *MockTodoUseCase) {
				m.On("GetByID", mock.Anything, "1").Return(Todo{ID: "1", Title: "make coffee"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "make coffee",
		},
		{
			name:   "not found",
			todoID: "2",
			setup: func(m *MockTodoUseCase) {
				m.On("GetByID", mock.Anything, "2").Return(Todo{}, nil).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "todo not found",
		},
		{
			name:   "use case error",
			todoID: "err",
			setup: func(m *MockTodoUseCase) {
				m.On("GetByID", mock.Anything, "err").Return(Todo{}, context.Canceled).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   context.Canceled.Error(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := new(MockTodoUseCase)
			if tc.setup != nil {
				tc.setup(mockUseCase)
			}
			handler := NewGetTodoHandler(mockUseCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/todos/"+tc.todoID, nil)
			req.SetPathValue("todo_id", tc.todoID)

			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedBody != "" {
				payload := internaltests.DecodeResponseJSON[map[string]string](t, reqCtx)
				if payload["message"] != tc.expectedBody && payload["title"] != tc.expectedBody {
					t.Fatalf("unexpected body: %v", payload)
				}
			}

			mockUseCase.AssertExpectations(t)
		})
	}
}
