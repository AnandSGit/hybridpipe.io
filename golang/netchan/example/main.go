// Example demonstrating the use of the NetChan library.
package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"hybridpipe.io/netchan"
)

func main() {
	// Run the server in a separate goroutine
	go runServer()

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	// Run the client
	runClient()
}

func runServer() {
	// Create a server with default configuration
	server, err := netchan.NewServer(nil)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// Create a channel
	ch := server.Chan("example")

	// Receive messages from the channel
	for i := 0; i < 5; i++ {
		// Receive a message
		value := ch.RecvOp()
		fmt.Printf("Server received: %v\n", value)

		// Send a response
		ch.SendOp(fmt.Sprintf("Response to %v", value))
	}

	fmt.Println("Server done")
}

func runClient() {
	// Create a client with default configuration
	config := netchan.DefaultConfig()
	client, err := netchan.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create a channel
	ch := client.Chan("example")

	// Send and receive messages
	var wg sync.WaitGroup
	wg.Add(2)

	// Send messages
	go func() {
		defer wg.Done()
		for i := 1; i <= 5; i++ {
			// Send a message
			message := fmt.Sprintf("Message %d", i)
			fmt.Printf("Client sending: %s\n", message)
			ch.SendOp(message)

			// Wait a bit
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Receive messages
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			// Receive a message
			value := ch.RecvOp()
			fmt.Printf("Client received: %v\n", value)
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	fmt.Println("Client done")
}
