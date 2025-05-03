package core

import (
	"fmt"
	"reflect"
)

// Constants for broker types
const (
	NATS      = iota // NATS messaging system
	KAFKA            // Apache Kafka
	RABBITMQ         // RabbitMQ (AMQP 0.9.1)
	RESERVED1        // Reserved for future use
	AMQP1            // AMQP 1.0
	MQTT             // MQTT protocol
	QPID             // Apache Qpid
	NSQ              // NSQ messaging platform
	TCP              // Direct TCP/IP communication
	REDIS            // Redis pub/sub
	NETCHAN          // Go channels over network
	MOCK             // In-memory mock for testing
)

// RoutersMap defines Broker or Router Registry for Pipe creation.
// It maps broker types to their corresponding implementation types.
var RoutersMap = make(map[int]reflect.Type)

// RegisterRouter registers a router implementation with the specified broker type.
func RegisterRouter(brokerType int, routerType reflect.Type) {
	RoutersMap[brokerType] = routerType
}

// DeployRouter creates and connects to the specified broker/router for communication.
// It returns a connected HybridPipe interface for the requested messaging system.
func DeployRouter(brokerType int) (HybridPipe, error) {
	// First try the factory-based approach
	factoryMutex.RLock()
	factory, factoryExists := registeredFactories[brokerType]
	factoryMutex.RUnlock()

	if factoryExists {
		router := factory()
		if err := router.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to messaging system: %w", err)
		}
		return router, nil
	}

	// Fall back to the reflection-based approach
	if routerType, exists := RoutersMap[brokerType]; exists {
		// Create a new instance of the requested broker implementation
		pipe := reflect.New(routerType).Interface().(HybridPipe)

		// Connect to the messaging system
		if err := pipe.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to messaging system: %w", err)
		}

		return pipe, nil
	}

	return nil, fmt.Errorf("unsupported broker type: %d", brokerType)
}
