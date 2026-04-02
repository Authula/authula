package examples

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

// TodoRepository interface definition
type TodoRepository interface {
	Create(ctx context.Context, todo *Todo) (*Todo, error)
	GetByID(ctx context.Context, id string) (*Todo, error)
	Update(ctx context.Context, todo *Todo) (*Todo, error)
	Delete(ctx context.Context, id string) error
	GetAllByUserID(ctx context.Context, userID string) ([]Todo, error)
	WithTx(tx bun.IDB) TodoRepository
}

type Todo struct {
	ID        string `bun:"id,pk,column:id"`
	UserID    string `bun:"user_id,column:user_id"`
	Title     string `bun:"title,column:title"`
	Completed bool   `bun:"completed,column:completed"`
}

// Bun implementation
type bunTodoRepository struct {
	db bun.IDB
}

func NewBunTodoRepository(db bun.IDB) TodoRepository {
	return &bunTodoRepository{db: db}
}

func (r *bunTodoRepository) Create(ctx context.Context, todo *Todo) (*Todo, error) {
	_, err := r.db.NewInsert().Model(todo).Exec(ctx)
	return todo, err
}

func (r *bunTodoRepository) GetByID(ctx context.Context, id string) (*Todo, error) {
	todo := &Todo{}
	err := r.db.NewSelect().Model(todo).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return todo, nil
}

func (r *bunTodoRepository) Update(ctx context.Context, todo *Todo) (*Todo, error) {
	_, err := r.db.NewUpdate().Model(todo).Where("id = ?", todo.ID).Exec(ctx)
	return todo, err
}

func (r *bunTodoRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model(&Todo{}).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *bunTodoRepository) GetAllByUserID(ctx context.Context, userID string) ([]Todo, error) {
	var todos []Todo
	err := r.db.NewSelect().Model(&todos).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []Todo{}, nil
		}
		return nil, err
	}
	return todos, nil
}

func (r *bunTodoRepository) WithTx(tx bun.IDB) TodoRepository {
	return &bunTodoRepository{db: tx}
}
