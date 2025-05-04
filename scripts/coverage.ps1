# Script to generate test coverage reports for the hybridpipe.io repository

# Change to the root directory of the repository
Set-Location -Path (Split-Path -Parent $PSScriptRoot)

# Create a directory for coverage reports
New-Item -Path "coverage" -ItemType Directory -Force | Out-Null

# Run tests with coverage
go test -coverprofile=coverage/coverage.out ./...

# Generate coverage report in terminal
go tool cover -func=coverage/coverage.out

# Generate HTML coverage report
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

Write-Host "Coverage report generated in coverage/coverage.html"

# Open the coverage report in the default browser
Start-Process "coverage/coverage.html"
