// Package core provides the core interfaces and functionality for the HybridPipe messaging system.
package core

// AMQPF contains AMQP 1.0 specific configuration
type AMQPF struct {
	// AMQPServer is the AMQP server connection string
	AMQPServer string
	// AMQPPort is the AMQP server port
	AMQPPort string
}
