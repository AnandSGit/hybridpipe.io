// Package tcp provides an implementation of the HybridPipe interface for direct TCP/IP communication.
package tcp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"hybridpipe.io/core"
)

// Packet implements the HybridPipe interface for direct TCP/IP communication.
type Packet struct {
	// mode can be "client" or "server"
	mode string
	// conn is the TCP connection (client mode)
	conn net.Conn
	// listener is the TCP listener (server mode)
	listener net.Listener
	// clients maps client addresses to their connections (server mode)
	clients map[string]net.Conn
	// handlers maps pipe names to their message handlers
	handlers map[string]core.Process
	// stopChan is used to signal goroutines to stop
	stopChan chan struct{}
	// mutex protects concurrent access to the maps and connection
	mutex sync.RWMutex
	// running indicates if the server is running
	running bool
	// config holds the TCP configuration
	config *core.TCPConfig
}

// New creates a new TCP packet with the specified configuration.
func New(config *core.TCPConfig) *Packet {
	return &Packet{
		mode:     config.TCPMode,
		clients:  make(map[string]net.Conn),
		handlers: make(map[string]core.Process),
		stopChan: make(chan struct{}),
		config:   config,
	}
}

// createTCPTLSConfig creates a TLS configuration from certificate files.
func createTCPTLSConfig(certFile, keyFile, caFile string, isServer bool) (*tls.Config, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, nil
	}

	// Load client/server cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Load CA cert
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if isServer {
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// Connect establishes a connection to the TCP server or starts a TCP server.
func (tp *Packet) Connect() error {
	// Get configuration if not already set
	if tp.config == nil {
		config, err := core.GetProtocolConfig(core.TCP)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		tcpConfig := config.(core.TCPConfig)
		tp.config = &tcpConfig
	}

	// Initialize maps if needed
	tp.mutex.Lock()
	if tp.clients == nil {
		tp.clients = make(map[string]net.Conn)
	}
	if tp.handlers == nil {
		tp.handlers = make(map[string]core.Process)
	}
	if tp.stopChan == nil {
		tp.stopChan = make(chan struct{})
	}
	tp.mutex.Unlock()

	// Set up TLS if enabled
	var tlsConfig *tls.Config
	var err error
	if tp.config.TCPTLSEnabled {
		isServer := tp.mode == "server"
		tlsConfig, err = createTCPTLSConfig(
			tp.config.TCPCertFile,
			tp.config.TCPKeyFile,
			tp.config.TCPCAFile,
			isServer,
		)
		if err != nil {
			return fmt.Errorf("failed to create TLS configuration: %w", err)
		}
	}

	// Set connection timeout
	timeout := time.Duration(tp.config.TCPConnectTimeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second // Default timeout
	}

	// Handle based on mode
	if tp.mode == "server" {
		// Start TCP server
		return tp.startServer(tlsConfig)
	} else {
		// Connect as TCP client
		return tp.connectClient(tlsConfig, timeout)
	}
}

// startServer starts a TCP server.
func (tp *Packet) startServer(tlsConfig *tls.Config) error {
	// Create address
	addr := fmt.Sprintf("%s:%d", tp.config.TCPHost, tp.config.TCPPort)

	// Create listener
	var listener net.Listener
	var err error

	if tlsConfig != nil {
		listener, err = tls.Listen("tcp", addr, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("failed to start TCP server on %s: %w", addr, err)
	}

	// Store the listener
	tp.mutex.Lock()
	tp.listener = listener
	tp.running = true
	tp.mutex.Unlock()

	// Start accepting connections
	go tp.acceptConnections()

	log.Printf("Started TCP server on %s", addr)
	return nil
}

// acceptConnections accepts incoming TCP connections.
func (tp *Packet) acceptConnections() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in TCP server: %v", r)
		}
	}()

	for {
		// Check if server is still running
		tp.mutex.RLock()
		listener := tp.listener
		running := tp.running
		tp.mutex.RUnlock()

		if !running || listener == nil {
			log.Printf("TCP server stopped")
			return
		}

		// Accept a connection
		conn, err := listener.Accept()
		if err != nil {
			// Check if server is still running
			tp.mutex.RLock()
			running := tp.running
			tp.mutex.RUnlock()

			if !running {
				return
			}

			log.Printf("Error accepting TCP connection: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Set keep alive if enabled
		if tp.config.TCPKeepAlive {
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				tcpConn.SetKeepAlive(true)
				if tp.config.TCPKeepAlivePeriod > 0 {
					tcpConn.SetKeepAlivePeriod(time.Duration(tp.config.TCPKeepAlivePeriod) * time.Second)
				}
			}
		}

		// Store the connection
		clientAddr := conn.RemoteAddr().String()
		tp.mutex.Lock()
		tp.clients[clientAddr] = conn
		tp.mutex.Unlock()

		log.Printf("Accepted TCP connection from %s", clientAddr)

		// Handle the connection
		go tp.handleConnection(conn)
	}
}

