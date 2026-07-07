package todos_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/testdb"
)

// ================== TEST FIXTURE ==================

// setupTestDB creates a test Postgres database using the shared testcontainer.
func setupTestDB(t *testing.T) bun.IDB {
	return testdb.NewIntegrationTestDB(t)
}

// Todo represents the table structure (Bun model).
type Todo struct {
	ID        string `bun:"id,pk"`
	Title     string `bun:"title"`
	Completed bool   `bun:"completed"`
}

// TodoRepository implements CRUD operations using Bun ORM.
type TodoRepository struct {
	db bun.IDB
}

func NewTodoRepository(db bun.IDB) *TodoRepository {
	return &TodoRepository{db: db}
}

func (r *TodoRepository) Create(ctx context.Context, title string) (string, error) {
	id := "todo-" + randomID()
	todo := &Todo{
		ID:        id,
		Title:     title,
		Completed: false,
	}

	_, err := r.db.NewInsert().Model(todo).Exec(ctx)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *TodoRepository) GetByID(ctx context.Context, id string) (title string, completed bool, err error) {
	todo := new(Todo)
	err = r.db.NewSelect().Model(todo).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return "", false, err
	}
	return todo.Title, todo.Completed, nil
}

func (r *TodoRepository) MarkComplete(ctx context.Context, id string) error {
	_, err := r.db.NewUpdate().Model((*Todo)(nil)).Set("completed = ?", true).Where("id = ?", id).Exec(ctx)
	return err
}

func randomID() string {
	return "abc123"
}

// ================== REAL DATABASE TESTS ==================

func TestTodoRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		title         string
		wantTitle     string
		wantCompleted bool
	}{
		{
			name:          "creates todo successfully",
			title:         "Learn Go testing",
			wantTitle:     "Learn Go testing",
			wantCompleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			ctx := context.Background()
			_, err := db.NewCreateTable().Model((*Todo)(nil)).Exec(ctx)
			assert.NoError(t, err)

			repo := NewTodoRepository(db)

			todoID, err := repo.Create(ctx, tt.title)

			assert.NoError(t, err)
			assert.NotEmpty(t, todoID)

			title, completed, err := repo.GetByID(ctx, todoID)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantTitle, title)
			assert.Equal(t, tt.wantCompleted, completed)
		})
	}
}

func TestTodoRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{
			name: "returns error for missing todo",
			id:   "nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewTodoRepository(db)
			ctx := context.Background()

			_, _, err := repo.GetByID(ctx, tt.id)
			assert.Error(t, err)
		})
	}
}

func TestTodoRepository_MarkComplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		title    string
		wantDone bool
	}{
		{
			name:     "marks todo complete",
			title:    "Fix bug",
			wantDone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			ctx := context.Background()
			_, err := db.NewCreateTable().Model((*Todo)(nil)).Exec(ctx)
			assert.NoError(t, err)

			repo := NewTodoRepository(db)

			todoID, err := repo.Create(ctx, tt.title)
			assert.NoError(t, err)

			err = repo.MarkComplete(ctx, todoID)
			assert.NoError(t, err)

			_, completed, err := repo.GetByID(ctx, todoID)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantDone, completed)
		})
	}
}

func TestTodoRepository_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{
			name:    "simple title",
			title:   "Read documentation",
			wantErr: false,
		},
		{
			name:    "long title",
			title:   "This is a very long title that should still work fine",
			wantErr: false,
		},
		{
			name:    "empty title",
			title:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			ctx := context.Background()
			_, err := db.NewCreateTable().Model((*Todo)(nil)).Exec(ctx)
			assert.NoError(t, err)

			repo := NewTodoRepository(db)

			todoID, err := repo.Create(ctx, tt.title)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, todoID)

				title, _, err := repo.GetByID(ctx, todoID)
				assert.NoError(t, err)
				assert.Equal(t, tt.title, title)
			}
		})
	}
}
