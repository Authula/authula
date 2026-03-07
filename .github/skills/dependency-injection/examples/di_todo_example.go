package examples

import (
	"github.com/uptrace/bun"
)

// BuildTodosPlugin demonstrates the complete DI chain
func BuildTodosPlugin(db bun.IDB) *TodosPlugin {
	// Step 1: Repository (database layer)
	todoRepo := NewBunTodoRepository(db)

	// Step 2: Services (business logic)
	todoService := NewTodoService(todoRepo)

	// Step 3: Use Cases (orchestration)
	createUseCase := NewCreateTodoUseCase(todoService)
	markCompleteUseCase := NewMarkTodoCompleteUseCase(todoService)

	// Step 4: Plugin (lifecycle management)
	plugin := &TodosPlugin{
		todoService:         todoService,
		createTodoUseCase:   createUseCase,
		markCompleteUseCase: markCompleteUseCase,
	}

	return plugin
}

// Dependency graph:
//
// Database (bun.IDB)
//     ↓
// TodoRepository
//     ↓
// TodoService
//     ↓  ↓
// CreateTodoUseCase  MarkTodoCompleteUseCase
//     ↓                      ↓
// CreateTodoHandler  MarkTodoCompleteHandler
//     ↓                      ↓
// HTTP Routes
//
// Unidirectional: Each layer depends only on layers below it
// No circular dependencies
// All dependencies injected at construction time
