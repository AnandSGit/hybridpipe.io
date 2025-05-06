# HybridPipe Testing Framework

This document describes how to run tests for the HybridPipe messaging system, including protocol implementations.

## Prerequisites

- Go 1.18 or later
- Docker and Docker Compose
- Make (optional)

## Running Tests

### Running Tests with Docker

The easiest way to run all tests, including those that require external services, is to use Docker Compose. We provide a `docker-compose.test.yml` file that sets up all the necessary services for testing.

```bash
# Start the test infrastructure
docker-compose -f docker-compose.test.yml up -d

# Run the tests
go test -v ./...

# Clean up
docker-compose -f docker-compose.test.yml down
```

Alternatively, you can use the provided scripts:

```bash
# On Linux/macOS
./scripts/run_protocol_tests.sh

# On Windows
powershell -ExecutionPolicy Bypass -File .\scripts\run_protocol_tests.ps1
```

### Running Tests Without External Services

If you don't want to run tests that require external services, you can use the `-short` flag:

```bash
go test -short -v ./...
```

This will skip tests that require external services like MQTT brokers, Kafka, etc.

### Running Tests for a Specific Package

To run tests for a specific package, use:

```bash
go test -v ./path/to/package
```

For example, to run tests for the MQTT protocol:

```bash
go test -v ./protocols/mqtt
```

### Running Tests with Coverage

To run tests with coverage reporting:

```bash
# Run tests with coverage
go test -cover ./...

# Generate a coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Test Infrastructure

The test infrastructure includes the following services:

- MQTT Broker (Mosquitto)
- AMQP Broker (RabbitMQ)
- Kafka and Zookeeper
- Redis
- NSQ
- NATS
- TCP Echo Server
- Qpid Broker

These services are configured in the `docker-compose.test.yml` file.

## Environment Variables

The following environment variables are used by the tests:

- `MQTT_BROKER_ADDRESS`: Address of the MQTT broker (default: `tcp://localhost:1883`)
- `RABBITMQ_SERVER_ADDRESS`: Address of the RabbitMQ server (default: `amqp://guest:guest@localhost:5672/`)
- `KAFKA_BROKER_ADDRESS`: Address of the Kafka broker (default: `localhost:9092`)
- `REDIS_SERVER_ADDRESS`: Address of the Redis server (default: `localhost`)
- `NSQ_NSQD_ADDRESS`: Address of the NSQ daemon (default: `localhost:4150`)
- `NSQ_LOOKUPD_ADDRESS`: Address of the NSQ lookup daemon (default: `localhost:4161`)
- `NATS_SERVER_ADDRESS`: Address of the NATS server (default: `nats://localhost:4222`)
- `TCP_SERVER_ADDRESS`: Address of the TCP echo server (default: `localhost:9000`)
- `QPID_SERVER_ADDRESS`: Address of the Qpid broker (default: `amqp://guest:guest@localhost:5672`)

## Test Coverage

The test coverage for the protocol implementations is as follows:

- Mock: 95%+ (most comprehensive tests)
- Other protocols: Varies depending on external service availability

To achieve 100% test coverage, you need to run the tests with all the required external services.

## Troubleshooting

### Tests Fail with Connection Errors

If tests fail with connection errors, make sure that all the required services are running. You can check the status of the services with:

```bash
docker-compose -f docker-compose.test.yml ps
```

### Tests Hang

If tests hang, it might be because they're waiting for a response from an external service. You can try increasing the timeout or running the tests with the `-timeout` flag:

```bash
go test -timeout 30s ./...
```

### Tests Fail with Permission Errors

If tests fail with permission errors, make sure that you have the necessary permissions to access the external services. You might need to run the tests with elevated privileges or adjust the permissions of the Docker containers.

## Adding Tests for a New Protocol

When adding a new protocol implementation, follow these steps to add tests:

1. Create a new test file named `<protocol>_test.go` in the protocol package
2. Implement tests for all methods of the HybridPipe interface
3. Use the test utilities from the `protocols/testutils` package
4. Test both success and error cases
5. Test the registration of the protocol with the core registry
6. Add any necessary configuration to the Docker Compose file
7. Update the environment variables in the test scripts
