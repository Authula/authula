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
	tests := []struct {
		name            string
		seedUserID      string
		seedEmail       string
		authenticatedAs string
		payload         map[string]any
		wantStatus      int
		checkResponse   func(t *testing.T, w *http.Response)
	}{
		{
			name:            "authenticated - creates todo successfully",
			seedUserID:      "alice",
			seedEmail:       "alice@example.com",
			authenticatedAs: "alice",
			payload:         map[string]any{"title": "Learn testing"},
			wantStatus:      http.StatusCreated,
			checkResponse: func(t *testing.T, w *http.Response) {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response["id"])
			},
		},
		{
			name:       "unauthenticated - returns 401",
			payload:    map[string]any{"title": "Learn testing"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:            "invalid payload - returns validation error",
			seedUserID:      "charlie",
			seedEmail:       "charlie@example.com",
			authenticatedAs: "charlie",
			payload:         map[string]any{},
			wantStatus:      http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTodosFixture(t)
			if tt.seedUserID != "" {
				f.SeedUser(tt.seedUserID, tt.seedEmail)
			}
			if tt.authenticatedAs != "" {
				f.AuthenticateAs(tt.authenticatedAs)
			}

			w := f.JSONRequest(http.MethodPost, "/todos", tt.payload)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestGetTodo(t *testing.T) {
	tests := []struct {
		name            string
		seedUserID      string
		seedEmail       string
		authenticatedAs string
		createTodo      string
		path            string
		wantStatus      int
		checkResponse   func(t *testing.T, w *http.Response)
	}{
		{
			name:            "authenticated - returns todo",
			seedUserID:      "alice",
			seedEmail:       "alice@example.com",
			authenticatedAs: "alice",
			createTodo:      "Read documentation",
			path:            "/todos/{todoID}",
			wantStatus:      http.StatusOK,
			checkResponse: func(t *testing.T, w *http.Response) {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Read documentation", response["title"])
			},
		},
		{
			name:       "unauthenticated - returns 401",
			path:       "/todos/todo-1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:            "not found - returns 404",
			seedUserID:      "bob",
			seedEmail:       "bob@example.com",
			authenticatedAs: "bob",
			path:            "/todos/nonexistent-id",
			wantStatus:      http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTodosFixture(t)
			if tt.seedUserID != "" {
				f.SeedUser(tt.seedUserID, tt.seedEmail)
			}
			if tt.authenticatedAs != "" {
				f.AuthenticateAs(tt.authenticatedAs)
			}

			path := tt.path
			if tt.createTodo != "" {
				todoID := f.CreateTodo(tt.createTodo)
				path = "/todos/" + todoID
			}

			w := f.JSONRequest(http.MethodGet, path, nil)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestMarkComplete(t *testing.T) {
	tests := []struct {
		name            string
		seedUsers       []struct{ id, email string }
		authenticatedAs string
		createTodo      string
		path            string
		requestAs       string
		wantStatus      int
	}{
		{
			name:            "success - marks todo complete",
			seedUsers:       []struct{ id, email string }{{id: "bob", email: "bob@example.com"}},
			authenticatedAs: "bob",
			createTodo:      "Fix bug",
			path:            "/todos/{todoID}/complete",
			wantStatus:      http.StatusOK,
		},
		{
			name:       "unauthenticated - returns 401",
			path:       "/todos/todo-1/complete",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:            "not found - returns 404",
			seedUsers:       []struct{ id, email string }{{id: "alice", email: "alice@example.com"}},
			authenticatedAs: "alice",
			path:            "/todos/nonexistent-id/complete",
			wantStatus:      http.StatusNotFound,
		},
		{
			name:            "forbidden - marks other user's todo",
			seedUsers:       []struct{ id, email string }{{id: "alice", email: "alice@example.com"}, {id: "bob", email: "bob@example.com"}},
			authenticatedAs: "alice",
			createTodo:      "Alice's task",
			requestAs:       "bob",
			path:            "/todos/{todoID}/complete",
			wantStatus:      http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTodosFixture(t)
			for _, user := range tt.seedUsers {
				f.SeedUser(user.id, user.email)
			}
			if tt.authenticatedAs != "" {
				f.AuthenticateAs(tt.authenticatedAs)
			}

			path := tt.path
			if tt.createTodo != "" {
				todoID := f.CreateTodo(tt.createTodo)
				path = "/todos/" + todoID + "/complete"
			}
			if tt.requestAs != "" {
				f.AuthenticateAs(tt.requestAs)
			}

			w := f.JSONRequest(http.MethodPut, path, nil)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
