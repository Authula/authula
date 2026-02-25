---
name: tdd-unit-testing
description: Write unit tests in Go following strict Red-Green-Refactor TDD discipline. Use when the user asks for new features, bug fixes, refactoring, or code changes that require tests.
---

# When to use this skill

- User asks something similar to "implement", "add", "fix", "refactor", or "change" any Go code.
- User mentions features, endpoints, services, repositories, or business logic.
- User says something similar to "write tests for", "add test coverage", or "make it testable".
- Any task where untested code would be produced.

# TDD Workflow (MANDATORY)

**Always follow Red-Green-Refactor cycle exactly:**

1. **Red**: Write a failing test first that expresses the desired behavior.
   - Make it obvious why it fails.
   - Run `go test ./... -v` to confirm failure.

2. **Green**: Write the *minimum* production code to make the test pass.
   - No refactoring yet; just make it green.
   - Run `go test ./...` to confirm it passes.

3. **Refactor**: Clean up code and tests while keeping all tests green.
   - Improve names, remove duplication, extract helpers.
   - Run `go test ./...` after every refactor.

**Never** write production code first. Never write multiple tests without running them.

# Testing conventions

- **Table-driven tests** for multiple cases (use `tt` struct pattern).
- **100% test coverage** for new/changed code (`go test -cover`).
- **No external dependencies** in unit tests (mock interfaces, use `testify/mock` if needed).
- **Descriptive test names**: `TestUserService_GetByID_ReturnsUser_WhenExists`, etc.
- **Arrange-Act-Assert** structure with blank lines.
- **Use `t.Parallel()`** for independent tests.
- **Race condition checks**: Run tests with `-race` flag to ensure thread safety.
- **Test edge cases** and error conditions, not just happy paths.
- **Minimal tests**: Keep tests focused on one behavior per test function. Avoid testing multiple things in one test. Keep tests small and focused.

# Plugin Integration Testing (Integration Fixture Pattern)

For testing plugins, follow the **Integration Fixture Pattern**. This pattern exercises the full HTTP lifecycle including routes, hooks, and database persistence.

- **File Organization**: Use `routes_integration_test.go` and `test_helpers_integration_test.go`.
- **Fixture-based Testing**: Extend `BaseTestFixture` to set up a fresh database, router and auth stack for each test.
- **ID Aliasing**: Use `ResolveID("test-user")` for stable, readable IDs in tests.
- **Hook Injection**: Register temporary `models.HookBefore` to mock authentication or context values.
- **Declarative RBAC**: Test permission enforcement by mapping plugin routes to required permissions in the test setup.

See [references/integration_testing_conventions.md](references/integration_testing_conventions.md) and [examples/plugin_integration_test.go](examples/plugin_integration_test.go) for details.

# Example patterns

Follow `examples/user_service_test.go` exactly for new tests.

## Mocking repositories/services
Use `testify/mock` for interface dependencies.

## Error cases
Test both happy path and every error condition.

# Instructions for the agent

1. Identify the feature/behavior to test.
2. Write ONE failing test first (Red).
3. Implement minimal code to pass it (Green).
4. Refactor both test and code.
5. Add more tests following the same cycle.
6. Show `go test ./... -cover -v` output confirming 100% coverage.
7. If refactoring existing code, write tests for new/changed behavior first.

**Output format**:

Red: Failing test

```go
// test code...
```

`go test` output showing failure

Green: Minimal implementation

```go
// minimal code to pass...
```

`go test` output showing success

Refactor: Clean implementation

```go
// final refactored code...
```

`go test` output showing 100% coverage

