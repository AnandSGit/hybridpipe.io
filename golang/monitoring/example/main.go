// Example demonstrating the use of the monitoring package.
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"hybridpipe.io/core"
	"hybridpipe.io/monitoring"
	"hybridpipe.io/protocols/mock"
)

func main() {
	// Create a message tracer
	tracer := monitoring.NewMessageTracer(1000)

	// Create a mock router
	router := mock.NewMockRouter()

	// Create a tracing middleware
	tracingRouter := monitoring.NewTracingMiddleware(router, tracer, "MOCK")

	// Connect to the router
	if err := tracingRouter.Connect(); err != nil {
		log.Fatalf("Failed to connect to router: %v", err)
	}
	defer tracingRouter.Close()

	// Create metrics
	messagesReceived := monitoring.NewCounter("messages_received")
	messagesSent := monitoring.NewCounter("messages_sent")
	messageSize := monitoring.NewHistogram("message_size", []int64{10, 100, 1000, 10000})
	activeConnections := monitoring.NewGauge("active_connections")

	// Register the metrics
	monitoring.Register(messagesReceived)
	monitoring.Register(messagesSent)
	monitoring.Register(messageSize)
	monitoring.Register(activeConnections)

	// Start the monitoring server
	if err := monitoring.StartDefaultMonitoringServer(":8080"); err != nil {
		log.Fatalf("Failed to start monitoring server: %v", err)
	}
	defer monitoring.StopDefaultMonitoringServer()

	// Subscribe to a pipe
	if err := tracingRouter.Subscribe("test", func(data []byte) error {
		// Increment the messages received counter
		messagesReceived.Increment(1)

		// Observe the message size
		messageSize.Observe(int64(len(data)))

		// Process the message
		var message string
		if err := core.Decode(data, &message); err != nil {
			return fmt.Errorf("failed to decode message: %w", err)
		}

		fmt.Printf("Received message: %s\n", message)
		return nil
	}); err != nil {
		log.Fatalf("Failed to subscribe to pipe: %v", err)
	}

	// Simulate connections
	activeConnections.Set(1)

	// Send messages
	for i := 1; i <= 10; i++ {
		// Create a message
		message := fmt.Sprintf("Message %d", i)

		// Send the message
		if err := tracingRouter.Dispatch("test", message); err != nil {
			log.Printf("Failed to send message: %v", err)
			continue
		}

		// Increment the messages sent counter
		messagesSent.Increment(1)

		// Simulate connection changes
		if i%3 == 0 {
			activeConnections.Increment(1)
		} else if i%5 == 0 {
			activeConnections.Decrement(1)
		}

		// Wait a bit
		time.Sleep(time.Duration(rand.Intn(500)+500) * time.Millisecond)
	}

	// Print the metrics
	fmt.Printf("Messages received: %d\n", messagesReceived.GetValue())
	fmt.Printf("Messages sent: %d\n", messagesSent.GetValue())
	fmt.Printf("Active connections: %d\n", activeConnections.GetValue())

	// Print the histogram buckets
	buckets := messageSize.GetBuckets()
	fmt.Println("Message size histogram:")
	for bucket, count := range buckets {
		fmt.Printf("  <= %d bytes: %d\n", bucket, count)
	}

	// Print the traces
	fmt.Println("Message traces:")
	for _, trace := range tracer.GetTraces() {
		fmt.Println(trace)
	}

	fmt.Println("Monitoring server is running on http://localhost:8080")
	fmt.Println("Press Ctrl+C to exit")

	// Wait for the user to press Ctrl+C
	select {}
}
