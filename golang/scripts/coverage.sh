#!/bin/bash

# Script to generate test coverage reports for the hybridpipe.io repository

# Exit on error
set -e

# Change to the root directory of the repository
cd "$(dirname "$0")/.."

# Create a directory for coverage reports
mkdir -p coverage

# Run tests with coverage
go test -coverprofile=coverage/coverage.out ./...

# Generate coverage report in terminal
go tool cover -func=coverage/coverage.out

# Generate HTML coverage report
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

echo "Coverage report generated in coverage/coverage.html"
