# Testing Framework for HybridPipe.io

This document describes the testing framework for the HybridPipe.io repository.

## Overview

The testing framework for HybridPipe.io is built using Go's standard testing package, along with additional tools and libraries to enhance testing capabilities. The framework includes:

- Unit tests for individual packages and modules
- Integration tests for testing interactions between components
- Mocking for external dependencies
- Test utilities for common testing tasks
- Coverage reporting

## Running Tests

### Running All Tests

To run all tests in the repository, use the following command:

```bash
go test ./...
```

Or use the Makefile target:

```bash
make test
```

### Running Short Tests

To run only short tests (skipping long-running tests), use the following command:

```bash
go test -short ./...
```

Or use the Makefile target:

```bash
make test-short
```

### Running Tests with Verbose Output

To run tests with verbose output, use the following command:

```bash
go test -v ./...
```

Or use the Makefile target:

```bash
make test-verbose
```

### Running Tests with Race Detection

To run tests with race detection, use the following command:

```bash
go test -race ./...
```

Or use the Makefile target:

```bash
make test-race
```

## Test Coverage

### Generating Coverage Reports

To generate a test coverage report, use the following command:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

Or use the Makefile target:

```bash
make coverage
```

To generate an HTML coverage report and open it in the default browser, use the following command:

```bash
make coverage-html
```

Alternatively, you can use the provided scripts:

- For Linux/macOS: `scripts/coverage.sh`
- For Windows: `scripts/coverage.ps1`

## Test Structure

### Test Files

Test files are named with the `_test.go` suffix and are placed in the same package as the code they test. For example, the tests for `core/hybridpipe.go` are in `core/hybridpipe_test.go`.

### Test Functions

Test functions are named with the `Test` prefix followed by the name of the function or feature being tested. For example, the test for the `Deploy` function is named `TestDeploy`.

### Test Helpers

Test helper functions are named with a lowercase prefix followed by a descriptive name. For example, `testBasicOperations` is a helper function for testing basic operations.

## Mocking

The testing framework uses the following approaches for mocking:

1. **Interface Mocking**: Using the `testify/mock` package to create mock implementations of interfaces.
2. **Generated Mocks**: Using `mockgen` to generate mock implementations of interfaces.
3. **Custom Mocks**: Creating custom mock implementations for specific testing scenarios.

### Example: Mocking the HybridPipe Interface

```go
// MockHybridPipe is a mock implementation of the HybridPipe interface.
type MockHybridPipe struct {
    mock.Mock
}

// Connect implements the HybridPipe interface.
func (m *MockHybridPipe) Connect() error {
    args := m.Called()
    return args.Error(0)
}

// Close implements the HybridPipe interface.
func (m *MockHybridPipe) Close() {
    m.Called()
}

// Dispatch implements the HybridPipe interface.
func (m *MockHybridPipe) Dispatch(pipe string, data interface{}) error {
    args := m.Called(pipe, data)
    return args.Error(0)
}

// Subscribe implements the HybridPipe interface.
func (m *MockHybridPipe) Subscribe(pipe string, callback core.Process) error {
    args := m.Called(pipe, callback)
    return args.Error(0)
}

// Unsubscribe implements the HybridPipe interface.
func (m *MockHybridPipe) Unsubscribe(pipe string) error {
    args := m.Called(pipe)
    return args.Error(0)
}
```

## Test Utilities

The `testutils` package provides utilities for testing, including:

- `AssertEventually`: Repeatedly calls a check function until it returns true or the timeout is reached.
- `WithTimeout`: Runs a function with a timeout.
- `TempDir`: Creates a temporary directory for testing.
- `LoadTestData`: Loads test data from the testdata directory.
- `JSONEqual`: Asserts that two JSON strings are equal, ignoring formatting differences.
- `CaptureOutput`: Captures stdout and stderr during the execution of a function.

## Best Practices

1. **Test Coverage**: Aim for at least 85% test coverage for each package.
2. **Test Independence**: Each test should be independent and not rely on the state of other tests.
3. **Test Readability**: Tests should be readable and easy to understand.
4. **Test Maintainability**: Tests should be easy to maintain and update.
5. **Test Performance**: Tests should be fast and efficient.
6. **Test Reliability**: Tests should be reliable and not flaky.
7. **Test Documentation**: Tests should be well-documented.
8. **Test Edge Cases**: Tests should cover edge cases and error conditions.
9. **Test Concurrency**: Tests should handle concurrency correctly.
10. **Test Cleanup**: Tests should clean up after themselves.

## Linting

The repository uses `golangci-lint` for linting. To run the linter, use the following command:

```bash
golangci-lint run
```

Or use the Makefile target:

```bash
make lint
```

The linting configuration is defined in the `.golangci.yml` file.

## Continuous Integration

The repository uses GitHub Actions for continuous integration. The CI pipeline runs tests, linting, and coverage reporting for each pull request and push to the main branch.

## Adding New Tests

When adding new tests, follow these guidelines:

1. Create a test file with the `_test.go` suffix in the same package as the code being tested.
2. Use the `testing` package and the `testify` library for assertions.
3. Use the `testutils` package for common testing tasks.
4. Use mocking for external dependencies.
5. Aim for at least 85% test coverage.
6. Follow the naming conventions for test functions and helpers.
7. Document the tests with comments.
8. Run the tests locally before submitting a pull request.

## Example Test

```go
func TestDispatch(t *testing.T) {
    // Create a mock router
    mockRouter := &MockHybridPipe{}
    
    // Set up expectations
    mockRouter.On("Dispatch", "test-pipe", mock.Anything).Return(nil)
    
    // Create a dispatcher
    dispatcher := NewDispatcher(mockRouter)
    
    // Test data
    testData := map[string]interface{}{
        "message": "Hello, World!",
    }
    
    // Dispatch a message
    err := dispatcher.Dispatch("test-pipe", testData)
    
    // Assert that there was no error
    assert.NoError(t, err)
    
    // Assert that the expectations were met
    mockRouter.AssertExpectations(t)
}
```
