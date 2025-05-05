#!/bin/bash

# Script to run tests for the protocols folder with coverage

# Exit on error
set -e

# Change to the root directory of the repository
cd "$(dirname "$0")/.."

# Create a directory for coverage reports
mkdir -p coverage

# Run tests for the protocols folder with coverage
echo "Running tests for the protocols folder with coverage..."
go test -coverprofile=coverage/protocols.out ./protocols/...

# Generate coverage report in terminal
echo "Coverage report for protocols folder:"
go tool cover -func=coverage/protocols.out

# Generate HTML coverage report
go tool cover -html=coverage/protocols.out -o coverage/protocols.html

echo "Coverage report generated in coverage/protocols.html"

# Open the coverage report in the default browser
if [ "$(uname)" = "Darwin" ]; then
    open coverage/protocols.html
elif [ "$(uname)" = "Linux" ]; then
    xdg-open coverage/protocols.html 2>/dev/null || echo "Coverage report generated at coverage/protocols.html"
else
    echo "Coverage report generated at coverage/protocols.html"
fi
