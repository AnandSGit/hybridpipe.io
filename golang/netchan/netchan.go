// Package netchan provides Go channel capability over a network between different processes.
// It implements Go's native channel operators (<- and ->) for sending messages over the network.
package netchan

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"hybridpipe.io/core"
)

// Config holds the configuration for the NetChan library.
type Config struct {
	// Network is the network type (e.g., "tcp", "udp")
	Network string
	// Address is the address to listen on or connect to
	Address string
	// BufferSize is the size of the channel buffer
	BufferSize int
	// Timeout is the timeout for network operations in milliseconds
	Timeout int
	// KeepAlive enables keep-alive for TCP connections
	KeepAlive bool
	// KeepAlivePeriod is the keep-alive period in seconds
	KeepAlivePeriod int
	// ReconnectDelay is the delay between reconnection attempts in milliseconds
	ReconnectDelay int
	// MaxReconnectAttempts is the maximum number of reconnection attempts
	MaxReconnectAttempts int
}

// DefaultConfig returns a default configuration for the NetChan library.
func DefaultConfig() *Config {
	return &Config{
		Network:              "tcp",
		Address:              "localhost:9000",
		BufferSize:           10,
		Timeout:              5000,
		KeepAlive:            true,
		KeepAlivePeriod:      60,
		ReconnectDelay:       1000,
		MaxReconnectAttempts: 5,
	}
}

// NetChan represents a network channel.
type NetChan struct {
	// config is the configuration for the NetChan
	config *Config
	// listener is the network listener (server mode)
	listener net.Listener
	// conn is the network connection (client mode)
	conn net.Conn
	// mode can be "server" or "client"
	mode string
	// sendChans maps channel names to send channels
	sendChans map[string]chan interface{}
	// recvChans maps channel names to receive channels
	recvChans map[string]chan interface{}
	// clients maps client addresses to their connections (server mode)
	clients map[string]net.Conn
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// ctx is the context for the NetChan
	ctx context.Context
	// cancel is the cancel function for the context
	cancel context.CancelFunc
	// running indicates if the NetChan is running
	running bool
}

// NewServer creates a new NetChan server.
func NewServer(config *Config) (*NetChan, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	nc := &NetChan{
		config:    config,
		mode:      "server",
		sendChans: make(map[string]chan interface{}),
		recvChans: make(map[string]chan interface{}),
		clients:   make(map[string]net.Conn),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Create listener
	listener, err := net.Listen(config.Network, config.Address)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	nc.listener = listener
	nc.running = true

	// Start accepting connections
	go nc.acceptConnections()

	log.Printf("NetChan server started on %s", config.Address)
	return nc, nil
}

// NewClient creates a new NetChan client.
func NewClient(config *Config) (*NetChan, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	nc := &NetChan{
		config:    config,
		mode:      "client",
		sendChans: make(map[string]chan interface{}),
		recvChans: make(map[string]chan interface{}),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Connect to the server
	if err := nc.connect(); err != nil {
		cancel()
		return nil, err
	}

	log.Printf("NetChan client connected to %s", config.Address)
	return nc, nil
}

// connect connects to the server.
func (nc *NetChan) connect() error {
	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: time.Duration(nc.config.Timeout) * time.Millisecond,
	}

	// Connect to the server
	conn, err := dialer.Dial(nc.config.Network, nc.config.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Set keep alive if enabled
	if nc.config.KeepAlive {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			if nc.config.KeepAlivePeriod > 0 {
				tcpConn.SetKeepAlivePeriod(time.Duration(nc.config.KeepAlivePeriod) * time.Second)
			}
		}
	}

	// Store the connection
	nc.mutex.Lock()
	nc.conn = conn
	nc.running = true
	nc.mutex.Unlock()

	// Start reading messages
	go nc.readMessages(conn)

	return nil
}

// acceptConnections accepts incoming connections.
func (nc *NetChan) acceptConnections() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in acceptConnections: %v", r)
		}
	}()

	for {
		// Check if server is still running
		nc.mutex.RLock()
		listener := nc.listener
		running := nc.running
		nc.mutex.RUnlock()

		if !running || listener == nil {
			log.Printf("NetChan server stopped")
			return
		}

		// Accept a connection
		conn, err := listener.Accept()
		if err != nil {
			// Check if server is still running
			nc.mutex.RLock()
			running := nc.running
			nc.mutex.RUnlock()

			if !running {
				return
			}

			log.Printf("Error accepting connection: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Set keep alive if enabled
		if nc.config.KeepAlive {
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				tcpConn.SetKeepAlive(true)
				if nc.config.KeepAlivePeriod > 0 {
					tcpConn.SetKeepAlivePeriod(time.Duration(nc.config.KeepAlivePeriod) * time.Second)
				}
			}
		}

		// Store the connection
		clientAddr := conn.RemoteAddr().String()
		nc.mutex.Lock()
		nc.clients[clientAddr] = conn
		nc.mutex.Unlock()

		log.Printf("Accepted connection from %s", clientAddr)

		// Handle the connection
		go nc.readMessages(conn)
	}
}

