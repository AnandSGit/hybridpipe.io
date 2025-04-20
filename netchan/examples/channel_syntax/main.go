package main

import (
	"fmt"
	"log"
	"time"

	"github.com/AnandSGit/hybridpipe.io/netchan"
)

func main() {
	// Example of using NetChan with Go channel syntax
	fmt.Println("NetChan with Go channel syntax example")
	
	// Create a server in a separate goroutine
	go runServer()
	
	// Wait for server to start
	time.Sleep(500 * time.Millisecond)
	
	// Run client
	runClient()
}

func runServer() {
	// Create a NetChan listener
	netChan, err := netchan.Listen[string](":8080", "")
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	
	// Create a channel wrapper
	wrapper := netchan.NewChannelWrapper(netChan, 10)
	if err := wrapper.Initialize(); err != nil {
		log.Fatalf("Failed to initialize wrapper: %v", err)
	}
	
	fmt.Println("Server: Listening on :8080")
	
	// Use Go channel syntax to receive messages
	for {
		// Get the receive channel
		recvChan := wrapper.RecvChan()
		
		// Use native channel syntax to receive
		msg, ok := <-recvChan
		if !ok {
			fmt.Println("Server: Channel closed")
			break
		}
		
		fmt.Printf("Server received: %s\n", msg)
		
		// Send a response using the send channel
		wrapper.SendChan() <- fmt.Sprintf("Echo: %s", msg)
	}
}

func runClient() {
	// Connect to the server
	netChan, err := netchan.Dial[string]("localhost:8080", "")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	
	// Create a channel wrapper
	wrapper := netchan.NewChannelWrapper(netChan, 10)
	if err := wrapper.Initialize(); err != nil {
		log.Fatalf("Failed to initialize wrapper: %v", err)
	}
	
	// Get the send and receive channels
	sendChan := wrapper.SendChan()
	recvChan := wrapper.RecvChan()
	
	// Send messages using Go channel syntax
	messages := []string{
		"Hello, NetChan!",
		"This is using Go channel syntax",
		"Over the network!",
		"Isn't that cool?",
	}
	
	for _, msg := range messages {
		// Send using native channel syntax
		sendChan <- msg
		fmt.Printf("Client sent: %s\n", msg)
		
		// Receive response using native channel syntax
		response, ok := <-recvChan
		if !ok {
			fmt.Println("Client: Channel closed")
			break
		}
		
		fmt.Printf("Client received: %s\n", response)
		time.Sleep(500 * time.Millisecond)
	}
	
	// Close the wrapper
	if err := wrapper.Close(); err != nil {
		log.Printf("Error closing wrapper: %v", err)
	}
}
