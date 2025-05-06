package netchan

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// MockTransport is a mock implementation of the transport layer for testing.
type MockTransport struct {
	// Peer is the other end of the connection
	peer *MockTransport

	// Message handling
	onMessage func(*Message) error
	onClose   func(error)

	// Message queue
	queue     []*Message
	queueLock sync.Mutex
	queueCond *sync.Cond

	// State
	closed    bool
	closeLock sync.RWMutex
	done      chan struct{}
	doneOnce  sync.Once

	// Configuration
	options Options

	// Simulation
	latency    time.Duration
	packetLoss float64
	errorRate  float64
}

// NewMockTransport creates a new mock transport.
func NewMockTransport(options Options) *MockTransport {
	t := &MockTransport{
		options: options,
		done:    make(chan struct{}),
	}
	t.queueCond = sync.NewCond(&t.queueLock)
	return t
}

// Connect connects this transport to another mock transport.
func (t *MockTransport) Connect(peer *MockTransport) {
	t.peer = peer
	peer.peer = t

	// Start the message processing goroutines
	go t.processQueue()
	go peer.processQueue()
}

// processQueue processes messages in the queue.
func (t *MockTransport) processQueue() {
	for {
		// Check if we're shutting down
		if isShutdown := func() bool {
			select {
			case <-t.done:
				return true
			default:
				return false
			}
		}(); isShutdown {
			return
		}

		// Get a message from the queue
		t.queueLock.Lock()
		for len(t.queue) == 0 && !t.closed {
			t.queueCond.Wait()
		}

		if t.closed {
			t.queueLock.Unlock()
			return
		}

		msg := t.queue[0]
		t.queue = t.queue[1:]
		t.queueLock.Unlock()

		// Process the message
		if t.onMessage != nil {
			if err := t.onMessage(msg); err != nil {
				// log.Printf("netchan: mock transport message handler error: %v", err)
			}
		}
	}
}

// SendMessage sends a message to the peer.
func (t *MockTransport) SendMessage(msg *Message) error {
	// Check if we're closed
	t.closeLock.RLock()
	if t.closed {
		t.closeLock.RUnlock()
		return ErrConnectionClosed
	}
	t.closeLock.RUnlock()

	// Check if the peer is connected
	if t.peer == nil {
		return errors.New("netchan: mock transport not connected")
	}

	// Simulate latency
	if t.latency > 0 {
		time.Sleep(t.latency)
	}

	// Simulate packet loss
	if t.packetLoss > 0 && t.packetLoss > float64(time.Now().UnixNano()%100)/100.0 {
		return nil // Silently drop the message
	}

	// Simulate errors
	if t.errorRate > 0 && t.errorRate > float64(time.Now().UnixNano()%100)/100.0 {
		return fmt.Errorf("netchan: mock transport simulated error")
	}

	// Add the message to the peer's queue
	t.peer.queueLock.Lock()
	t.peer.queue = append(t.peer.queue, msg)
	t.peer.queueCond.Signal()
	t.peer.queueLock.Unlock()

	return nil
}

// Close closes the transport.
func (t *MockTransport) Close() error {
	// Set the closed flag
	t.closeLock.Lock()
	if t.closed {
		t.closeLock.Unlock()
		return nil // Already closed
	}
	t.closed = true
	t.closeLock.Unlock()

	// Signal shutdown
	t.doneOnce.Do(func() {
		close(t.done)
	})

	// Signal the queue condition
	t.queueLock.Lock()
	t.queueCond.Broadcast()
	t.queueLock.Unlock()

	// Notify the peer
	if t.peer != nil && t.peer.onClose != nil {
		t.peer.onClose(ErrConnectionClosed)
	}

	return nil
}

// SetMessageHandler sets the message handler.
func (t *MockTransport) SetMessageHandler(handler func(*Message) error) {
	t.onMessage = handler
}

// SetCloseHandler sets the close handler.
func (t *MockTransport) SetCloseHandler(handler func(error)) {
	t.onClose = handler
}

// SetLatency sets the simulated network latency.
func (t *MockTransport) SetLatency(latency time.Duration) {
	t.latency = latency
}

// SetPacketLoss sets the simulated packet loss rate (0.0-1.0).
func (t *MockTransport) SetPacketLoss(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	t.packetLoss = rate
}

// SetErrorRate sets the simulated error rate (0.0-1.0).
func (t *MockTransport) SetErrorRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	t.errorRate = rate
}

// QueueLength returns the current queue length.
func (t *MockTransport) QueueLength() int {
	t.queueLock.Lock()
	defer t.queueLock.Unlock()
	return len(t.queue)
}
