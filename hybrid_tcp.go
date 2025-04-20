package hybridpipe

import (
	"bufio"
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
)

// TCPPacket implements the HybridPipe interface for direct TCP/IP communication.
type TCPPacket struct {
	// mode can be "client" or "server"
	mode string
	// conn is the TCP connection (client mode)
	conn net.Conn
	// listener is the TCP listener (server mode)
	listener net.Listener
	// clients maps client addresses to their connections (server mode)
	clients map[string]net.Conn
	// handlers maps pipe names to their message handlers
	handlers map[string]Process
	// stopChan is used to signal goroutines to stop
	stopChan chan struct{}
	// mutex protects concurrent access to the maps and connection
	mutex sync.RWMutex
	// running indicates if the server is running
	running bool
}

// createTCPTLSConfig creates a TLS configuration for TCP connections.
func createTCPTLSConfig(clientCert, clientKey, caCert string, isServer bool) (*tls.Config, error) {
	// Create a basic TLS configuration
	tlsConfig := tls.Config{
		MinVersion: tls.VersionTLS12, // Enforce minimum TLS version for security
	}

	// Skip TLS setup if certificates are not provided
	if clientCert == "" || clientKey == "" || caCert == "" {
		log.Printf("Warning: TLS certificates not provided, using insecure connection")
		return nil, nil
	}

	// Load client/server certificate
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	// Load CA certificate
	caCertData, err := os.ReadFile(caCert)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Create certificate pool and add CA certificate
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCertData) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	if isServer {
		// Server configuration
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		// Client configuration
		tlsConfig.RootCAs = caCertPool
	}

	return &tlsConfig, nil
}