// readMessages reads messages from a connection.
func (nc *NetChan) readMessages(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in readMessages: %v", r)
		}
		conn.Close()

		// Remove the connection from the clients map
		if nc.mode == "server" {
			clientAddr := conn.RemoteAddr().String()
			nc.mutex.Lock()
			delete(nc.clients, clientAddr)
			nc.mutex.Unlock()

			log.Printf("Closed connection from %s", clientAddr)
		} else {
			// Client mode: try to reconnect
			nc.reconnect()
		}
	}()

	// Buffer for reading messages
	buffer := make([]byte, 4096)

	for {
		// Check if still running
		nc.mutex.RLock()
		running := nc.running
		nc.mutex.RUnlock()

		if !running {
			return
		}

		// Set read deadline
		if nc.config.Timeout > 0 {
			conn.SetReadDeadline(time.Now().Add(time.Duration(nc.config.Timeout) * time.Millisecond))
		}

		// Read message
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from connection: %v", err)
			return
		}

		// Process the message
		if n > 0 {
			// Decode the message
			var msg Message
			if err := core.Decode(buffer[:n], &msg); err != nil {
				log.Printf("Error decoding message: %v", err)
				continue
			}

			// Process the message based on its type
			switch msg.Type {
			case MessageTypeSend:
				nc.handleSendMessage(msg)
			case MessageTypeReceive:
				nc.handleReceiveMessage(msg, conn)
			case MessageTypeClose:
				nc.handleCloseMessage(msg)
			default:
				log.Printf("Unknown message type: %d", msg.Type)
			}
		}
	}
}

// Message represents a network message.
type Message struct {
	// Type is the message type
	Type int
	// Channel is the channel name
	Channel string
	// Data is the message data
	Data interface{}
}

// MessageType constants
const (
	MessageTypeSend    = 1
	MessageTypeReceive = 2
	MessageTypeClose   = 3
)

// handleSendMessage handles a send message.
func (nc *NetChan) handleSendMessage(msg Message) {
	// Get or create receive channel for the channel name
	nc.mutex.Lock()
	ch, exists := nc.recvChans[msg.Channel]
	if !exists {
		ch = make(chan interface{}, nc.config.BufferSize)
		nc.recvChans[msg.Channel] = ch
	}
	nc.mutex.Unlock()

	// Send the data to the channel
	select {
	case ch <- msg.Data:
		// Data sent successfully
	default:
		// Channel is full, log and drop the message
		log.Printf("Channel %s is full, dropping message", msg.Channel)
	}
}

// handleReceiveMessage handles a receive message.
func (nc *NetChan) handleReceiveMessage(msg Message, conn net.Conn) {
	// Get the send channel for the channel name
	nc.mutex.RLock()
	ch, exists := nc.sendChans[msg.Channel]
	nc.mutex.RUnlock()

	if !exists {
		// No data available for this channel
		return
	}

	// Try to receive data from the channel
	select {
	case data := <-ch:
		// Create a send message
		sendMsg := Message{
			Type:    MessageTypeSend,
			Channel: msg.Channel,
			Data:    data,
		}

		// Encode the message
		encoded, err := core.Encode(sendMsg)
		if err != nil {
			log.Printf("Error encoding message: %v", err)
			return
		}

		// Send the message
		if _, err := conn.Write(encoded); err != nil {
			log.Printf("Error writing to connection: %v", err)
			return
		}
	default:
		// No data available, do nothing
	}
}

