package amqp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"hybridpipe.io/core"
)

func TestAMQPRouter(t *testing.T) {
	// Skip in short mode as this test requires an AMQP server
	if testing.Short() {
		t.Skip("Skipping AMQP tests in short mode")
	}

	// Create a mock AMQP config
	config := &core.AMQPF{
		AMQPServer: "amqp://guest:guest@localhost:5672",
	}

	// Create a new AMQP router
	router := New(config)

	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// Test dispatching when not connected
		err := router.Dispatch("test-pipe", "test-data")
		assert.Error(t, err, "Dispatch should return an error when not connected")
		assert.Equal(t, core.ErrNotConnected, err, "Dispatch should return ErrNotConnected when not connected")

		// Test removing when not connected
		err = router.Remove("test-pipe")
		assert.Error(t, err, "Remove should return an error when not connected")
		assert.Equal(t, core.ErrNotConnected, err, "Remove should return ErrNotConnected when not connected")
	})
}
