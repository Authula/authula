package todos_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Integration test demonstrates testing a route end-to-end with fixtures.
// Use this pattern to test the full plugin initialization, service registration, and HTTP handling.

// Fixture helpers for integration testing (in real code, these would bootstrap the plugin).
// This demonstrates end-to-end testing with all dependencies wired together.
type todosFixture struct {
	// Would contain: database, plugin, router, etc.
}

func newTodosFixture(t *testing.T) *todosFixture {
	// In real tests, this would:
	// 1. Create in-memory database
	// 2. Initialize repositories
	// 3. Create services
	// 4. Create handlers with those services
	// 5. Register routes
	// 6. Return fixture with router for testing
	return &todosFixture{}
}

func (f *todosFixture) SeedUser(id, email string)    { /* insert user in DB */ }
func (f *todosFixture) AuthenticateAs(userID string) { /* set auth context */ }
func (f *todosFixture) JSONRequest(method, path string, body any) *http.Response {
	// In real code:
	// - Encode body as JSON
	// - Create HTTP request
	// - Send through router
	// - Return response
	return nil
}
func (f *todosFixture) CreateTodo(title string) string { return "" }

func TestCreateTodo(t *testing.T) {
	t.Run("authenticated - creates todo successfully", func(t *testing.T) {
		// Arrange: Set up fixtures and seed data
		f := newTodosFixture(t) // Helper that initializes plugin and router
		f.SeedUser("alice", "alice@example.com")
		f.AuthenticateAs("alice")

		// Act: Make HTTP request
		payload := map[string]any{"title": "Learn testing"}
		w := f.JSONRequest(http.MethodPost, "/todos", payload)

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["id"])
	})

	t.Run("unauthenticated - returns 401", func(t *testing.T) {
		// Arrange: No authentication set
		f := newTodosFixture(t)

		// Act: Make HTTP request without auth
		payload := map[string]any{"title": "Learn testing"}
		w := f.JSONRequest(http.MethodPost, "/todos", payload)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid payload - returns validation error", func(t *testing.T) {
		// Arrange
		f := newTodosFixture(t)
		f.SeedUser("charlie", "charlie@example.com")
		f.AuthenticateAs("charlie")

		// Act: Request with missing required field
		payload := map[string]any{} // missing title
		w := f.JSONRequest(http.MethodPost, "/todos", payload)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetTodo(t *testing.T) {
	t.Run("authenticated - returns todo", func(t *testing.T) {
		// Arrange
		f := newTodosFixture(t)
		f.SeedUser("alice", "alice@example.com")
		f.AuthenticateAs("alice")
		todoID := f.CreateTodo("Read documentation")

		// Act
		w := f.JSONRequest(http.MethodGet, "/todos/"+todoID, nil)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Read documentation", response["title"])
	})

	t.Run("unauthenticated - returns 401", func(t *testing.T) {
		// Arrange: Plugin requires authentication
		f := newTodosFixture(t)

		// Act: No authentication set
		w := f.JSONRequest(http.MethodGet, "/todos/todo-1", nil)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("not found - returns 404", func(t *testing.T) {
		// Arrange
		f := newTodosFixture(t)
		f.SeedUser("bob", "bob@example.com")
		f.AuthenticateAs("bob")

		// Act: Request non-existent todo
		w := f.JSONRequest(http.MethodGet, "/todos/nonexistent-id", nil)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestMarkComplete(t *testing.T) {
	t.Run("success - marks todo complete", func(t *testing.T) {
		// Arrange
		f := newTodosFixture(t)
		f.SeedUser("bob", "bob@example.com")
		f.AuthenticateAs("bob")
		todoID := f.CreateTodo("Fix bug")

		// Act
		w := f.JSONRequest(http.MethodPut, "/todos/"+todoID+"/complete", nil)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("unauthenticated - returns 401", func(t *testing.T) {
		// Arrange
		f := newTodosFixture(t)

		// Act
		w := f.JSONRequest(http.MethodPut, "/todos/todo-1/complete", nil)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("not found - returns 404", func(t *testing.T) {
		// Arrange
		f := newTodosFixture(t)
		f.SeedUser("alice", "alice@example.com")
		f.AuthenticateAs("alice")

		// Act
		w := f.JSONRequest(http.MethodPut, "/todos/nonexistent-id/complete", nil)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("forbidden - marks other user's todo", func(t *testing.T) {
		// Arrange
		f := newTodosFixture(t)
		f.SeedUser("alice", "alice@example.com")
		f.SeedUser("bob", "bob@example.com")
		f.AuthenticateAs("alice")
		todoID := f.CreateTodo("Alice's task")

		// Act: Bob tries to complete Alice's todo
		f.AuthenticateAs("bob")
		w := f.JSONRequest(http.MethodPut, "/todos/"+todoID+"/complete", nil)

		// Assert
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
