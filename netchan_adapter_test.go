package hybridpipe

import (
	"sync"
	"testing"
	"time"
)

func TestNetChanAdapter(t *testing.T) {
	// Create a new adapter
	adapter, err := NewNetChanAdapter(10)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Close()

	t.Run("SendAndReceive", func(t *testing.T) {
		// Get send and receive channels
		sendCh, err := adapter.GetSendChan("test")
		if err != nil {
			t.Fatalf("Failed to get send channel: %v", err)
		}

		recvCh, err := adapter.GetRecvChan("test")
		if err != nil {
			t.Fatalf("Failed to get receive channel: %v", err)
		}

		// Set up a wait group to wait for the message to be received
		var wg sync.WaitGroup
		wg.Add(1)

		// Set up a variable to store the received message
		var receivedMessage string

		// Start a goroutine to receive the message
		go func() {
			defer wg.Done()
			msg, ok := <-recvCh
			if !ok {
				t.Errorf("Channel closed unexpectedly")
				return
			}
			if str, ok := msg.(string); ok {
				receivedMessage = str
			} else {
				t.Errorf("Received message is not a string: %v", msg)
			}
		}()

		// Send a message
		sendCh <- "Hello, NetChan!"

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
	})

	t.Run("MultipleMessages", func(t *testing.T) {
		// Get send and receive channels
		sendCh, err := adapter.GetSendChan("test-multiple")
		if err != nil {
			t.Fatalf("Failed to get send channel: %v", err)
		}

		recvCh, err := adapter.GetRecvChan("test-multiple")
		if err != nil {
			t.Fatalf("Failed to get receive channel: %v", err)
		}

		// Number of messages to send
		numMessages := 10
		messages := make([]string, numMessages)
		for i := 0; i < numMessages; i++ {
			messages[i] = "Message " + string(rune('A'+i))
		}

		// Set up a wait group to wait for all messages to be received
		var wg sync.WaitGroup
		wg.Add(numMessages)

		// Set up a slice to store the received messages
		receivedMessages := make([]string, 0, numMessages)
		var receiveMutex sync.Mutex

		// Start a goroutine to receive the messages
		go func() {
			for i := 0; i < numMessages; i++ {
				msg, ok := <-recvCh
				if !ok {
					t.Errorf("Channel closed unexpectedly")
					return
				}
				if str, ok := msg.(string); ok {
					receiveMutex.Lock()
					receivedMessages = append(receivedMessages, str)
					receiveMutex.Unlock()
					wg.Done()
				} else {
					t.Errorf("Received message is not a string: %v", msg)
					wg.Done()
				}
			}
		}()

		// Send the messages
		for _, msg := range messages {
			sendCh <- msg
		}

		// Wait for all messages to be received with a timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All messages received
		case <-time.After(5 * time.Second):
			t.Fatal("Timed out waiting for messages")
		}

		// Check if the correct number of messages was received
		receiveMutex.Lock()
		if len(receivedMessages) != numMessages {
			t.Errorf("Incorrect number of messages received. Expected: %d, Got: %d", numMessages, len(receivedMessages))
		}
		receiveMutex.Unlock()
	})

	t.Run("RemovePipe", func(t *testing.T) {
		// Get send and receive channels
		sendCh, err := adapter.GetSendChan("test-remove")
		if err != nil {
			t.Fatalf("Failed to get send channel: %v", err)
		}

		recvCh, err := adapter.GetRecvChan("test-remove")
		if err != nil {
			t.Fatalf("Failed to get receive channel: %v", err)
		}

		// Send and receive a message to verify the pipe works
		var wg sync.WaitGroup
		wg.Add(1)

		// Variable to store the received message
		_ = ""
		go func() {
			defer wg.Done()
			msg, ok := <-recvCh
			if !ok {
				t.Errorf("Channel closed unexpectedly")
				return
			}
			if _, ok := msg.(string); !ok {
				t.Errorf("Received message is not a string: %v", msg)
			}
		}()

		sendCh <- "Test message"

		// Wait for the message to be received
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

		// Remove the pipe
		if err := adapter.RemovePipe("test-remove"); err != nil {
			t.Fatalf("Failed to remove pipe: %v", err)
		}

		// Try to get the channels again
		_, err = adapter.GetSendChan("test-remove")
		if err != nil {
			t.Fatalf("Failed to get send channel after removal: %v", err)
		}

		_, err = adapter.GetRecvChan("test-remove")
		if err != nil {
			t.Fatalf("Failed to get receive channel after removal: %v", err)
		}
	})
}
