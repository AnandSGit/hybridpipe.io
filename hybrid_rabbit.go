package hybridpipe

import (
	"fmt"
	"log"
	"sync"

	amqp "github.com/streadway/amqp"
)

// RabbitPacket implements the HybridPipe interface for RabbitMQ messaging system.
type RabbitPacket struct {
	// HandleConn is the connection to the RabbitMQ server
	HandleConn *amqp.Connection
	// RChannel is the AMQP channel for communication
	RChannel *amqp.Channel
	// consumers maps pipe names to their consumer cancel channels
	consumers map[string]chan bool
	// mutex protects concurrent access to the consumers map
	mutex sync.RWMutex
}

// Connect establishes a connection to the RabbitMQ server using the configuration in HPipeConfig.
func (rp *RabbitPacket) Connect() error {
	// Connect to RabbitMQ server
	conn, err := amqp.Dial(HPipeConfig.RServerPort)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ server: %w", err)
	}

	// Create a channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	// Initialize the connection and consumers map
	rp.HandleConn = conn
	rp.RChannel = channel
	rp.consumers = make(map[string]chan bool)

	log.Printf("Connected to RabbitMQ server at %s", HPipeConfig.RServerPort)
	return nil
}

// Dispatch publishes a message to the specified pipe.
func (rp *RabbitPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection
	if rp.HandleConn == nil || rp.RChannel == nil {
		return fmt.Errorf("not connected to RabbitMQ server")
	}

	// Declare a queue for the pipe
	queue, err := rp.RChannel.QueueDeclare(
		pipe,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", pipe, err)
	}

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message
	err = rp.RChannel.Publish(
		"",         // exchange
		queue.Name, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        bytes,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (rp *RabbitPacket) Accept(pipe string, fn Process) error {
	// Validate connection
	if rp.HandleConn == nil || rp.RChannel == nil {
		return fmt.Errorf("not connected to RabbitMQ server")
	}

	// Declare a queue for the pipe
	queue, err := rp.RChannel.QueueDeclare(
		pipe,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", pipe, err)
	}

	// Start consuming messages
	messages, err := rp.RChannel.Consume(
		queue.Name, // queue
		pipe,       // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer for pipe %s: %w", pipe, err)
	}

	// Create a channel to signal the consumer to stop
	rp.mutex.Lock()
	// Cancel any existing consumer for this pipe
	if done, exists := rp.consumers[pipe]; exists {
		close(done)
	}
	done := make(chan bool)
	rp.consumers[pipe] = done
	rp.mutex.Unlock()

	// Start a goroutine to handle messages
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in RabbitMQ consumer for pipe %s: %v", pipe, r)
			}
		}()

		for {
			select {
			case <-done:
				log.Printf("Stopping RabbitMQ consumer for pipe %s", pipe)
				return
			case msg, ok := <-messages:
				if !ok {
					log.Printf("RabbitMQ channel closed for pipe %s", pipe)
					return
				}

				// Decode the message
				var data interface{}
				if err := Decode(msg.Body, &data); err != nil {
					log.Printf("Failed to decode message from pipe %s: %v", pipe, err)
					continue
				}

				// Process the message
				fn(data)
			}
		}
	}()

	log.Printf("Subscribed to RabbitMQ pipe %s", pipe)
	return nil
}

// Remove unsubscribes from the specified pipe.
func (rp *RabbitPacket) Remove(pipe string) error {
	// Validate connection
	if rp.HandleConn == nil || rp.RChannel == nil {
		return fmt.Errorf("not connected to RabbitMQ server")
	}

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Check if the pipe is subscribed
	done, exists := rp.consumers[pipe]
	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Cancel the consumer
	err := rp.RChannel.Cancel(pipe, false)
	if err != nil {
		return fmt.Errorf("failed to cancel consumer for pipe %s: %w", pipe, err)
	}

	// Signal the consumer goroutine to stop
	close(done)
	delete(rp.consumers, pipe)

	log.Printf("Unsubscribed from RabbitMQ pipe %s", pipe)
	return nil
}

// Close terminates the connection to the RabbitMQ server.
func (rp *RabbitPacket) Close() {
	// Check if connected
	if rp.HandleConn == nil {
		return
	}

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Cancel all consumers
	for pipe, done := range rp.consumers {
		// Signal the consumer goroutine to stop
		close(done)
		delete(rp.consumers, pipe)

		// Try to cancel the consumer
		if rp.RChannel != nil {
			if err := rp.RChannel.Cancel(pipe, false); err != nil {
				log.Printf("Error canceling consumer for pipe %s: %v", pipe, err)
			}
		}
	}

	// Close the channel
	if rp.RChannel != nil {
		if err := rp.RChannel.Close(); err != nil {
			log.Printf("Error closing RabbitMQ channel: %v", err)
		}
		rp.RChannel = nil
	}

	// Close the connection
	if err := rp.HandleConn.Close(); err != nil {
		log.Printf("Error closing RabbitMQ connection: %v", err)
	}
	rp.HandleConn = nil

	log.Printf("Closed RabbitMQ connection")
}
