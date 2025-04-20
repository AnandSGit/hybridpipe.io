package hybridpipe

import (
	"sync"
	"testing"
	"time"
)

func TestNetChanPacket(t *testing.T) {
	// Create a new NetChanPacket
	ncp := &NetChanPacket{}

	// Connect to the NetChan system
	if err := ncp.Connect(); err != nil {
		t.Fatalf("Failed to connect to NetChan system: %v", err)
	}

	// Set up a wait group to wait for the message to be received
	var wg sync.WaitGroup
	wg.Add(1)

	// Set up a variable to store the received message
	var receivedMessage string

	// Accept messages from the "test" pipe
	if err := ncp.Accept("test", func(data interface{}) {
		// Convert the received data to a string
		if msg, ok := data.(string); ok {
			receivedMessage = msg
			wg.Done()
		} else {
			t.Errorf("Received data is not a string: %v", data)
			wg.Done()
		}
	}); err != nil {
		t.Fatalf("Failed to accept messages from pipe: %v", err)
	}

	// Dispatch a message to the "test" pipe
	if err := ncp.Dispatch("test", "Hello, NetChan!"); err != nil {
		t.Fatalf("Failed to dispatch message: %v", err)
	}

	// Wait for the message to be received with a timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Check if the received message is correct
	if receivedMessage != "Hello, NetChan!" {
		t.Errorf("Received message is incorrect. Expected: %s, Got: %s", "Hello, NetChan!", receivedMessage)
	}

	// Remove the subscription
	if err := ncp.Remove("test"); err != nil {
		t.Fatalf("Failed to remove subscription: %v", err)
	}

	// Close the connection
	ncp.Close()
}

func TestZMQPacket(t *testing.T) {
	// Create a new ZMQPacket
	zp := &ZMQPacket{}

	// Connect to the ZeroMQ system
	if err := zp.Connect(); err != nil {
		t.Fatalf("Failed to connect to ZeroMQ system: %v", err)
	}

	// Set up a wait group to wait for the message to be received
	var wg sync.WaitGroup
	wg.Add(1)

	// Set up a variable to store the received message
	var receivedMessage string

	// Accept messages from the "test" pipe
	if err := zp.Accept("test", func(data interface{}) {
		// Convert the received data to a string
		if msg, ok := data.(string); ok {
			receivedMessage = msg
			wg.Done()
		} else {
			t.Errorf("Received data is not a string: %v", data)
			wg.Done()
		}
	}); err != nil {
		t.Fatalf("Failed to accept messages from pipe: %v", err)
	}

	// Dispatch a message to the "test" pipe
	if err := zp.Dispatch("test", "Hello, ZeroMQ!"); err != nil {
		t.Fatalf("Failed to dispatch message: %v", err)
	}

	// Wait for the message to be received with a timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Check if the received message is correct
	if receivedMessage != "Hello, ZeroMQ!" {
		t.Errorf("Received message is incorrect. Expected: %s, Got: %s", "Hello, ZeroMQ!", receivedMessage)
	}

	// Remove the subscription
	if err := zp.Remove("test"); err != nil {
		t.Fatalf("Failed to remove subscription: %v", err)
	}

	// Close the connection
	zp.Close()
}
