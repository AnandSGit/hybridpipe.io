package examples

import (
	"fmt"
	"log"
	"time"

	"hybridpipe.io/core"
	"hybridpipe.io/protocols/netchan"
)

// NetChanExample demonstrates the use of the NetChan protocol.
func NetChanExample() {
	fmt.Println("Starting NetChan example...")

	// Create a new NetChan router
	config := &core.NetChanConfig{
		ChannelBufferSize: 10,
		SendTimeout:       1000,
	}
	router := netchan.New(config)

	// Connect to the NetChan system
	if err := router.Connect(); err != nil {
		log.Fatalf("Failed to connect to NetChan system: %v", err)
	}
	defer router.Close()

	// Create a channel to receive messages
	receivedCh := make(chan interface{}, 10)

	// Subscribe to a pipe
	err := router.Subscribe("example-pipe", func(data []byte) error {
		// Decode the message
		var msg interface{}
		if err := core.Decode(data, &msg); err != nil {
			return fmt.Errorf("failed to decode message: %w", err)
		}

		// Send the message to the channel
		receivedCh <- msg
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to pipe: %v", err)
	}

	// Start a goroutine to receive messages
	go func() {
		for msg := range receivedCh {
			fmt.Printf("Received: %v\n", msg)
		}
	}()

	// Send messages
	messages := []string{
		"Hello, NetChan!",
		"This is using HybridPipe",
		"With NetChan protocol",
		"Isn't that cool?",
	}

	for _, msg := range messages {
		// Send the message
		err := router.Dispatch("example-pipe", msg)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			continue
		}
		fmt.Printf("Sent: %s\n", msg)
		time.Sleep(500 * time.Millisecond)
	}

	// Wait a bit to ensure all messages are processed
	time.Sleep(1 * time.Second)

	// Print statistics
	fmt.Printf("\nStatistics:\n")
	fmt.Printf("Messages sent and received through NetChan\n")

	fmt.Println("NetChan example completed successfully")
}
