#!/bin/bash

# Script to check test coverage against a threshold

# Exit on error
set -e

# Default threshold
THRESHOLD=${1:-80}

# Create directory for coverage reports
mkdir -p coverage

# Run tests with coverage
echo "Running tests with coverage..."
go test -coverprofile=coverage/coverage.out ./...

# Generate coverage report
echo "Generating coverage report..."
COVERAGE=$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $3}' | tr -d '%')

# Compare coverage with threshold
echo "Coverage: $COVERAGE%, Threshold: $THRESHOLD%"
if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "Coverage is below threshold!"
    exit 1
else
    echo "Coverage is above threshold!"
    exit 0
fi