// Connect establishes a TCP connection or starts a TCP server based on the configuration.
func (tp *TCPPacket) Connect() error {
	// Initialize maps and channels
	tp.mutex.Lock()
	tp.mode = HPipeConfig.TCPF.TCPMode
	if tp.handlers == nil {
		tp.handlers = make(map[string]Process)
	}
	if tp.clients == nil {
		tp.clients = make(map[string]net.Conn)
	}
	if tp.stopChan == nil {
		tp.stopChan = make(chan struct{})
	}
	tp.mutex.Unlock()

	// Set up TLS if enabled
	var tlsConfig *tls.Config
	var err error
	if HPipeConfig.TCPF.TCPTLSEnabled {
		isServer := tp.mode == "server"
		tlsConfig, err = createTCPTLSConfig(
			HPipeConfig.TCPF.TCPCertFile,
			HPipeConfig.TCPF.TCPKeyFile,
			HPipeConfig.TCPF.TCPCAFile,
			isServer,
		)
		if err != nil {
			return fmt.Errorf("failed to create TLS configuration: %w", err)
		}
	}

	// Set connection timeout
	timeout := time.Duration(HPipeConfig.TCPF.TCPConnectTimeout) * time.Second
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

// startServer starts a TCP server that listens for incoming connections.
func (tp *TCPPacket) startServer(tlsConfig *tls.Config) error {
	// Get host and port from configuration with defaults
	host := HPipeConfig.TCPF.TCPHost
	if host == "" {
		host = "0.0.0.0" // Default to all interfaces for server
	}

	port := HPipeConfig.TCPF.TCPPort
	if port <= 0 {
		port = 9000 // Default port
	}

	// Create address string with proper IPv6 support
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Create listener
	var listener net.Listener
	var err error

	if tlsConfig != nil {
		// TLS listener
		listener, err = tls.Listen("tcp", address, tlsConfig)
	} else {
		// Plain TCP listener
		listener, err = net.Listen("tcp", address)
	}

	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	// Store the listener
	tp.mutex.Lock()
	tp.listener = listener
	tp.running = true
	tp.mutex.Unlock()

	// Start accepting connections in a goroutine
	go tp.acceptConnections()

	log.Printf("TCP server started on %s", address)
	return nil
}

// acceptConnections accepts incoming TCP connections and handles them.
func (tp *TCPPacket) acceptConnections() {
	tp.mutex.RLock()
	listener := tp.listener
	stopChan := tp.stopChan
	tp.mutex.RUnlock()

	if listener == nil {
		log.Printf("TCP listener is nil, cannot accept connections")
		return
	}

	for {
		// Check if we should stop
		select {
		case <-stopChan:
			return
		default:
			// Continue accepting
		}

		// Accept a connection
		conn, err := listener.Accept()
		if err != nil {
			// Check if we're shutting down
			select {
			case <-stopChan:
				return
			default:
				log.Printf("Error accepting TCP connection: %v", err)
				continue
			}
		}

		// Configure keep-alive if enabled
		if tcpConn, ok := conn.(*net.TCPConn); ok && HPipeConfig.TCPF.TCPKeepAlive {
			period := time.Duration(HPipeConfig.TCPF.TCPKeepAlivePeriod) * time.Second
			if period == 0 {
				period = 30 * time.Second // Default period
			}
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(period)
		}

		// Store the client connection
		clientAddr := conn.RemoteAddr().String()
		tp.mutex.Lock()
		tp.clients[clientAddr] = conn
		tp.mutex.Unlock()

		log.Printf("Accepted TCP connection from %s", clientAddr)

		// Handle the connection in a goroutine
		go tp.handleConnection(conn)
	}
}

// handleConnection reads messages from a TCP connection and processes them.
func (tp *TCPPacket) handleConnection(conn net.Conn) {
	defer func() {
		// Remove the client when the connection closes
		clientAddr := conn.RemoteAddr().String()
		tp.mutex.Lock()
		delete(tp.clients, clientAddr)
		tp.mutex.Unlock()

		conn.Close()
		log.Printf("Closed TCP connection from %s", clientAddr)
	}()

	// Create a buffered reader
	reader := bufio.NewReader(conn)

	// Set buffer size
	bufferSize := HPipeConfig.TCPF.TCPBufferSize
	if bufferSize <= 0 {
		bufferSize = 4096 // Default buffer size
	}

	for {
		// Check if we should stop
		tp.mutex.RLock()
		stopChan := tp.stopChan
		tp.mutex.RUnlock()

		select {
		case <-stopChan:
			return
		default:
			// Continue reading
		}

		// Read message length (4 bytes)
		lengthBytes := make([]byte, 4)
		if _, err := io.ReadFull(reader, lengthBytes); err != nil {
			if err != io.EOF {
				log.Printf("Error reading message length: %v", err)
			}
			return
		}

		// Convert bytes to uint32
		messageLength := binary.BigEndian.Uint32(lengthBytes)
		if messageLength > uint32(bufferSize) {
			log.Printf("Message too large: %d bytes", messageLength)
			return
		}

		// Read pipe name length (1 byte)
		pipeNameLengthBytes := make([]byte, 1)
		if _, err := io.ReadFull(reader, pipeNameLengthBytes); err != nil {
			log.Printf("Error reading pipe name length: %v", err)
			return
		}
		pipeNameLength := pipeNameLengthBytes[0]

		// Read pipe name
		pipeNameBytes := make([]byte, pipeNameLength)
		if _, err := io.ReadFull(reader, pipeNameBytes); err != nil {
			log.Printf("Error reading pipe name: %v", err)
			return
		}
		pipeName := string(pipeNameBytes)

		// Read message data
		messageBytes := make([]byte, messageLength)
		if _, err := io.ReadFull(reader, messageBytes); err != nil {
			log.Printf("Error reading message data: %v", err)
			return
		}

		// Process the message
		tp.processMessage(pipeName, messageBytes)
	}
}

// processMessage processes a received message for a specific pipe.
func (tp *TCPPacket) processMessage(pipe string, messageBytes []byte) {
	// Get the handler for this pipe
	tp.mutex.RLock()
	handler, exists := tp.handlers[pipe]
	tp.mutex.RUnlock()

	if !exists {
		log.Printf("No handler for pipe %s", pipe)
		return
	}

	// Decode the message
	var data interface{}
	if err := Decode(messageBytes, &data); err != nil {
		log.Printf("Failed to decode message for pipe %s: %v", pipe, err)
		return
	}

	// Process the message
	handler(data)
}

// connectClient connects to a TCP server as a client.
func (tp *TCPPacket) connectClient(tlsConfig *tls.Config, timeout time.Duration) error {
	// Get host and port from configuration with defaults
	host := HPipeConfig.TCPF.TCPHost
	if host == "" {
		host = "localhost" // Default host
	}

	port := HPipeConfig.TCPF.TCPPort
	if port <= 0 {
		port = 9000 // Default port
	}

	// Create address string with proper IPv6 support
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	// Connect to the server
	var conn net.Conn
	var err error

	if tlsConfig != nil {
		// TLS connection
		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		// Plain TCP connection
		conn, err = dialer.Dial("tcp", address)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to TCP server: %w", err)
	}

	// Configure keep-alive if enabled
	if tcpConn, ok := conn.(*net.TCPConn); ok && HPipeConfig.TCPF.TCPKeepAlive {
		period := time.Duration(HPipeConfig.TCPF.TCPKeepAlivePeriod) * time.Second
		if period == 0 {
			period = 30 * time.Second // Default period
		}
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(period)
	}

	// Store the connection
	tp.mutex.Lock()
	tp.conn = conn
	tp.running = true
	tp.mutex.Unlock()

	// Start reading messages in a goroutine
	go tp.readMessages()

	log.Printf("Connected to TCP server at %s", address)
	return nil
}

// readMessages reads messages from the TCP connection in client mode.
func (tp *TCPPacket) readMessages() {
	tp.mutex.RLock()
	conn := tp.conn
	stopChan := tp.stopChan
	tp.mutex.RUnlock()

	if conn == nil {
		log.Printf("TCP connection is nil, cannot read messages")
		return
	}

	// Create a buffered reader
	reader := bufio.NewReader(conn)

	// Set buffer size
	bufferSize := HPipeConfig.TCPF.TCPBufferSize
	if bufferSize <= 0 {
		bufferSize = 4096 // Default buffer size
	}

	defer conn.Close()

	for {
		// Check if we should stop
		select {
		case <-stopChan:
			return
		default:
			// Continue reading
		}

		// Read message length (4 bytes)
		lengthBytes := make([]byte, 4)
		if _, err := io.ReadFull(reader, lengthBytes); err != nil {
			if err != io.EOF {
				log.Printf("Error reading message length: %v", err)
			}
			return
		}

		// Convert bytes to uint32
		messageLength := binary.BigEndian.Uint32(lengthBytes)
		if messageLength > uint32(bufferSize) {
			log.Printf("Message too large: %d bytes", messageLength)
			return
		}

		// Read pipe name length (1 byte)
		pipeNameLengthBytes := make([]byte, 1)
		if _, err := io.ReadFull(reader, pipeNameLengthBytes); err != nil {
			log.Printf("Error reading pipe name length: %v", err)
			return
		}
		pipeNameLength := pipeNameLengthBytes[0]

		// Read pipe name
		pipeNameBytes := make([]byte, pipeNameLength)
		if _, err := io.ReadFull(reader, pipeNameBytes); err != nil {
			log.Printf("Error reading pipe name: %v", err)
			return
		}
		pipeName := string(pipeNameBytes)

		// Read message data
		messageBytes := make([]byte, messageLength)
		if _, err := io.ReadFull(reader, messageBytes); err != nil {
			log.Printf("Error reading message data: %v", err)
			return
		}

		// Process the message
		tp.processMessage(pipeName, messageBytes)
	}
}

// Dispatch publishes a message to the specified pipe.
func (tp *TCPPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection with read lock
	tp.mutex.RLock()
	running := tp.running
	mode := tp.mode
	tp.mutex.RUnlock()

	if !running {
		return fmt.Errorf("not connected to TCP server")
	}

	// Encode the data
	messageBytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create message with header
	// Format: [message length (4 bytes)][pipe name length (1 byte)][pipe name][message data]
	pipeNameBytes := []byte(pipe)
	if len(pipeNameBytes) > 255 {
		return fmt.Errorf("pipe name too long (max 255 bytes): %s", pipe)
	}

	// Calculate total message size
	messageLength := len(messageBytes)
	totalLength := 4 + 1 + len(pipeNameBytes) + messageLength

	// Create the complete message
	completeMessage := make([]byte, totalLength)

	// Write message length (4 bytes)
	binary.BigEndian.PutUint32(completeMessage[0:4], uint32(messageLength))

	// Write pipe name length (1 byte)
	completeMessage[4] = byte(len(pipeNameBytes))

	// Write pipe name
	copy(completeMessage[5:5+len(pipeNameBytes)], pipeNameBytes)

	// Write message data
	copy(completeMessage[5+len(pipeNameBytes):], messageBytes)

	// Send the message
	if mode == "client" {
		// Client mode - send to the server
		tp.mutex.RLock()
		conn := tp.conn
		tp.mutex.RUnlock()

		if conn == nil {
			return fmt.Errorf("not connected to TCP server")
		}

		if _, err := conn.Write(completeMessage); err != nil {
			return fmt.Errorf("failed to send message to TCP server: %w", err)
		}
	} else {
		// Server mode - send to all clients
		tp.mutex.RLock()
		clients := make(map[string]net.Conn, len(tp.clients))
		for addr, conn := range tp.clients {
			clients[addr] = conn
		}
		tp.mutex.RUnlock()

		if len(clients) == 0 {
			return fmt.Errorf("no connected clients to send message to")
		}

		// Send to all clients
		for addr, conn := range clients {
			if _, err := conn.Write(completeMessage); err != nil {
				log.Printf("Failed to send message to client %s: %v", addr, err)
				// Continue sending to other clients
			}
		}
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (tp *TCPPacket) Accept(pipe string, fn Process) error {
	// Validate connection with read lock
	tp.mutex.RLock()
	running := tp.running
	tp.mutex.RUnlock()

	if !running {
		return fmt.Errorf("not connected to TCP server")
	}

	// Register the handler
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	// Check if already subscribed
	if _, exists := tp.handlers[pipe]; exists {
		return fmt.Errorf("already subscribed to pipe %s", pipe)
	}

	// Store the handler
	tp.handlers[pipe] = fn

	log.Printf("Subscribed to TCP pipe %s", pipe)
	return nil
}

// Remove unsubscribes from the specified pipe.
func (tp *TCPPacket) Remove(pipe string) error {
	// Validate connection with read lock
	tp.mutex.RLock()
	running := tp.running
	tp.mutex.RUnlock()

	if !running {
		return fmt.Errorf("not connected to TCP server")
	}

	// Remove the handler
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	// Check if the pipe is subscribed
	if _, exists := tp.handlers[pipe]; !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Remove the handler
	delete(tp.handlers, pipe)

	log.Printf("Unsubscribed from TCP pipe %s", pipe)
	return nil
}

// Close terminates the TCP connection or stops the TCP server.
func (tp *TCPPacket) Close() {
	// Lock for thread safety
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	// Check if running
	if !tp.running {
		return
	}

	// Signal all goroutines to stop
	close(tp.stopChan)
	tp.stopChan = make(chan struct{}) // Create a new channel for future use

	// Close all client connections (server mode)
	for addr, conn := range tp.clients {
		conn.Close()
		log.Printf("Closed connection to client %s", addr)
	}
	tp.clients = make(map[string]net.Conn)

	// Close the listener (server mode)
	if tp.listener != nil {
		tp.listener.Close()
		tp.listener = nil
		log.Printf("Stopped TCP server")
	}

	// Close the connection (client mode)
	if tp.conn != nil {
		tp.conn.Close()
		tp.conn = nil
		log.Printf("Disconnected from TCP server")
	}

	// Clear handlers
	tp.handlers = make(map[string]Process)
	tp.running = false
}