// handleConnection handles a TCP connection.
func (tp *Packet) handleConnection(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in TCP connection handler: %v", r)
		}
		conn.Close()

		// Remove the connection from the clients map
		clientAddr := conn.RemoteAddr().String()
		tp.mutex.Lock()
		delete(tp.clients, clientAddr)
		tp.mutex.Unlock()

		log.Printf("Closed TCP connection from %s", clientAddr)
	}()

	// Buffer for reading message length
	lenBuf := make([]byte, 4)

	for {
		// Check if server is still running
		tp.mutex.RLock()
		running := tp.running
		stopChan := tp.stopChan
		tp.mutex.RUnlock()

		if !running {
			return
		}

		// Check for stop signal
		select {
		case <-stopChan:
			return
		default:
			// Continue processing
		}

		// Read message length
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			if err != io.EOF {
				log.Printf("Error reading message length: %v", err)
			}
			return
		}

		// Convert to uint32
		msgLen := binary.BigEndian.Uint32(lenBuf)
		if msgLen == 0 {
			continue
		}

		// Read message
		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(conn, msgBuf); err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		// Process the message
		tp.processMessage(msgBuf)
	}
}

// processMessage processes a received TCP message.
func (tp *Packet) processMessage(data []byte) {
	// The first part of the message is the pipe name
	// Format: [pipe length (1 byte)][pipe name][message data]
	if len(data) < 1 {
		log.Printf("Received invalid message: too short")
		return
	}

	pipeLen := int(data[0])
	if len(data) < 1+pipeLen {
		log.Printf("Received invalid message: pipe name too long")
		return
	}

	pipe := string(data[1 : 1+pipeLen])
	msgData := data[1+pipeLen:]

	// Find the handler for the pipe
	tp.mutex.RLock()
	handler, exists := tp.handlers[pipe]
	tp.mutex.RUnlock()

	if !exists {
		log.Printf("No handler for pipe %s", pipe)
		return
	}

	// Process the message
	if err := handler(msgData); err != nil {
		log.Printf("Error processing message from pipe %s: %v", pipe, err)
	}
}

// connectClient connects to a TCP server.
func (tp *Packet) connectClient(tlsConfig *tls.Config, timeout time.Duration) error {
	// Create address
	addr := fmt.Sprintf("%s:%d", tp.config.TCPHost, tp.config.TCPPort)

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	// Connect to the server
	var conn net.Conn
	var err error

	if tlsConfig != nil {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to TCP server at %s: %w", addr, err)
	}

	// Set keep alive if enabled
	if tp.config.TCPKeepAlive {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			if tp.config.TCPKeepAlivePeriod > 0 {
				tcpConn.SetKeepAlivePeriod(time.Duration(tp.config.TCPKeepAlivePeriod) * time.Second)
			}
		}
	}

	// Store the connection
	tp.mutex.Lock()
	tp.conn = conn
	tp.running = true
	tp.mutex.Unlock()

	// Start reading messages
	go tp.readMessages()

	log.Printf("Connected to TCP server at %s", addr)
	return nil
}

// readMessages reads messages from the TCP connection.
func (tp *Packet) readMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in TCP client reader: %v", r)
		}

		// Close the connection
		tp.mutex.Lock()
		if tp.conn != nil {
			tp.conn.Close()
			tp.conn = nil
		}
		tp.running = false
		tp.mutex.Unlock()

		log.Printf("Closed TCP connection")
	}()

	// Buffer for reading message length
	lenBuf := make([]byte, 4)

	for {
		// Check if client is still running
		tp.mutex.RLock()
		conn := tp.conn
		running := tp.running
		stopChan := tp.stopChan
		tp.mutex.RUnlock()

		if !running || conn == nil {
			return
		}

		// Check for stop signal
		select {
		case <-stopChan:
			return
		default:
			// Continue processing
		}

		// Read message length
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			if err != io.EOF {
				log.Printf("Error reading message length: %v", err)
			}
			return
		}

		// Convert to uint32
		msgLen := binary.BigEndian.Uint32(lenBuf)
		if msgLen == 0 {
			continue
		}

		// Read message
		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(conn, msgBuf); err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		// Process the message
		tp.processMessage(msgBuf)
	}
}

