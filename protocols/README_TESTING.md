# HybridPipe Protocol Testing Framework

This document describes the testing framework for the HybridPipe protocol implementations.

## Overview

The testing framework for HybridPipe protocol implementations is built using Go's standard testing package, along with additional tools and libraries to enhance testing capabilities. The framework includes:

- Unit tests for each protocol implementation
- Common test utilities for protocol testing
- Test coverage reporting

## Test Structure

### Test Files

Each protocol implementation has a corresponding test file named `<protocol>_test.go` in the same package. For example, the tests for the MQTT protocol are in `protocols/mqtt/mqtt_test.go`.

### Test Utilities

The `protocols/testutils` package provides utilities for testing protocol implementations, including:

- `TestProtocolBasicOperations`: Tests the basic operations of a HybridPipe implementation
- `TestProtocolContextAwareOperations`: Tests the context-aware operations of a HybridPipe implementation
- `TestProtocolErrorHandling`: Tests the error handling of a HybridPipe implementation
- `TestProtocolConcurrentOperations`: Tests the concurrent operations of a HybridPipe implementation

### Test Categories

Each protocol test file includes the following test categories:

1. **Interface Tests**: Tests that the protocol implementation satisfies the HybridPipe interface
2. **Basic Operation Tests**: Tests the basic operations of the protocol implementation
3. **Context-Aware Operation Tests**: Tests the context-aware operations of the protocol implementation
4. **Error Handling Tests**: Tests the error handling of the protocol implementation
5. **Concurrent Operation Tests**: Tests the concurrent operations of the protocol implementation
6. **Method-Specific Tests**: Tests specific methods of the protocol implementation
7. **End-to-End Tests**: Tests the end-to-end functionality of the protocol implementation
8. **Registration Tests**: Tests the registration of the protocol implementation with the core registry

## Running Tests

### Running All Protocol Tests

To run all tests for the protocol implementations, use the following command:

```bash
go test ./protocols/...
```

### Running Tests for a Specific Protocol

To run tests for a specific protocol implementation, use the following command:

```bash
go test ./protocols/<protocol>/
```

For example, to run tests for the MQTT protocol:

```bash
go test ./protocols/mqtt/
```

### Running Tests with Short Mode

To run tests in short mode (skipping tests that require external servers), use the following command:

```bash
go test -short ./protocols/...
```

### Running Tests with Verbose Output

To run tests with verbose output, use the following command:

```bash
go test -v ./protocols/...
```

## Test Coverage

### Generating Coverage Reports

To generate a test coverage report for the protocol implementations, use the provided scripts:

- For Windows: `scripts/test_protocols.ps1`
- For Linux/macOS: `scripts/test_protocols.sh`

These scripts will:

1. Run tests for the protocol implementations with coverage
2. Generate a coverage report in the terminal
3. Generate an HTML coverage report
4. Open the HTML coverage report in the default browser

Alternatively, you can manually generate a coverage report using the following commands:

```bash
go test -coverprofile=coverage/protocols.out ./protocols/...
go tool cover -func=coverage/protocols.out
go tool cover -html=coverage/protocols.out -o coverage/protocols.html
```

## Adding Tests for a New Protocol

When adding a new protocol implementation, follow these steps to add tests:

1. Create a new test file named `<protocol>_test.go` in the protocol package
2. Implement tests for all methods of the HybridPipe interface
3. Use the test utilities from the `protocols/testutils` package
4. Test both success and error cases
5. Test the registration of the protocol with the core registry

## Best Practices

1. **Test Independence**: Each test should be independent and not rely on the state of other tests
2. **Test Readability**: Tests should be readable and easy to understand
3. **Test Maintainability**: Tests should be easy to maintain and update
4. **Test Performance**: Tests should be fast and efficient
5. **Test Reliability**: Tests should be reliable and not flaky
6. **Test Documentation**: Tests should be well-documented
7. **Test Edge Cases**: Tests should cover edge cases and error conditions
8. **Test Concurrency**: Tests should handle concurrency correctly
9. **Test Cleanup**: Tests should clean up after themselves
10. **Test Coverage**: Aim for 100% test coverage for each protocol implementation
