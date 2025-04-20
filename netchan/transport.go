package netchan

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Transport handles the network communication for a channel.
type Transport struct {
	// Connection
	conn     net.Conn
	connLock sync.RWMutex

	// Configuration
	options Options

	// State
	closed    bool
	closeLock sync.RWMutex
	done      chan struct{}
	doneOnce  sync.Once

	// Addresses
	listenAddr string
	dialAddr   string

	// Message handling
	onMessage func(*Message) error
	onClose   func(error)

	// Reconnection
	reconnecting bool
	reconnectMu  sync.Mutex
}

// NewTransport creates a new transport.
func NewTransport(options Options) *Transport {
	return &Transport{
		options: options,
		done:    make(chan struct{}),
	}
}

// Listen starts listening for connections.
func (t *Transport) Listen(addr string) error {
	t.listenAddr = addr

	// Create a listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("netchan: failed to listen: %w", err)
	}

	// Start the accept loop
	go t.acceptLoop(listener)

	return nil
}

// acceptLoop accepts incoming connections.
func (t *Transport) acceptLoop(listener net.Listener) {
	defer listener.Close()

	for {
		// Check if we're shutting down
		select {
		case <-t.done:
			return
		default:
			// Continue accepting
		}

		// Set accept timeout
		if t.options.Timeout > 0 {
			listener.(*net.TCPListener).SetDeadline(time.Now().Add(t.options.Timeout))
		}

		// Accept a connection
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout, just retry
				continue
			}

			// Check if we're shutting down
			select {
			case <-t.done:
				return
			default:
				// Log the error and continue
				// log.Printf("netchan: accept error: %v", err)
				continue
			}
		}

		// Apply TLS if configured
		if t.options.TLSConfig != nil {
			tlsConfig := t.options.TLSConfig.(*tls.Config)
			conn = tls.Server(conn, tlsConfig)
		}

		// Set the connection
		t.setConnection(conn)

		// Start the reader loop
		go t.readLoop()
	}
}

// Dial connects to a remote endpoint.
func (t *Transport) Dial(addr string) error {
	t.dialAddr = addr

	// Connect to the remote endpoint
	conn, err := net.DialTimeout("tcp", addr, t.options.Timeout)
	if err != nil {
		return fmt.Errorf("netchan: failed to dial: %w", err)
	}

	// Apply TLS if configured
	if t.options.TLSConfig != nil {
		tlsConfig := t.options.TLSConfig.(*tls.Config)
		conn = tls.Client(conn, tlsConfig)
	}

	// Set the connection
	t.setConnection(conn)

	// Start the reader loop
	go t.readLoop()

	return nil
}

// setConnection sets the current connection.
func (t *Transport) setConnection(conn net.Conn) {
	t.connLock.Lock()
	defer t.connLock.Unlock()

	// Close the old connection if it exists
	if t.conn != nil {
		t.conn.Close()
	}

	t.conn = conn
}

// getConnection gets the current connection.
func (t *Transport) getConnection() net.Conn {
	t.connLock.RLock()
	defer t.connLock.RUnlock()
	return t.conn
}

// readLoop reads messages from the connection.
func (t *Transport) readLoop() {
	conn := t.getConnection()
	if conn == nil {
		return
	}

	for {
		// Check if we're shutting down
		select {
		case <-t.done:
			return
		default:
			// Continue reading
		}

		// Set read timeout
		if t.options.Timeout > 0 {
			conn.SetReadDeadline(time.Now().Add(t.options.Timeout))
		}

		// Read a message
		msg, err := ReadMessage(conn)
		if err != nil {
			if err == io.EOF || isConnectionClosed(err) {
				// Connection closed
				t.handleConnectionClosed(err)
				return
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout, just retry
				continue
			}

			// Other error
			t.handleConnectionClosed(err)
			return
		}

		// Process the message
		if t.onMessage != nil {
			if err := t.onMessage(msg); err != nil {
				// log.Printf("netchan: message handler error: %v", err)
			}
		}
	}
}

// handleConnectionClosed handles a closed connection.
func (t *Transport) handleConnectionClosed(err error) {
	// Check if we're shutting down
	select {
	case <-t.done:
		return
	default:
		// Continue handling
	}

	// Notify the channel
	if t.onClose != nil {
		t.onClose(err)
	}

	// Try to reconnect if enabled
	if t.options.Reconnect && t.dialAddr != "" {
		t.reconnect()
	}
}

// reconnect attempts to reconnect to the remote endpoint.
func (t *Transport) reconnect() {
	t.reconnectMu.Lock()
	if t.reconnecting {
		t.reconnectMu.Unlock()
		return
	}
	t.reconnecting = true
	t.reconnectMu.Unlock()

	go func() {
		defer func() {
			t.reconnectMu.Lock()
			t.reconnecting = false
			t.reconnectMu.Unlock()
		}()

		backoff := t.options.ReconnectBackoff
		delay := backoff.InitialDelay

		for attempt := 1; ; attempt++ {
			// Check if we're shutting down
			select {
			case <-t.done:
				return
			case <-time.After(delay):
				// Continue reconnecting
			}

			// Try to reconnect
			conn, err := net.DialTimeout("tcp", t.dialAddr, t.options.Timeout)
			if err == nil {
				// Apply TLS if configured
				if t.options.TLSConfig != nil {
					tlsConfig := t.options.TLSConfig.(*tls.Config)
					conn = tls.Client(conn, tlsConfig)
				}

				// Set the connection
				t.setConnection(conn)

				// Start the reader loop
				go t.readLoop()

				// Reconnected successfully
				return
			}

			// Calculate next delay
			delay = calculateBackoff(delay, backoff)
		}
	}()
}

// calculateBackoff calculates the next backoff delay.
func calculateBackoff(currentDelay time.Duration, policy BackoffPolicy) time.Duration {
	// Apply the factor
	nextDelay := time.Duration(float64(currentDelay) * policy.Factor)

	// Apply jitter
	if policy.Jitter > 0 {
		jitter := rand.Float64() * policy.Jitter
		nextDelay = time.Duration(float64(nextDelay) * (1.0 + jitter))
	}

	// Cap at max delay
	if nextDelay > policy.MaxDelay {
		nextDelay = policy.MaxDelay
	}

	return nextDelay
}

// SendMessage sends a message.
func (t *Transport) SendMessage(msg *Message) error {
	conn := t.getConnection()
	if conn == nil {
		return ErrConnectionClosed
	}

	// Set write timeout
	if t.options.Timeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(t.options.Timeout))
	}

	// Write the message
	return WriteMessage(conn, msg)
}

// Close closes the transport.
func (t *Transport) Close() error {
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

	// Close the connection
	t.connLock.Lock()
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
	t.connLock.Unlock()

	return nil
}

// SetMessageHandler sets the message handler.
func (t *Transport) SetMessageHandler(handler func(*Message) error) {
	t.onMessage = handler
}

// SetCloseHandler sets the close handler.
func (t *Transport) SetCloseHandler(handler func(error)) {
	t.onClose = handler
}

// isConnectionClosed checks if an error indicates a closed connection.
func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}

	// Check for common "connection closed" errors
	if err == io.EOF {
		return true
	}

	if opErr, ok := err.(*net.OpError); ok {
		return opErr.Err.Error() == "use of closed network connection"
	}

	return false
}