// Dispatch sends a message to the specified pipe.
func (tp *Packet) Dispatch(pipe string, data any) error {
	// Validate connection with read lock
	tp.mutex.RLock()
	running := tp.running
	mode := tp.mode
	tp.mutex.RUnlock()

	if !running {
		return core.ErrNotConnected
	}

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create the message
	// Format: [pipe length (1 byte)][pipe name][message data]
	pipeBytes := []byte(pipe)
	if len(pipeBytes) > 255 {
		return fmt.Errorf("pipe name too long: %s", pipe)
	}

	msg := make([]byte, 1+len(pipeBytes)+len(bytes))
	msg[0] = byte(len(pipeBytes))
	copy(msg[1:], pipeBytes)
	copy(msg[1+len(pipeBytes):], bytes)

	// Add length prefix
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(msg)))

	// Send the message
	if mode == "client" {
		// Client mode: send to the server
		tp.mutex.RLock()
		conn := tp.conn
		tp.mutex.RUnlock()

		if conn == nil {
			return core.ErrNotConnected
		}

		// Send length prefix
		if _, err := conn.Write(lenBuf); err != nil {
			return fmt.Errorf("failed to send message length: %w", err)
		}

		// Send message
		if _, err := conn.Write(msg); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	} else {
		// Server mode: send to all clients
		tp.mutex.RLock()
		clients := make([]net.Conn, 0, len(tp.clients))
		for _, conn := range tp.clients {
			clients = append(clients, conn)
		}
		tp.mutex.RUnlock()

		if len(clients) == 0 {
			return fmt.Errorf("no clients connected")
		}

		// Send to all clients
		for _, conn := range clients {
			// Send length prefix
			if _, err := conn.Write(lenBuf); err != nil {
				log.Printf("Failed to send message length to %s: %v", conn.RemoteAddr(), err)
				continue
			}

			// Send message
			if _, err := conn.Write(msg); err != nil {
				log.Printf("Failed to send message to %s: %v", conn.RemoteAddr(), err)
				continue
			}
		}
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (tp *Packet) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with dispatch
	}

	// Create a channel to signal completion
	done := make(chan error, 1)

	// Dispatch the message asynchronously
	go func() {
		done <- tp.Dispatch(pipe, data)
	}()

	// Wait for either context cancellation or dispatch completion
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (tp *Packet) Accept(pipe string, fn func(any)) error {
	// Create a wrapper that converts the function to the expected type
	wrapper := func(data []byte) error {
		var decoded any
		if err := core.Decode(data, &decoded); err != nil {
			return err
		}
		fn(decoded)
		return nil
	}

	return tp.Subscribe(pipe, wrapper)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (tp *Packet) Subscribe(pipe string, callback core.Process) error {
	// Validate connection with read lock
	tp.mutex.RLock()
	running := tp.running
	tp.mutex.RUnlock()

	if !running {
		return core.ErrNotConnected
	}

	// Store the handler
	tp.mutex.Lock()
	tp.handlers[pipe] = callback
	tp.mutex.Unlock()

	log.Printf("Subscribed to TCP pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (tp *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with subscription
	}

	// Create a wrapper that checks context before processing
	wrapper := func(data []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return callback(data)
		}
	}

	// Subscribe with the wrapper
	if err := tp.Subscribe(pipe, wrapper); err != nil {
		return err
	}

	// Set up a goroutine to unsubscribe when the context is canceled
	go func() {
		<-ctx.Done()
		tp.Unsubscribe(pipe)
	}()

	return nil
}

// Remove unsubscribes from the specified pipe.
func (tp *Packet) Remove(pipe string) error {
	return tp.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (tp *Packet) Unsubscribe(pipe string) error {
	// Validate connection with read lock
	tp.mutex.RLock()
	running := tp.running
	tp.mutex.RUnlock()

	if !running {
		return core.ErrNotConnected
	}

	// Lock for thread safety
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	// Check if the pipe is subscribed
	if _, exists := tp.handlers[pipe]; !exists {
		return core.ErrPipeNotFound
	}

	// Remove the handler
	delete(tp.handlers, pipe)

	log.Printf("Unsubscribed from TCP pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the TCP server or stops the TCP server.
func (tp *Packet) Disconnect() error {
	// Check if connected with read lock
	tp.mutex.RLock()
	running := tp.running
	mode := tp.mode
	tp.mutex.RUnlock()

	if !running {
		return nil
	}

	// Signal all goroutines to stop
	close(tp.stopChan)

	// Lock for thread safety
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	// Clear the handlers
	tp.handlers = make(map[string]core.Process)

	if mode == "client" {
		// Client mode: close the connection
		if tp.conn != nil {
			if err := tp.conn.Close(); err != nil {
				log.Printf("Error closing TCP connection: %v", err)
			}
			tp.conn = nil
		}
	} else {
		// Server mode: close the listener and all client connections
		if tp.listener != nil {
			if err := tp.listener.Close(); err != nil {
				log.Printf("Error closing TCP listener: %v", err)
			}
			tp.listener = nil
		}

		// Close all client connections
		for addr, conn := range tp.clients {
			if err := conn.Close(); err != nil {
				log.Printf("Error closing TCP connection to %s: %v", addr, err)
			}
		}
		tp.clients = make(map[string]net.Conn)
	}

	tp.running = false
	tp.stopChan = make(chan struct{})

	log.Printf("Disconnected from TCP %s", mode)
	return nil
}

// Close terminates the connection to the TCP server or stops the TCP server.
func (tp *Packet) Close() error {
	return tp.Disconnect()
}

// IsConnected returns true if the connection to the TCP server is active or the TCP server is running.
func (tp *Packet) IsConnected() bool {
	tp.mutex.RLock()
	defer tp.mutex.RUnlock()
	return tp.running
}
