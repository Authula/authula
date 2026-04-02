package examples

import "context"

// TodoService interface
type TodoService interface {
	CreateTodo(ctx context.Context, title string, userID string) (*Todo, error)
	MarkComplete(ctx context.Context, todoID string) (*Todo, error)
	DeleteTodo(ctx context.Context, todoID string) error
	GetTodo(ctx context.Context, todoID string) (*Todo, error)
}

// todoService implementation
type todoService struct {
	repo TodoRepository
}

func NewTodoService(repo TodoRepository) TodoService {
	return &todoService{
		repo: repo,
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
	return s.repo.Create(ctx, todo)
}

func (s *todoService) MarkComplete(ctx context.Context, todoID string) (*Todo, error) {
	todo, err := s.repo.GetByID(ctx, todoID)
	if err != nil {
		return nil, err
	}
	if todo == nil {
		return nil, ErrNotFound
	}
	todo.Completed = true
	return s.repo.Update(ctx, todo)
}

func (s *todoService) DeleteTodo(ctx context.Context, todoID string) error {
	return s.repo.Delete(ctx, todoID)
}

func (s *todoService) GetTodo(ctx context.Context, todoID string) (*Todo, error) {
	return s.repo.GetByID(ctx, todoID)
}

// Utility
var (
	ErrEmptyTitle = nil // In real code, proper error type
	ErrNotFound   = nil
)

func generateID() string {
	return "id" // Simplified for example
}
