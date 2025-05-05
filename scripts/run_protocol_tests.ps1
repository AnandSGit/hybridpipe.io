# Script to run protocol tests with infrastructure

# Function to clean up on exit
function Cleanup {
    Write-Host "Cleaning up..."
    docker-compose -f docker-compose.test.yml down
}

# Create directory for test results
New-Item -Path "test-results" -ItemType Directory -Force | Out-Null
New-Item -Path "coverage" -ItemType Directory -Force | Out-Null

try {
    # Start the test infrastructure
    Write-Host "Starting test infrastructure..."
    docker-compose -f docker-compose.test.yml up -d

    # Wait for all services to be healthy
    Write-Host "Waiting for services to be ready..."
    $containers = docker-compose -f docker-compose.test.yml ps --services
    foreach ($container in $containers) {
        Write-Host "Waiting for $container..."
        $status = ""
        while ($status -ne "healthy") {
            $status = docker inspect --format='{{.State.Health.Status}}' $container
            if ($status -ne "healthy") {
                Start-Sleep -Seconds 2
            }
        }
        Write-Host "$container is ready"
    }

    Write-Host "All services are ready"

    # Set environment variables for tests
    $env:MQTT_BROKER_ADDRESS = "tcp://localhost:1883"
    $env:RABBITMQ_SERVER_ADDRESS = "amqp://guest:guest@localhost:5672/"
    $env:KAFKA_BROKER_ADDRESS = "localhost:9092"
    $env:REDIS_SERVER_ADDRESS = "localhost"
    $env:NSQ_NSQD_ADDRESS = "localhost:4150"
    $env:NSQ_LOOKUPD_ADDRESS = "localhost:4161"
    $env:NATS_SERVER_ADDRESS = "nats://localhost:4222"
    $env:TCP_SERVER_ADDRESS = "localhost:9000"
    $env:QPID_SERVER_ADDRESS = "amqp://guest:guest@localhost:5672"

    # Run tests with coverage
    Write-Host "Running tests with coverage..."
    go test -coverprofile=coverage/protocols.out ./protocols/...

    # Generate coverage report
    Write-Host "Generating coverage report..."
    go tool cover -func=coverage/protocols.out
    go tool cover -html=coverage/protocols.out -o coverage/protocols.html

    Write-Host "Coverage report generated in coverage/protocols.html"

    # Open the coverage report in the default browser
    Start-Process "coverage/protocols.html"
}
finally {
    # Clean up
    Cleanup
}
