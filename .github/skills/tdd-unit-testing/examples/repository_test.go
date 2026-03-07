package todos_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uptake/go-bun"
	"github.com/uptake/go-bun/driver/sqliteshim"
)

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
	// In real code, use UUID library
	return "abc123"
}

// ================== REAL DATABASE TESTS ==================

// TestTodoRepository_Create_Success tests CREATE against real SQLite via Bun.
func TestTodoRepository_Create_Success(t *testing.T) {
	t.Parallel()

	// Arrange: Set up in-memory SQLite database with Bun
	db := setupTestDB(t)
	repo := NewTodoRepository(db)

	// Act
	ctx := context.Background()
	todoID, err := repo.Create(ctx, "Learn Go testing")

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, todoID)

	// Verify it was actually inserted
	title, completed, err := repo.GetByID(ctx, todoID)
	assert.NoError(t, err)
	assert.Equal(t, "Learn Go testing", title)
	assert.False(t, completed)
}

// TestTodoRepository_GetByID_NotFound shows testing error cases.
func TestTodoRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewTodoRepository(db)

	// Act: Retrieve non-existent todo
	ctx := context.Background()
	_, _, err := repo.GetByID(ctx, "nonexistent")

	// Assert: Should get error
	assert.Error(t, err)
}

// TestTodoRepository_MarkComplete tests UPDATE operation.
func TestTodoRepository_MarkComplete(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewTodoRepository(db)
	ctx := context.Background()

	// Arrange: Create a todo first
	todoID, err := repo.Create(ctx, "Fix bug")
	assert.NoError(t, err)

	// Act: Mark complete
	err = repo.MarkComplete(ctx, todoID)

	// Assert
	assert.NoError(t, err)

	// Verify update succeeded
	_, completed, err := repo.GetByID(ctx, todoID)
	assert.NoError(t, err)
	assert.True(t, completed)
}

// TestTodoRepository_TableDriven shows multiple operations in one test.
func TestTodoRepository_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{name: "simple title", title: "Buy milk", wantErr: false},
		{name: "long title", title: "Call mom about the party on Saturday", wantErr: false},
		{name: "empty title", title: "", wantErr: false}, // DB allows empty, validation in service layer
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewTodoRepository(db)
			ctx := context.Background()

			// Act
			todoID, err := repo.Create(ctx, tt.title)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, todoID)

				// Verify retrieval
				title, _, err := repo.GetByID(ctx, todoID)
				assert.NoError(t, err)
				assert.Equal(t, tt.title, title)
			}
		})
	}
}

// ================== TEST FIXTURE ==================

// setupTestDB creates an in-memory SQLite database with Bun ORM.
func setupTestDB(t *testing.T) bun.IDB {
	// Open in-memory SQLite
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	assert.NoError(t, err)

	// Create Bun database instance
	db := bun.NewDB(sqldb)

	t.Cleanup(func() {
		db.Close()
	})

	// Register models for schema creation
	db.RegisterModel((*Todo)(nil))

	// Create todos table using Bun schema
	ctx := context.Background()
	_, err = db.NewCreateTable().Model((*Todo)(nil)).Exec(ctx)
	assert.NoError(t, err)

	return db
}
