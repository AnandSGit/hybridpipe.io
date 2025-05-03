// Package hybridpipe provides a unified messaging interface for various message brokers.
// It enables communication between microservices or individual processes via
// different messaging systems like Kafka, NATS, RabbitMQ, ZeroMQ, AMQP 1.0, and MQTT.
package hybridpipe

import (
	"encoding/gob"
	"log"

	// Import all protocol implementations
	"hybridpipe.io/core"
	_ "hybridpipe.io/protocols/amqp"
	_ "hybridpipe.io/protocols/kafka"
	_ "hybridpipe.io/protocols/mock"
	_ "hybridpipe.io/protocols/mqtt"
	_ "hybridpipe.io/protocols/nats"
	_ "hybridpipe.io/protocols/netchan"
	_ "hybridpipe.io/protocols/nsq"
	_ "hybridpipe.io/protocols/qpid"
	_ "hybridpipe.io/protocols/rabbitmq"
	_ "hybridpipe.io/protocols/redis"
	_ "hybridpipe.io/protocols/tcp"
)

// Broker types - Users can use these constants to select their communication medium
// when calling the DeployRouter function.
const (
	// NATS - https://nats.io/
	NATS = core.NATS
	// KAFKA - https://kafka.apache.org/
	KAFKA = core.KAFKA
	// RABBITMQ - https://www.rabbitmq.com/
	RABBITMQ = core.RABBITMQ
	// Reserved for future use
	RESERVED1 = core.RESERVED1
	// AMQP1 - https://www.amqp.org
	AMQP1 = core.AMQP1
	// MQTT - http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html
	MQTT = core.MQTT
	// QPID - https://qpid.apache.org/
	QPID = core.QPID
	// NSQ - https://nsq.io/
	NSQ = core.NSQ
	// TCP - Direct TCP/IP communication
	TCP = core.TCP
	// REDIS - https://redis.io/
	REDIS = core.REDIS
	// NETCHAN - Go channels over network
	NETCHAN = core.NETCHAN
	// MOCK - In-memory mock implementation for testing
	MOCK = core.MOCK
)

// Initialize the system
func init() {
	// Initialize configuration
	if err := core.ReadConfig(); err != nil {
		log.Printf("Warning: Failed to read configuration: %v", err)
	}
}

// HybridPipe defines the interface for all messaging implementations.
// Any messaging system implementation must satisfy this interface.
type HybridPipe = core.HybridPipe

// Process defines the callback function that should be called when a client receives a message.
type Process = core.Process

// DeployRouter creates and connects to the specified broker/router for communication.
// It returns a connected HybridPipe interface for the requested messaging system.
func DeployRouter(brokerType int) (HybridPipe, error) {
	return core.DeployRouter(brokerType)
}

// Encode converts user-defined data (to be transmitted) into a byte stream.
// It uses the default serialization format (gob for backward compatibility).
// For more control over serialization, use EncodeWithOptions.
func Encode(data any) ([]byte, error) {
	return core.Encode(data)
}

// Decode converts a received byte stream back into user-defined interface data.
// It automatically detects the serialization format used.
// For more control over deserialization, use DecodeWithOptions.
func Decode(data []byte, target any) error {
	return core.Decode(data, target)
}

// Enable registers a specific user-defined data type to be passed via HybridPipe.
// This must be called before sending custom data types through the messaging system.
// Note: This is only needed for gob encoding, not for other formats.
func Enable(dataType any) {
	gob.Register(dataType)
}

// ProcessWrapper wraps a Process function to handle interface{} to []byte conversion.
// This is needed for backward compatibility with the old protocol implementations.
func ProcessWrapper(fn func(any)) Process {
	return core.ProcessWrapper(fn)
}

// LegacyProcessWrapper wraps a Process function to handle []byte to interface{} conversion.
// This is needed for backward compatibility with the old protocol implementations.
func LegacyProcessWrapper(fn Process) func(any) {
	return core.LegacyProcessWrapper(fn)
}