// handleCloseMessage handles a close message.
func (nc *NetChan) handleCloseMessage(msg Message) {
	// Close the channel
	nc.mutex.Lock()
	if ch, exists := nc.recvChans[msg.Channel]; exists {
		close(ch)
		delete(nc.recvChans, msg.Channel)
	}
	nc.mutex.Unlock()
}

// reconnect attempts to reconnect to the server.
func (nc *NetChan) reconnect() {
	if nc.mode != "client" {
		return
	}

	// Set running to false
	nc.mutex.Lock()
	nc.running = false
	nc.mutex.Unlock()

	// Try to reconnect
	for i := 0; i < nc.config.MaxReconnectAttempts; i++ {
		log.Printf("Attempting to reconnect to %s (attempt %d/%d)", nc.config.Address, i+1, nc.config.MaxReconnectAttempts)

		// Wait before reconnecting
		time.Sleep(time.Duration(nc.config.ReconnectDelay) * time.Millisecond)

		// Try to connect
		if err := nc.connect(); err != nil {
			log.Printf("Reconnection attempt failed: %v", err)
			continue
		}

		log.Printf("Reconnected to %s", nc.config.Address)
		return
	}

	log.Printf("Failed to reconnect after %d attempts", nc.config.MaxReconnectAttempts)
}

// Chan creates a new network channel with the specified name and type.
func (nc *NetChan) Chan(name string) *Channel {
	return &Channel{
		name:    name,
		netChan: nc,
	}
}

// Channel represents a network channel.
type Channel struct {
	// name is the channel name
	name string
	// netChan is the parent NetChan
	netChan *NetChan
}

// SendOp implements the send operator (ch <- value).
func (ch *Channel) SendOp(value interface{}) {
	err := ch.Send(value)
	if err != nil {
		log.Printf("Error sending to channel %s: %v", ch.name, err)
	}
}

// RecvOp implements the receive operator (value := <-ch).
func (ch *Channel) RecvOp() interface{} {
	value, err := ch.Receive()
	if err != nil {
		log.Printf("Error receiving from channel %s: %v", ch.name, err)
		return nil
	}
	return value
}

