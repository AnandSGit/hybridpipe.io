package main

import (
	"fmt"
	"log"
	"time"

	hp "github.com/AnandSGit/hybridpipe.io"
	"github.com/AnandSGit/hybridpipe.io/netchan"
)

// NetChanAdapter provides a bridge between HybridPipe and the channel wrapper
type NetChanAdapter struct {
	router     hp.HybridPipe
	wrappers   map[string]*netchan.ChannelWrapper[interface{}]
	bufferSize int
}

// NewNetChanAdapter creates a new adapter
func NewNetChanAdapter(bufferSize int) (*NetChanAdapter, error) {
	// Create a NetChan router through HybridPipe
	router, err := hp.DeployRouter(hp.NETCHAN)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy NetChan router: %v", err)
	}

	return &NetChanAdapter{
		router:     router,
		wrappers:   make(map[string]*netchan.ChannelWrapper[interface{}]),
		bufferSize: bufferSize,
	}, nil
}

// GetSendChan returns a channel that can be used to send values to the specified pipe
func (a *NetChanAdapter) GetSendChan(pipe string) (chan<- interface{}, error) {
	// Check if we already have a wrapper for this pipe
	if wrapper, exists := a.wrappers[pipe]; exists {
		return wrapper.SendChan(), nil
	}

	// Create a new NetChan
	netChan, err := netchan.Dial[interface{}](fmt.Sprintf("localhost:%d", hp.HPipeConfig.NetChanF.NetChanPort), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NetChan: %v", err)
	}

	// Create a wrapper
	wrapper := netchan.NewChannelWrapper(netChan, a.bufferSize)
	if err := wrapper.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize wrapper: %v", err)
	}

	// Store the wrapper
	a.wrappers[pipe] = wrapper

	// Set up the HybridPipe Accept handler to forward messages to the wrapper
	err = a.router.Accept(pipe, func(data interface{}) {
		wrapper.SendChan() <- data
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set up Accept handler: %v", err)
	}

	// Start a goroutine to forward messages from the wrapper to HybridPipe
	go func() {
		recvChan := wrapper.RecvChan()
		for msg := range recvChan {
			if err := a.router.Dispatch(pipe, msg); err != nil {
				log.Printf("Error dispatching message: %v", err)
			}
		}
	}()

	return wrapper.SendChan(), nil
}

// GetRecvChan returns a channel that can be used to receive values from the specified pipe
func (a *NetChanAdapter) GetRecvChan(pipe string) (<-chan interface{}, error) {
	// Check if we already have a wrapper for this pipe
	if wrapper, exists := a.wrappers[pipe]; exists {
		return wrapper.RecvChan(), nil
	}

	// Create a new NetChan
	netChan, err := netchan.Listen[interface{}](fmt.Sprintf(":%d", hp.HPipeConfig.NetChanF.NetChanPort), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NetChan: %v", err)
	}

	// Create a wrapper
	wrapper := netchan.NewChannelWrapper(netChan, a.bufferSize)
	if err := wrapper.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize wrapper: %v", err)
	}

	// Store the wrapper
	a.wrappers[pipe] = wrapper

	// Set up the HybridPipe Accept handler to forward messages to the wrapper
	err = a.router.Accept(pipe, func(data interface{}) {
		wrapper.SendChan() <- data
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set up Accept handler: %v", err)
	}

	// Start a goroutine to forward messages from the wrapper to HybridPipe
	go func() {
		recvChan := wrapper.RecvChan()
		for msg := range recvChan {
			if err := a.router.Dispatch(pipe, msg); err != nil {
				log.Printf("Error dispatching message: %v", err)
			}
		}
	}()

	return wrapper.RecvChan(), nil
}

// Close closes the adapter and all associated wrappers
func (a *NetChanAdapter) Close() error {
	// Close all wrappers
	for pipe, wrapper := range a.wrappers {
		if err := wrapper.Close(); err != nil {
			log.Printf("Error closing wrapper for pipe %s: %v", pipe, err)
		}
	}

	// Close the router
	a.router.Close()
	return nil
}

func main() {
	// Example of using NetChan with HybridPipe and Go channel syntax
	fmt.Println("NetChan with HybridPipe and Go channel syntax example")

	// Create a server in a separate goroutine
	go runHPServer()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Run client
	runHPClient()
}

func runHPServer() {
	// Create a NetChan adapter
	adapter, err := NewNetChanAdapter(10)
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Close()

	// Get a receive channel for the "test" pipe
	recvChan, err := adapter.GetRecvChan("test")
	if err != nil {
		log.Fatalf("Failed to get receive channel: %v", err)
	}

	// Get a send channel for the "test" pipe
	sendChan, err := adapter.GetSendChan("test")
	if err != nil {
		log.Fatalf("Failed to get send channel: %v", err)
	}

	fmt.Println("Server: Ready to receive messages")

	// Use Go channel syntax to receive messages
	for {
		// Use native channel syntax to receive
		msg, ok := <-recvChan
		if !ok {
			fmt.Println("Server: Channel closed")
			break
		}

		fmt.Printf("Server received: %v\n", msg)

		// Send a response using the send channel
		sendChan <- fmt.Sprintf("Echo: %v", msg)
	}
}

func runHPClient() {
	// Create a NetChan adapter
	adapter, err := NewNetChanAdapter(10)
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Close()

	// Get a send channel for the "test" pipe
	sendChan, err := adapter.GetSendChan("test")
	if err != nil {
		log.Fatalf("Failed to get send channel: %v", err)
	}

	// Get a receive channel for the "test" pipe
	recvChan, err := adapter.GetRecvChan("test")
	if err != nil {
		log.Fatalf("Failed to get receive channel: %v", err)
	}

	// Send messages using Go channel syntax
	messages := []string{
		"Hello, HybridPipe!",
		"This is using Go channel syntax",
		"With HybridPipe integration",
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

		fmt.Printf("Client received: %v\n", response)
		time.Sleep(500 * time.Millisecond)
	}
}
