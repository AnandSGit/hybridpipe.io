package main

import (
	"fmt"
	"time"
)

// This is a simplified example that demonstrates the concept of NetChan
// without requiring the full hybridpipe.io package

// SimpleChannel represents a channel that can be used to send and receive values
type SimpleChannel struct {
	sendChan chan interface{}
	recvChan chan interface{}
}

// NewSimpleChannel creates a new SimpleChannel
func NewSimpleChannel(bufferSize int) *SimpleChannel {
	return &SimpleChannel{
		sendChan: make(chan interface{}, bufferSize),
		recvChan: make(chan interface{}, bufferSize),
	}
}

// SendChan returns the send channel
func (c *SimpleChannel) SendChan() chan<- interface{} {
	return c.sendChan
}

// RecvChan returns the receive channel
func (c *SimpleChannel) RecvChan() <-chan interface{} {
	return c.recvChan
}

// Start starts forwarding messages between the send and receive channels
func (c *SimpleChannel) Start() {
	go func() {
		for msg := range c.sendChan {
			c.recvChan <- msg
		}
		close(c.recvChan)
	}()
}

// Close closes the channels
func (c *SimpleChannel) Close() {
	close(c.sendChan)
}

func main() {
	fmt.Println("NetChan concept demonstration")

	// Create a server in a separate goroutine
	go runServer()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Run client
	runClient()
}

func runServer() {
	// Create a channel
	channel := NewSimpleChannel(10)
	channel.Start()
	defer channel.Close()

	fmt.Println("Server: Ready to receive messages")

	// Use Go channel syntax to receive messages
	for {
		// Use native channel syntax to receive
		msg, ok := <-channel.RecvChan()
		if !ok {
			fmt.Println("Server: Channel closed")
			break
		}

		fmt.Printf("Server received: %v\n", msg)

		// Send a response using the send channel
		channel.SendChan() <- fmt.Sprintf("Echo: %v", msg)
	}
}

func runClient() {
	// Create a channel
	channel := NewSimpleChannel(10)
	channel.Start()
	defer channel.Close()

	// Send messages using Go channel syntax
	messages := []string{
		"Hello, NetChan!",
		"This is using Go channel syntax",
		"With a simplified implementation",
		"Isn't that cool?",
	}

	for _, msg := range messages {
		// Send using native channel syntax
		channel.SendChan() <- msg
		fmt.Printf("Client sent: %s\n", msg)

		// Receive response using native channel syntax
		response, ok := <-channel.RecvChan()
		if !ok {
			fmt.Println("Client: Channel closed")
			break
		}

		fmt.Printf("Client received: %v\n", response)
		time.Sleep(500 * time.Millisecond)
	}
}
