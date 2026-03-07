// Note: These mocks would live in your test package alongside the tests.
// This file shows interfaces and mock implementations for the Todos domain.

package todos_test

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockTodoService demonstrates mocking a service interface in tests.
// Use testify/mock to set up expectations and verify calls without real database access.
type MockTodoService struct {
	mock.Mock
}

// CreateTodo mocks the service method.
func (m *MockTodoService) CreateTodo(ctx context.Context, title string) (todoID string, err error) {
	args := m.Called(ctx, title)
	return args.String(0), args.Error(1)
}

// MarkTodoComplete mocks the service method.
func (m *MockTodoService) MarkTodoComplete(ctx context.Context, todoID string) error {
	args := m.Called(ctx, todoID)
	return args.Error(0)
}

// GetTodo mocks the service method.
func (m *MockTodoService) GetTodo(ctx context.Context, todoID string) (title string, completed bool, err error) {
	args := m.Called(ctx, todoID)
	return args.String(0), args.Bool(1), args.Error(2)
}

// MockTodoRepository demonstrates mocking a repository interface.
type MockTodoRepository struct {
	mock.Mock
}

// Create mocks repository insert.
func (m *MockTodoRepository) Create(ctx context.Context, title string) (todoID string, err error) {
	args := m.Called(ctx, title)
	return args.String(0), args.Error(1)
}

// GetByID mocks repository select.
func (m *MockTodoRepository) GetByID(ctx context.Context, todoID string) (title string, completed bool, err error) {
	args := m.Called(ctx, todoID)
	return args.String(0), args.Bool(1), args.Error(2)
}

// MarkComplete mocks repository update.
func (m *MockTodoRepository) MarkComplete(ctx context.Context, todoID string) error {
	args := m.Called(ctx, todoID)
	return args.Error(0)
}
