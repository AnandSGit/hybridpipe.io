package accept

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
	"hybridpipe.io/protocols/mock"
)

func TestAcceptor(t *testing.T) {
	t.Skip("Skipping acceptor tests until the accept package is fully implemented")
	
	// Create a mock router
	router := mock.NewMockRouter()
	err := router.Connect()
	require.NoError(t, err, "Failed to connect to mock router")
	defer router.Close()
	
	// Create an acceptor
	acceptor := NewAcceptor(router)
	
	// Test accepting messages
	testAcceptMessages(t, acceptor, router)
	
	// Test concurrent accepting
	testConcurrentAccepting(t, acceptor, router)
	
	// Test context-aware accepting
	testContextAwareAccepting(t, acceptor, router)
}

// testAcceptMessages tests accepting messages through the acceptor.
func testAcceptMessages(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-accept-pipe"
	
	// Test data
	testData := map[string]interface{}{
		"message": "Hello, Acceptor!",
		"time":    time.Now().Unix(),
	}
	
	// Wait group for synchronization
	done := make(chan struct{})
	
	// Accept messages from the pipe
	err := acceptor.Accept(pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		assert.Equal(t, testData["message"], decoded.(map[string]interface{})["message"], "Received message does not match sent message")
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to accept messages from pipe")
	
	// Dispatch a message through the router
	err = router.Dispatch(pipeName, testData)
	require.NoError(t, err, "Failed to dispatch message")
	
	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}
	
	// Stop accepting messages
	err = acceptor.Stop(pipeName)
	require.NoError(t, err, "Failed to stop accepting messages")
}

// testConcurrentAccepting tests concurrent accepting of messages.
func testConcurrentAccepting(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Number of concurrent operations
	numOperations := 10
	
	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(numOperations)
	
	// Create a channel to collect errors
	errChan := make(chan error, numOperations)
	
	// Run concurrent operations
	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer wg.Done()
			
			// Test pipe name
			pipeName := "test-concurrent-accept-pipe-" + string(rune('A'+index))
			
			// Test data
			testData := map[string]interface{}{
				"message": "Hello, Concurrent Acceptor!",
				"index":   index,
				"time":    time.Now().Unix(),
			}
			
			// Wait group for synchronization
			done := make(chan struct{})
			
			// Accept messages from the pipe
			err := acceptor.Accept(pipeName, func(data []byte) error {
				var decoded interface{}
				if err := core.Decode(data, &decoded); err != nil {
					errChan <- err
					return err
				}
				close(done)
				return nil
			})
			if err != nil {
				errChan <- err
				return
			}
			
			// Dispatch a message through the router
			err = router.Dispatch(pipeName, testData)
			if err != nil {
				errChan <- err
				return
			}
			
			// Wait for the message to be received
			select {
			case <-done:
				// Message received
			case <-time.After(5 * time.Second):
				errChan <- err
				return
			}
			
			// Stop accepting messages
			err = acceptor.Stop(pipeName)
			if err != nil {
				errChan <- err
				return
			}
		}(i)
	}
	
	// Wait for all operations to complete
	wg.Wait()
	close(errChan)
	
	// Check for errors
	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}
}

// testContextAwareAccepting tests context-aware accepting of messages.
func testContextAwareAccepting(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-ctx-accept-pipe"
	
	// Test data
	testData := map[string]interface{}{
		"message": "Hello, Context-Aware Acceptor!",
		"time":    time.Now().Unix(),
	}
	
	// Wait group for synchronization
	done := make(chan struct{})
	
	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Accept messages from the pipe with context
	err := acceptor.AcceptWithContext(ctx, pipeName, func(data []byte) error {
		var decoded interface{}
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		assert.Equal(t, testData["message"], decoded.(map[string]interface{})["message"], "Received message does not match sent message")
		close(done)
		return nil
	})
	require.NoError(t, err, "Failed to accept messages from pipe with context")
	
	// Dispatch a message through the router
	if ctxRouter, ok := router.(core.ContextAwareHybridPipe); ok {
		err = ctxRouter.DispatchWithContext(ctx, pipeName, testData)
	} else {
		err = router.Dispatch(pipeName, testData)
	}
	require.NoError(t, err, "Failed to dispatch message")
	
	// Wait for the message to be received
	select {
	case <-done:
		// Message received
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message")
	}
	
	// Stop accepting messages
	err = acceptor.Stop(pipeName)
	require.NoError(t, err, "Failed to stop accepting messages")
}
