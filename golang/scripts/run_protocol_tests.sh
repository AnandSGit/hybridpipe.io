#!/bin/bash

# Exit on error
set -e

# Function to clean up on exit
cleanup() {
  echo "Cleaning up..."
  docker-compose -f docker-compose.test.yml down
}

# Register the cleanup function to be called on exit
trap cleanup EXIT

# Create directory for test results
mkdir -p test-results
mkdir -p coverage

# Start the test infrastructure
echo "Starting test infrastructure..."
docker-compose -f docker-compose.test.yml up -d

# Wait for all services to be healthy
echo "Waiting for services to be ready..."
docker-compose -f docker-compose.test.yml ps | grep -v "healthy" | grep -v "NAME" | awk '{print $1}' | while read -r container; do
  echo "Waiting for $container..."
  while [ "$(docker inspect --format='{{.State.Health.Status}}' "$container")" != "healthy" ]; do
    sleep 2
  done
  echo "$container is ready"
done

echo "All services are ready"

# Set environment variables for tests
export MQTT_BROKER_ADDRESS="tcp://localhost:1883"
export RABBITMQ_SERVER_ADDRESS="amqp://guest:guest@localhost:5672/"
export KAFKA_BROKER_ADDRESS="localhost:9092"
export REDIS_SERVER_ADDRESS="localhost"
export NSQ_NSQD_ADDRESS="localhost:4150"
export NSQ_LOOKUPD_ADDRESS="localhost:4161"
export NATS_SERVER_ADDRESS="nats://localhost:4222"
export TCP_SERVER_ADDRESS="localhost:9000"
export QPID_SERVER_ADDRESS="amqp://guest:guest@localhost:5672"

# Run tests with coverage
echo "Running tests with coverage..."
go test -coverprofile=coverage/protocols.out ./protocols/...

# Generate coverage report
echo "Generating coverage report..."
go tool cover -func=coverage/protocols.out
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
