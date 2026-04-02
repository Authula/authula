package examples

import (
	"context"
	"errors"
)

// Errors

var ErrEmptyTitle = errors.New("title is required")
var ErrUnauthorized = errors.New("unauthorized")

type CreateTodoUseCase struct {
	todoService TodoService
}

type CreateTodoRequest struct {
	Title string `json:"title"`
}

type CreateTodoResponse struct {
	Todo *Todo `json:"todo"`
}

func NewCreateTodoUseCase(todoService TodoService) *CreateTodoUseCase {
	return &CreateTodoUseCase{
		todoService: todoService,
	}
}

func (uc *CreateTodoUseCase) Execute(ctx context.Context, req *CreateTodoRequest) (*CreateTodoResponse, error) {
	if req.Title == "" {
		return nil, ErrEmptyTitle
	}

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, ErrUnauthorized
	}

	todo, err := uc.todoService.CreateTodo(ctx, req.Title, userID)
	if err != nil {
		return nil, err
	}

	return &CreateTodoResponse{Todo: todo}, nil
}

type MarkTodoCompleteUseCase struct {
	todoService TodoService
}

type MarkTodoCompleteRequest struct {
	TodoID string `json:"todo_id"`
}

type MarkTodoCompleteResponse struct {
	Todo *Todo `json:"todo"`
}

func NewMarkTodoCompleteUseCase(todoService TodoService) *MarkTodoCompleteUseCase {
	return &MarkTodoCompleteUseCase{
		todoService: todoService,
	}
}

func (uc *MarkTodoCompleteUseCase) Execute(ctx context.Context, req *MarkTodoCompleteRequest) (*MarkTodoCompleteResponse, error) {
	todo, err := uc.todoService.MarkComplete(ctx, req.TodoID)
	if err != nil {
		return nil, err
	}
	return &MarkTodoCompleteResponse{Todo: todo}, nil
}
