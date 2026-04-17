package examples

import "context"

// Interface
type ITodoService interface {
	CreateTodo(ctx context.Context, title string, userID string) (*Todo, error)
	MarkComplete(ctx context.Context, todoID string) (*Todo, error)
	DeleteTodo(ctx context.Context, todoID string) error
	GetTodo(ctx context.Context, todoID string) (*Todo, error)
}

// Implementation
type todoService struct {
	todoRepo TodoRepository
}

func NewTodoService(todoRepo TodoRepository) ITodoService {
	return &todoService{
		todoRepo: todoRepo,
	}
}

func (s *todoService) CreateTodo(ctx context.Context, title string, userID string) (*Todo, error) {
	if title == "" {
		return nil, ErrEmptyTitle
	}
	todo := &Todo{
		ID:     generateID(),
		Title:  title,
		UserID: userID,
	}
	return s.todoRepo.Create(ctx, todo)
}

func (s *todoService) MarkComplete(ctx context.Context, todoID string) (*Todo, error) {
	todo, err := s.todoRepo.GetByID(ctx, todoID)
	if err != nil {
		return nil, err
	}
	if todo == nil {
		return nil, ErrNotFound
	}
	todo.Completed = true
	return s.todoRepo.Update(ctx, todo)
}

func (s *todoService) DeleteTodo(ctx context.Context, todoID string) error {
	return s.todoRepo.Delete(ctx, todoID)
}

func (s *todoService) GetTodo(ctx context.Context, todoID string) (*Todo, error) {
	return s.todoRepo.GetByID(ctx, todoID)
}

// Utility
var (
	ErrEmptyTitle = nil // In real code, proper error type
	ErrNotFound   = nil
)

func generateID() string {
	return "id" // Simplified for example
}
