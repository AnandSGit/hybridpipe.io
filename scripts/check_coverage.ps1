# Script to check test coverage against a threshold

# Default threshold
param(
    [int]$Threshold = 80
)

# Create directory for coverage reports
New-Item -Path "coverage" -ItemType Directory -Force | Out-Null

# Run tests with coverage
Write-Host "Running tests with coverage..."
go test -coverprofile=coverage/coverage.out ./...

# Generate coverage report
Write-Host "Generating coverage report..."
$coverageOutput = go tool cover -func=coverage/coverage.out
$totalLine = $coverageOutput | Select-String -Pattern "total:"
$coverage = [double]($totalLine -replace ".*total:\s+\(statements\)\s+(\d+\.\d+)%.*", '$1')

# Compare coverage with threshold
Write-Host "Coverage: $coverage%, Threshold: $Threshold%"
if ($coverage -lt $Threshold) {
    Write-Host "Coverage is below threshold!"
    exit 1
} else {
    Write-Host "Coverage is above threshold!"
    exit 0
}
