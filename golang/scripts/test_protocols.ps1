# Script to run tests for the protocols folder with coverage

# Change to the root directory of the repository
Set-Location -Path (Split-Path -Parent $PSScriptRoot)

# Create a directory for coverage reports
New-Item -Path "coverage" -ItemType Directory -Force | Out-Null

# Run tests for the protocols folder with coverage
Write-Host "Running tests for the protocols folder with coverage..."
go test -coverprofile=coverage/protocols.out ./protocols/...

# Generate coverage report in terminal
Write-Host "Coverage report for protocols folder:"
go tool cover -func=coverage/protocols.out

# Generate HTML coverage report
go tool cover -html=coverage/protocols.out -o coverage/protocols.html

Write-Host "Coverage report generated in coverage/protocols.html"

# Open the coverage report in the default browser
Start-Process "coverage/protocols.html"
