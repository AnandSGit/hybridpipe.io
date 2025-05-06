.PHONY: all build test clean examples

# Default target
all: build test examples

# Build the library
build:
	@echo "Building NetChan library..."
	@go build -v ./...

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run short tests only
test-short:
	@echo "Running short tests..."
	@go test -v -short ./...

# Build examples
examples:
	@echo "Building examples..."
	@go build -v ./examples/simple
	@go build -v ./examples/select

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f examples/simple/simple
	@rm -f examples/select/select
	@go clean

# Run the simple example receiver
run-receiver:
	@echo "Running simple receiver..."
	@go run examples/simple/main.go -mode receiver

# Run the simple example sender
run-sender:
	@echo "Running simple sender..."
	@go run examples/simple/main.go -mode sender

# Run the hub example
run-hub:
	@echo "Running hub..."
	@go run examples/select/main.go -mode hub

# Run the producer example
run-producer:
	@echo "Running producer..."
	@go run examples/select/main.go -mode producer

# Run the consumer example
run-consumer:
	@echo "Running consumer..."
	@go run examples/select/main.go -mode consumer

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Build, test, and build examples"
	@echo "  build        - Build the library"
	@echo "  test         - Run all tests"
	@echo "  test-short   - Run short tests only"
	@echo "  examples     - Build examples"
	@echo "  clean        - Clean build artifacts"
	@echo "  run-receiver - Run the simple example receiver"
	@echo "  run-sender   - Run the simple example sender"
	@echo "  run-hub      - Run the hub example"
	@echo "  run-producer - Run the producer example"
	@echo "  run-consumer - Run the consumer example"
	@echo "  help         - Show this help message"
