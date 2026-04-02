---
name: tdd-unit-testing
description: Write unit tests in Go following Red-Green-Refactor TDD discipline with mocked dependencies.
---

# TDD & Unit Testing

## When to use this skill

- Implement new features (write test first)
- Add test coverage for code changes
- Test business logic, services, and handlers
- Ensure error paths are covered

## Key principles

1. **Red-Green-Refactor**: Write failing test → implement → refactor
2. **Mock dependencies**: Use testify/mock to isolate units
3. **Table-driven tests**: Use `tt` patterns for multiple cases
4. **One behavior per test**: Keep tests focused and small
5. **100% coverage target**: Test success and error paths
6. **Descriptive names**: `TestTodoService_CreateTodo_ReturnsID_OnSuccess`

## Testing strategy

**Handlers**: Create handler struct with UseCase field; return `http.HandlerFunc` from `Handler()` method; test via httptest
**Services**: Mock repositories; test business logic and error handling
**Repositories**: Test against real SQLite database (Bun ORM); use test fixtures to set up schema
**Integration tests**: Use fixtures; test plugin routes end-to-end

## Pattern

Every test follows Arrange-Act-Assert (AAA):

1. **Arrange**: Create mocks, set expectations, prepare test data
2. **Act**: Call the function under test
3. **Assert**: Verify results and confirm all mock expectations were met

## Example files

See [examples/](examples/) for Todos testing patterns:

- `test_helpers.go` - MockTodoUseCase and MockTodoService interfaces
- `handler_test.go` - Handler struct with UseCase, Handler() method, httptest patterns
- `repository_test.go` - Real SQLite database tests with test fixtures, CRUD operations
- `todo_service_test.go` - Service tests with mocked repositories and table-driven tests
- `plugin_integration_test.go` - End-to-end route testing with fixtures

## Common mistakes

1. Not writing test first (Red-Green-Refactor discipline)
2. Testing multiple behaviors in one test
3. Using real database instead of mocks
4. Skipping error cases and edge cases
5. Not asserting mock expectations with `AssertExpectations(t)`
6. Tests that break on harmless refactoring

## Quick commands

```bash
go test ./...              # Run all tests
go test -cover ./...       # With coverage
go test -run TestFunc ...  # Specific test
go test -race ./...        # Detect race conditions
```
