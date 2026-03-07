package examples

import (
	"net/http"

	"github.com/uptightgo/bun"
)

// Type aliases for plugin interfaces (simplified)
type PluginContext struct {
	Database bun.IDB
}

type Route struct {
	Method  string
	Path    string
	Handler func(w http.ResponseWriter, r *http.Request) error
}

// TodosPlugin
type TodosPlugin struct {
	todoService         TodoService
	createTodoUseCase   *CreateTodoUseCase
	markCompleteUseCase *MarkTodoCompleteUseCase
}

func (p *TodosPlugin) Init(ctx *PluginContext) error {
	// Step 1: Create repository with database
	todoRepo := NewBunTodoRepository(ctx.Database)

	// Step 2: Create service
	p.todoService = NewTodoService(todoRepo)

	// Step 3: Create use cases
	p.createTodoUseCase = NewCreateTodoUseCase(p.todoService)
	p.markCompleteUseCase = NewMarkTodoCompleteUseCase(p.todoService)

	return nil
}

func (p *TodosPlugin) Routes() []Route {
	createHandler := NewCreateTodoHandler(p.createTodoUseCase)
	completeHandler := NewMarkTodoCompleteHandler(p.markCompleteUseCase)

	return []Route{
		{
			Method: http.MethodPost,
			Path:   "/todos",
			Handler: func(w http.ResponseWriter, r *http.Request) error {
				return createHandler.Handle(w, r)
			},
		},
		{
			Method: http.MethodPut,
			Path:   "/todos/{id}/complete",
			Handler: func(w http.ResponseWriter, r *http.Request) error {
				return completeHandler.Handle(w, r)
			},
		},
	}
}

func (p *TodosPlugin) Close() error {
	return nil
}
