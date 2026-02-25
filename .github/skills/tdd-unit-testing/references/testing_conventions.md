# Go TDD Unit Testing Conventions

## Test File Naming

- Same package name as production code (black box testing)
- Filename: `{production_file}_test.go`

## Test Structure

```go
func Test_<ServiceOrFunc>_<ScenarioOrMethod>_<ExpectedBehavior>(t *testing.T) {
  t.Parallel()
  // ... test code
}
```

```go
// Arrange (setup mocks, test data). Use testify/mock for mocks.

// Act (call the function)

// Assert (verify results, mock calls). Use testify/assert for assertions.
```

## Coverage Requirements

- New code: 100% coverage
- Changed code: 100% coverage for changed lines
- Use `go test ./... -coverprofile=coverage.out -covermode=atomic`

## Mocking Rules

- Mock interfaces only (never structs)
- Use testify/mock.MockedObject pattern
- Assert expectations with AssertExpectations(t)

## Common Test Helpers

See test_utils.go for reusable mocks
