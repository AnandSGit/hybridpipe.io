package accept

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hybridpipe.io/core"
	"hybridpipe.io/protocols/mock"
)

func TestAcceptor(t *testing.T) {
	// Create a mock router
	router := mock.NewMockRouter()
	err := router.Connect()
	require.NoError(t, err, "Failed to connect to mock router")
	defer router.Close()

	// Create an acceptor
	acceptor := NewAcceptor(router)

	// Test accepting messages
	testAcceptMessages(t, acceptor, router)

	// Test context-aware accepting
	testContextAwareAccepting(t, acceptor, router)

	// Test stopping acceptance
	testStopAccepting(t, acceptor, router)

	// Test stopping all acceptance
	testStopAllAccepting(t, acceptor, router)

	// Test IsAccepting method
	testIsAccepting(t, acceptor, router)

	// Test GetPipes method
	testGetPipes(t, acceptor, router)
}

// testAcceptMessages tests accepting messages through the acceptor.
func testAcceptMessages(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-accept-pipe"

	// Test data - use a simple string instead of a map to avoid serialization issues
	testData := "Hello, Acceptor!"

	// Wait group for synchronization
	done := make(chan struct{})

	// Accept messages from the pipe
	err := acceptor.Accept(pipeName, func(data []byte) error {
		var decoded string
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		assert.Equal(t, testData, decoded, "Received message does not match sent message")
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

// testStopAccepting tests stopping acceptance of messages.
func testStopAccepting(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-stop-accept-pipe"

	// Accept messages from the pipe
	err := acceptor.Accept(pipeName, func(data []byte) error {
		return nil
	})
	require.NoError(t, err, "Failed to accept messages from pipe")

	// Verify that we're accepting from the pipe
	assert.True(t, acceptor.IsAccepting(pipeName), "Should be accepting from pipe after Accept")

	// Stop accepting messages
	err = acceptor.Stop(pipeName)
	require.NoError(t, err, "Failed to stop accepting messages")

	// Verify that we're no longer accepting from the pipe
	assert.False(t, acceptor.IsAccepting(pipeName), "Should not be accepting from pipe after Stop")

	// Test stopping a pipe that doesn't exist
	err = acceptor.Stop("non-existent-pipe")
	assert.Error(t, err, "Stop should return an error for a non-existent pipe")
}

// testStopAllAccepting tests stopping acceptance of messages from all pipes.
func testStopAllAccepting(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Test pipe names
	pipeNames := []string{
		"test-stop-all-accept-pipe-1",
		"test-stop-all-accept-pipe-2",
		"test-stop-all-accept-pipe-3",
	}

	// Accept messages from all pipes
	for _, pipeName := range pipeNames {
		err := acceptor.Accept(pipeName, func(data []byte) error {
			return nil
		})
		require.NoError(t, err, "Failed to accept messages from pipe")
	}

	// Verify that we're accepting from all pipes
	for _, pipeName := range pipeNames {
		assert.True(t, acceptor.IsAccepting(pipeName), "Should be accepting from pipe after Accept")
	}

	// Stop accepting messages from all pipes
	err := acceptor.StopAll()
	require.NoError(t, err, "Failed to stop accepting messages from all pipes")

	// Verify that we're no longer accepting from any pipe
	for _, pipeName := range pipeNames {
		assert.False(t, acceptor.IsAccepting(pipeName), "Should not be accepting from pipe after StopAll")
	}
}

// testIsAccepting tests the IsAccepting method.
func testIsAccepting(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-is-accepting-pipe"

	// Initially, we should not be accepting from the pipe
	assert.False(t, acceptor.IsAccepting(pipeName), "Should not be accepting from pipe initially")

	// Accept messages from the pipe
	err := acceptor.Accept(pipeName, func(data []byte) error {
		return nil
	})
	require.NoError(t, err, "Failed to accept messages from pipe")

	// Now, we should be accepting from the pipe
	assert.True(t, acceptor.IsAccepting(pipeName), "Should be accepting from pipe after Accept")

	// Stop accepting messages
	err = acceptor.Stop(pipeName)
	require.NoError(t, err, "Failed to stop accepting messages")

	// Now, we should not be accepting from the pipe again
	assert.False(t, acceptor.IsAccepting(pipeName), "Should not be accepting from pipe after Stop")
}

// testGetPipes tests the GetPipes method.
func testGetPipes(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Initially, we should not be accepting from any pipes
	pipes := acceptor.GetPipes()
	assert.Empty(t, pipes, "Should not be accepting from any pipes initially")

	// Test pipe names
	pipeNames := []string{
		"test-get-pipes-pipe-1",
		"test-get-pipes-pipe-2",
		"test-get-pipes-pipe-3",
	}

	// Accept messages from all pipes
	for _, pipeName := range pipeNames {
		err := acceptor.Accept(pipeName, func(data []byte) error {
			return nil
		})
		require.NoError(t, err, "Failed to accept messages from pipe")
	}

	// Get the list of pipes
	pipes = acceptor.GetPipes()

	// Verify that we're accepting from all pipes
	assert.Equal(t, len(pipeNames), len(pipes), "Should be accepting from all pipes")
	for _, pipeName := range pipeNames {
		assert.Contains(t, pipes, pipeName, "GetPipes should include all pipes we're accepting from")
	}

	// Stop accepting messages from all pipes
	err := acceptor.StopAll()
	require.NoError(t, err, "Failed to stop accepting messages from all pipes")

	// Now, we should not be accepting from any pipes again
	pipes = acceptor.GetPipes()
	assert.Empty(t, pipes, "Should not be accepting from any pipes after StopAll")
}

// testContextAwareAccepting tests context-aware accepting of messages.
func testContextAwareAccepting(t *testing.T, acceptor *Acceptor, router core.HybridPipe) {
	// Test pipe name
	pipeName := "test-ctx-accept-pipe"

	// Test data - use a simple string instead of a map to avoid serialization issues
	testData := "Hello, Context-Aware Acceptor!"

	// Wait group for synchronization
	done := make(chan struct{})

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Accept messages from the pipe with context
	err := acceptor.AcceptWithContext(ctx, pipeName, func(data []byte) error {
		var decoded string
		if err := core.Decode(data, &decoded); err != nil {
			t.Errorf("Failed to decode message: %v", err)
			return err
		}
		assert.Equal(t, testData, decoded, "Received message does not match sent message")
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
