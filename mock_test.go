package hybridpipe

import (
	"testing"
	"time"
)

// TestMockImplementation tests the mock implementation of HybridPipe.
func TestMockImplementation(t *testing.T) {
	// Register a test data type
	type TestData struct {
		ID      string
		Value   int
		Message string
	}
	Enable(TestData{})

	// Create a mock router
	mock, err := DeployRouter(MOCK)
	if err != nil {
		t.Fatalf("Failed to create mock router: %v", err)
	}
	defer mock.Close()

	// Get the mock packet for additional testing
	mockPacket, ok := mock.(*MockPacket)
	if !ok {
		t.Fatalf("Failed to cast to MockPacket")
	}

	// Test dispatch
	testData := TestData{
		ID:      "test-1",
		Value:   42,
		Message: "Hello, Mock!",
	}

	// Dispatch a message
	err = mock.Dispatch("test.topic", testData)
	if err != nil {
		t.Fatalf("Failed to dispatch message: %v", err)
	}

	// Verify message count
	if count := mockPacket.GetMessageCount(); count != 1 {
		t.Errorf("Expected message count to be 1, got %d", count)
	}

	// Verify messages
	messages := mockPacket.GetMessages("test.topic")
	if len(messages) != 1 {
		t.Errorf("Expected 1 message in test.topic, got %d", len(messages))
	}

	// Test subscription
	receivedData := make(chan TestData, 1)

	err = mock.Accept("test.topic", func(data interface{}) {
		if td, ok := data.(TestData); ok {
			receivedData <- td
		}
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Verify subscription count
	if count := mockPacket.GetSubscriptionCount(); count != 1 {
		t.Errorf("Expected subscription count to be 1, got %d", count)
	}

	// Dispatch another message
	testData2 := TestData{
		ID:      "test-2",
		Value:   84,
		Message: "Hello again, Mock!",
	}

	err = mock.Dispatch("test.topic", testData2)
	if err != nil {
		t.Fatalf("Failed to dispatch second message: %v", err)
	}

	// Wait for the message to be processed
	select {
	case received := <-receivedData:
		// Just verify we received something, the exact data might vary due to serialization
		if received.ID == "" || received.Value == 0 || received.Message == "" {
			t.Errorf("Received empty or invalid data")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Timed out waiting for message")
	}

	// Test unsubscribe
	err = mock.Remove("test.topic")
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	// Verify subscription count after unsubscribe
	if count := mockPacket.GetSubscriptionCount(); count != 0 {
		t.Errorf("Expected subscription count to be 0 after unsubscribe, got %d", count)
	}

	// Test error simulation
	mockPacket.SetErrorMode(true)

	err = mock.Dispatch("test.topic", testData)
	if err == nil {
		t.Errorf("Expected error when dispatching in error mode, got nil")
	}

	// Reset error mode
	mockPacket.SetErrorMode(false)

	// Test delay simulation
	mockPacket.SetDelay(100 * time.Millisecond)

	start := time.Now()
	err = mock.Dispatch("test.topic", testData)
	if err != nil {
		t.Fatalf("Failed to dispatch message with delay: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < 100*time.Millisecond {
		t.Errorf("Expected delay of at least 100ms, got %v", elapsed)
	}

	// Reset delay
	mockPacket.SetDelay(0)

	// Test message clearing
	mockPacket.ClearMessages("test.topic")

	messages = mockPacket.GetMessages("test.topic")
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clearing, got %d", len(messages))
	}

	// Test reset
	mockPacket.Reset()

	if count := mockPacket.GetMessageCount(); count != 0 {
		t.Errorf("Expected message count to be 0 after reset, got %d", count)
	}

	if count := mockPacket.GetSubscriptionCount(); count != 0 {
		t.Errorf("Expected subscription count to be 0 after reset, got %d", count)
	}
}
