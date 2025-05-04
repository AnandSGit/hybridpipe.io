.PHONY: test test-short test-verbose test-race coverage coverage-html lint vet fmt clean

# Default target
all: lint test

# Run all tests
test:
	@echo "Running all tests..."
	@go test ./...

# Run tests with -short flag (skips long-running tests)
test-short:
	@echo "Running short tests..."
	@go test -short ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v ./...

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	@go test -race ./...

# Generate test coverage report
coverage:
	@echo "Generating test coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

# Generate HTML coverage report and open it in the default browser
coverage-html: coverage
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Opening coverage report in browser..."
	@start coverage.html

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -f coverage.out coverage.html
	@go clean

# Install test dependencies
install-test-deps:
	@echo "Installing test dependencies..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go get github.com/stretchr/testify/assert
	@go get github.com/stretchr/testify/mock
	@go get github.com/stretchr/testify/require
	@go get github.com/golang/mock/mockgen
	@go mod tidy

# Generate mocks for testing
generate-mocks:
	@echo "Generating mocks..."
	@go generate ./...
