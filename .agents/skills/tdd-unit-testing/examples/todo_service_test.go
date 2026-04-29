package todos_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestTodoService_CreateTodo demonstrates table-driven test pattern for creating todos.
func TestTodoService_CreateTodo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		title   string
		setup   func(*MockTodoRepository)
		wantErr bool
		wantID  string
	}{
		{
			name:  "create todo successfully",
			title: "Fix bug",
			setup: func(mockTodoRepo *MockTodoRepository) {
				mockTodoRepo.On("Create", mock.Anything, "Fix bug").
					Return("todo-111", nil).
					Once()
			},
			wantID: "todo-111",
		},
		{
			name:  "create todo with empty title",
			title: "",
			setup: func(mockTodoRepo *MockTodoRepository) {
				mockTodoRepo.On("Create", mock.Anything, "").
					Return("", errors.New("empty title")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockTodoRepository)
			tt.setup(mockRepo)

			// Act
			ctx := context.Background()
			todoID, err := mockRepo.Create(ctx, tt.title)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, todoID)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestTodoService_GetTodo_TableDriven demonstrates table-driven test pattern.
func TestTodoService_GetTodo_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		todoID    string
		setup     func(*MockTodoRepository)
		wantErr   bool
		wantTitle string
	}{
		{
			name:   "todo exists",
			todoID: "todo-1",
			setup: func(mockTodoRepo *MockTodoRepository) {
				mockTodoRepo.On("GetByID", mock.Anything, "todo-1").
					Return("Write tests", false, nil)
			},
			wantTitle: "Write tests",
		},
		{
			name:   "todo not found",
			todoID: "todo-999",
			setup: func(mockTodoRepo *MockTodoRepository) {
				mockTodoRepo.On("GetByID", mock.Anything, "todo-999").
					Return("", false, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockTodoRepository)
			tt.setup(mockRepo)

			// Act
			title, _, err := mockRepo.GetByID(context.Background(), tt.todoID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTitle, title)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
