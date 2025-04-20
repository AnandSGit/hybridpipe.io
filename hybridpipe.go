// Package hybridpipe provides a unified messaging interface for various message brokers.
// It enables communication between microservices or individual processes via
// different messaging systems like Kafka, NATS, RabbitMQ, ZeroMQ, AMQP 1.0, and MQTT.
package hybridpipe

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"reflect"
)

// Broker types - Users can use these constants to select their communication medium
// when calling the DeployRouter function.
const (
	// NATS - https://nats.io/
	NATS = iota
	// KAFKA - https://kafka.apache.org/
	KAFKA
	// RABBITMQ - https://www.rabbitmq.com/
	RABBITMQ
	// ZEROMQ - https://zeromq.org/
	ZEROMQ
	// AMQP1 - https://www.amqp.org
	AMQP1
	// MQTT - http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html
	MQTT
	// QPID - https://qpid.apache.org/
	QPID
	// NSQ - https://nsq.io/
	NSQ
	// TCP - Direct TCP/IP communication
	TCP
	// REDIS - https://redis.io/
	REDIS
	// MOCK - In-memory mock implementation for testing
	MOCK
)

// RoutersMap defines Broker or Router Registry for Pipe creation.
// It maps broker types to their corresponding implementation types.
var RoutersMap = make(map[int]reflect.Type)

// Initialize the Register Map with all supported interfaces
func init() {
	// Initialize configuration
	if err := ReadConfig(); err != nil {
		log.Printf("Warning: Failed to read configuration: %v", err)
	}

	// Register all supported messaging implementations
	RoutersMap[NATS] = reflect.TypeOf((*NatsPacket)(nil)).Elem()
	RoutersMap[KAFKA] = reflect.TypeOf((*KafkaPacket)(nil)).Elem()
	RoutersMap[RABBITMQ] = reflect.TypeOf((*RabbitPacket)(nil)).Elem()
	RoutersMap[ZEROMQ] = reflect.TypeOf((*ZMQPacket)(nil)).Elem()
	RoutersMap[AMQP1] = reflect.TypeOf((*AMQPPacket)(nil)).Elem()
	RoutersMap[MQTT] = reflect.TypeOf((*MQTTPacket)(nil)).Elem()
	RoutersMap[QPID] = reflect.TypeOf((*QpidPacket)(nil)).Elem()
	RoutersMap[NSQ] = reflect.TypeOf((*NSQPacket)(nil)).Elem()
	RoutersMap[TCP] = reflect.TypeOf((*TCPPacket)(nil)).Elem()
	RoutersMap[REDIS] = reflect.TypeOf((*RedisPacket)(nil)).Elem()
	RoutersMap[MOCK] = reflect.TypeOf((*MockPacket)(nil)).Elem()
}

// HybridPipe defines the interface for all messaging implementations.
// Any messaging system implementation must satisfy this interface.
type HybridPipe interface {
	// Connect establishes a connection to the messaging system.
	Connect() error

	// Dispatch sends a message to the specified pipe.
	Dispatch(pipe string, data interface{}) error

	// Accept subscribes to messages from the specified pipe and processes them with the provided function.
	Accept(pipe string, fn Process) error

	// Remove unsubscribes from the specified pipe.
	Remove(pipe string) error

	// Close terminates the connection to the messaging system.
	Close()
}

// Process defines the callback function that should be called when a client receives a message.
type Process = func(data interface{})

// DeployRouter creates and connects to the specified broker/router for communication.
// It returns a connected HybridPipe interface for the requested messaging system.
func DeployRouter(brokerType int) (HybridPipe, error) {
	// Validate broker type
	if _, exists := RoutersMap[brokerType]; !exists {
		return nil, fmt.Errorf("unsupported broker type: %d", brokerType)
	}

	// Special handling for MOCK broker type
	if brokerType == MOCK {
		// Create a new MockPacket with a default name
		mock := NewMockPacket("default")

		// Connect to the mock messaging system
		if err := mock.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to mock messaging system: %w", err)
		}

		return mock, nil
	}

	// Create a new instance of the requested broker implementation
	pipe := reflect.New(RoutersMap[brokerType]).Interface().(HybridPipe)

	// Connect to the messaging system
	if err := pipe.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to messaging system: %w", err)
	}

	return pipe, nil
}

// Encode converts user-defined data (to be transmitted) into a byte stream.
// It uses Go's gob encoding for serialization.
func Encode(data interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	if err := encoder.Encode(&data); err != nil {
		return nil, fmt.Errorf("encoding error: %w", err)
	}

	return buffer.Bytes(), nil
}

// Decode converts a received byte stream back into user-defined interface data.
// It uses Go's gob decoding for deserialization.
func Decode(data []byte, target interface{}) error {
	reader := bytes.NewReader(data)
	decoder := gob.NewDecoder(reader)

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decoding error: %w", err)
	}

	return nil
}

// Enable registers a specific user-defined data type to be passed via HybridPipe.
// This must be called before sending custom data types through the messaging system.
func Enable(dataType interface{}) {
	gob.Register(dataType)
}