// Send sends a value to the channel.
func (ch *Channel) Send(value interface{}) error {
	// Get or create send channel for the channel name
	ch.netChan.mutex.Lock()
	sendCh, exists := ch.netChan.sendChans[ch.name]
	if !exists {
		sendCh = make(chan interface{}, ch.netChan.config.BufferSize)
		ch.netChan.sendChans[ch.name] = sendCh
	}
	ch.netChan.mutex.Unlock()

	// Send the value to the channel
	select {
	case sendCh <- value:
		// Value sent successfully
	default:
		// Channel is full
		return fmt.Errorf("channel %s is full", ch.name)
	}

	// Create a send message
	msg := Message{
		Type:    MessageTypeSend,
		Channel: ch.name,
		Data:    value,
	}

	// Encode the message
	encoded, err := core.Encode(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Send the message
	if ch.netChan.mode == "client" {
		// Client mode: send to the server
		ch.netChan.mutex.RLock()
		conn := ch.netChan.conn
		ch.netChan.mutex.RUnlock()

		if conn == nil {
			return fmt.Errorf("not connected to server")
		}

		if _, err := conn.Write(encoded); err != nil {
			return fmt.Errorf("failed to write to connection: %w", err)
		}
	} else {
		// Server mode: send to all clients
		ch.netChan.mutex.RLock()
		clients := make([]net.Conn, 0, len(ch.netChan.clients))
		for _, conn := range ch.netChan.clients {
			clients = append(clients, conn)
		}
		ch.netChan.mutex.RUnlock()

		if len(clients) == 0 {
			return fmt.Errorf("no clients connected")
		}

		// Send to all clients
		for _, conn := range clients {
			if _, err := conn.Write(encoded); err != nil {
				log.Printf("Failed to write to client %s: %v", conn.RemoteAddr(), err)
			}
		}
	}

	return nil
}

// Receive receives a value from the channel.
func (ch *Channel) Receive() (interface{}, error) {
	// Get or create receive channel for the channel name
	ch.netChan.mutex.Lock()
	recvCh, exists := ch.netChan.recvChans[ch.name]
	if !exists {
		recvCh = make(chan interface{}, ch.netChan.config.BufferSize)
		ch.netChan.recvChans[ch.name] = recvCh
	}
	ch.netChan.mutex.Unlock()

	// Create a receive message
	msg := Message{
		Type:    MessageTypeReceive,
		Channel: ch.name,
	}

	// Encode the message
	encoded, err := core.Encode(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	// Send the message
	if ch.netChan.mode == "client" {
		// Client mode: send to the server
		ch.netChan.mutex.RLock()
		conn := ch.netChan.conn
		ch.netChan.mutex.RUnlock()

		if conn == nil {
			return nil, fmt.Errorf("not connected to server")
		}

		if _, err := conn.Write(encoded); err != nil {
			return nil, fmt.Errorf("failed to write to connection: %w", err)
		}
	} else {
		// Server mode: send to all clients
		ch.netChan.mutex.RLock()
		clients := make([]net.Conn, 0, len(ch.netChan.clients))
		for _, conn := range ch.netChan.clients {
			clients = append(clients, conn)
		}
		ch.netChan.mutex.RUnlock()

		if len(clients) == 0 {
			return nil, fmt.Errorf("no clients connected")
		}

		// Send to all clients
		for _, conn := range clients {
			if _, err := conn.Write(encoded); err != nil {
				log.Printf("Failed to write to client %s: %v", conn.RemoteAddr(), err)
			}
		}
	}

	// Wait for a value
	select {
	case value := <-recvCh:
		return value, nil
	case <-time.After(time.Duration(ch.netChan.config.Timeout) * time.Millisecond):
		return nil, fmt.Errorf("timeout waiting for value")
	case <-ch.netChan.ctx.Done():
		return nil, fmt.Errorf("NetChan is shutting down")
	}
}

// Close closes the channel.
func (ch *Channel) Close() error {
	// Create a close message
	msg := Message{
		Type:    MessageTypeClose,
		Channel: ch.name,
	}

	// Encode the message
	encoded, err := core.Encode(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Send the message
	if ch.netChan.mode == "client" {
		// Client mode: send to the server
		ch.netChan.mutex.RLock()
		conn := ch.netChan.conn
		ch.netChan.mutex.RUnlock()

		if conn == nil {
			return fmt.Errorf("not connected to server")
		}

		if _, err := conn.Write(encoded); err != nil {
			return fmt.Errorf("failed to write to connection: %w", err)
		}
	} else {
		// Server mode: send to all clients
		ch.netChan.mutex.RLock()
		clients := make([]net.Conn, 0, len(ch.netChan.clients))
		for _, conn := range ch.netChan.clients {
			clients = append(clients, conn)
		}
		ch.netChan.mutex.RUnlock()

		if len(clients) == 0 {
			return fmt.Errorf("no clients connected")
		}

		// Send to all clients
		for _, conn := range clients {
			if _, err := conn.Write(encoded); err != nil {
				log.Printf("Failed to write to client %s: %v", conn.RemoteAddr(), err)
			}
		}
	}

	// Close and remove the channels
	ch.netChan.mutex.Lock()
	if sendCh, exists := ch.netChan.sendChans[ch.name]; exists {
		close(sendCh)
		delete(ch.netChan.sendChans, ch.name)
	}
	if recvCh, exists := ch.netChan.recvChans[ch.name]; exists {
		close(recvCh)
		delete(ch.netChan.recvChans, ch.name)
	}
	ch.netChan.mutex.Unlock()

	return nil
}

// Close closes the NetChan.
func (nc *NetChan) Close() error {
	// Set running to false
	nc.mutex.Lock()
	nc.running = false
	nc.mutex.Unlock()

	// Cancel the context
	nc.cancel()

	// Close all channels
	nc.mutex.Lock()
	for name, ch := range nc.sendChans {
		close(ch)
		delete(nc.sendChans, name)
	}
	for name, ch := range nc.recvChans {
		close(ch)
		delete(nc.recvChans, name)
	}
	nc.mutex.Unlock()

	// Close all connections
	if nc.mode == "server" {
		// Close the listener
		if nc.listener != nil {
			nc.listener.Close()
		}

		// Close all client connections
		nc.mutex.Lock()
		for addr, conn := range nc.clients {
			conn.Close()
			delete(nc.clients, addr)
		}
		nc.mutex.Unlock()
	} else {
		// Close the connection to the server
		nc.mutex.Lock()
		if nc.conn != nil {
			nc.conn.Close()
			nc.conn = nil
		}
		nc.mutex.Unlock()
	}

	return nil
}
