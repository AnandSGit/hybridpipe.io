// Package netchan provides network channels that extend Go's channel semantics across network boundaries.
package netchan

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// MessageType represents the type of a NetChan message.
type MessageType byte

const (
	// MessageHandshake is sent during channel establishment.
	MessageHandshake MessageType = iota + 1
	// MessageData contains a serialized value.
	MessageData
	// MessageClose indicates channel closure.
	MessageClose
	// MessageAck acknowledges receipt of a message.
	MessageAck
	// MessagePing is used for keepalive.
	MessagePing
	// MessagePong is the response to a ping.
	MessagePong
)

// Protocol constants
const (
	// ProtocolVersion is the current version of the NetChan protocol.
	ProtocolVersion = 1
	// DefaultBufferSize is the default buffer size for buffered channels.
	DefaultBufferSize = 10
	// DefaultTimeout is the default timeout for operations.
	DefaultTimeout = 30 * time.Second
	// MaxMessageSize is the maximum size of a message.
	MaxMessageSize = 16 * 1024 * 1024 // 16MB
	// HeaderSize is the size of the message header.
	HeaderSize = 10 // 1 + 4 + 1 + 4 (type + seq + flags + length)
)

// Flags for message headers
const (
	// FlagNone indicates no special flags.
	FlagNone byte = 0
	// FlagCompressed indicates the message is compressed.
	FlagCompressed byte = 1 << 0
	// FlagEncrypted indicates the message is encrypted.
	FlagEncrypted byte = 1 << 1
)

// Common errors
var (
	// ErrChannelClosed is returned when attempting to use a closed channel.
	ErrChannelClosed = errors.New("netchan: channel is closed")
	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("netchan: operation timed out")
	// ErrInvalidType is returned when attempting to send a value of the wrong type.
	ErrInvalidType = errors.New("netchan: invalid type")
	// ErrHandshakeFailed is returned when the handshake fails.
	ErrHandshakeFailed = errors.New("netchan: handshake failed")
	// ErrMessageTooLarge is returned when a message exceeds the maximum size.
	ErrMessageTooLarge = errors.New("netchan: message too large")
	// ErrConnectionClosed is returned when the connection is closed.
	ErrConnectionClosed = errors.New("netchan: connection closed")
)

// Options configures a NetChan.
type Options struct {
	// BufferSize is the size of the channel buffer.
	BufferSize int
	// Timeout is the timeout for operations.
	Timeout time.Duration
	// Reconnect enables automatic reconnection.
	Reconnect bool
	// ReconnectBackoff is the backoff policy for reconnection.
	ReconnectBackoff BackoffPolicy
	// Compression enables compression for messages.
	Compression bool
	// CompressionLevel is the compression level (0-9).
	CompressionLevel int
	// TLSConfig is the TLS configuration for secure connections.
	TLSConfig interface{} // *tls.Config
}

// BackoffPolicy defines the reconnection backoff policy.
type BackoffPolicy struct {
	// InitialDelay is the initial delay before reconnecting.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between reconnection attempts.
	MaxDelay time.Duration
	// Factor is the multiplier for the delay after each attempt.
	Factor float64
	// Jitter is the random jitter factor (0.0-1.0).
	Jitter float64
}

// DefaultOptions returns the default options.
func DefaultOptions() Options {
	return Options{
		BufferSize:       DefaultBufferSize,
		Timeout:          DefaultTimeout,
		Reconnect:        true,
		ReconnectBackoff: DefaultBackoffPolicy(),
		Compression:      false,
		CompressionLevel: 6,
		TLSConfig:        nil,
	}
}

// DefaultBackoffPolicy returns the default backoff policy.
func DefaultBackoffPolicy() BackoffPolicy {
	return BackoffPolicy{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Factor:       2.0,
		Jitter:       0.1,
	}
}

// Option is a function that configures Options.
type Option func(*Options)

// WithBufferSize sets the buffer size.
func WithBufferSize(size int) Option {
	return func(o *Options) {
		o.BufferSize = size
	}
}

// WithTimeout sets the operation timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

// WithReconnect enables or disables automatic reconnection.
func WithReconnect(enable bool) Option {
	return func(o *Options) {
		o.Reconnect = enable
	}
}

// WithReconnectBackoff sets the reconnection backoff policy.
func WithReconnectBackoff(policy BackoffPolicy) Option {
	return func(o *Options) {
		o.ReconnectBackoff = policy
	}
}

// WithCompression enables or disables compression.
func WithCompression(enable bool) Option {
	return func(o *Options) {
		o.Compression = enable
	}
}

// WithCompressionLevel sets the compression level.
func WithCompressionLevel(level int) Option {
	return func(o *Options) {
		o.CompressionLevel = level
	}
}

// WithTLS sets the TLS configuration.
func WithTLS(config interface{}) Option {
	return func(o *Options) {
		o.TLSConfig = config
	}
}

// Chan is a network channel.
type Chan[T any] struct {
	// Type information
	valueType reflect.Type

	// Channel state
	buffer    chan T
	closed    bool
	closeLock sync.RWMutex

	// Transport
	transport transportLayer

	// Protocol state
	nextSeq     uint32
	seqLock     sync.Mutex
	pendingRecv map[uint32]chan []byte
	pendingLock sync.RWMutex

	// Serialization
	serializer *Serializer

	// Configuration
	options Options

	// Control
	done     chan struct{}
	doneOnce sync.Once
}

// Make creates a new unbuffered network channel.
func Make[T any](prototype T, opts ...Option) *Chan[T] {
	options := DefaultOptions()
	options.BufferSize = 0

	for _, opt := range opts {
		opt(&options)
	}

	return makeChan(prototype, options)
}

// MakeBuffered creates a new buffered network channel.
func MakeBuffered[T any](prototype T, size int, opts ...Option) *Chan[T] {
	options := DefaultOptions()
	options.BufferSize = size

	for _, opt := range opts {
		opt(&options)
	}

	return makeChan(prototype, options)
}

// makeChan creates a new channel with the given options.
func makeChan[T any](prototype T, options Options) *Chan[T] {
	bufSize := options.BufferSize
	if bufSize < 0 {
		bufSize = 0
	}

	ch := &Chan[T]{
		valueType:   reflect.TypeOf(prototype),
		buffer:      make(chan T, bufSize),
		closed:      false,
		nextSeq:     1,
		pendingRecv: make(map[uint32]chan []byte),
		serializer:  NewSerializer(options.Compression, options.CompressionLevel),
		options:     options,
		done:        make(chan struct{}),
	}

	return ch
}

// Listen creates a new channel that listens for connections.
func Listen[T any](addr string, prototype T, opts ...Option) (*Chan[T], error) {
	ch := Make(prototype, opts...)

	// Create a transport
	transport := NewTransport(ch.options)
	ch.transport = transport

	// Set up message handling
	transport.SetMessageHandler(func(msg *Message) error {
		return ch.handleMessage(msg)
	})

	transport.SetCloseHandler(func(err error) {
		// Handle connection closure
		// For now, just log it
		// log.Printf("netchan: connection closed: %v", err)
	})

	// Start listening
	if err := transport.Listen(addr); err != nil {
		return nil, err
	}

	return ch, nil
}

// handleMessage processes an incoming message.
func (ch *Chan[T]) handleMessage(msg *Message) error {
	switch msg.Type {
	case MessageHandshake:
		// Handle handshake
		return ch.handleHandshake(msg)

	case MessageData:
		// Handle data message
		return ch.handleData(msg)

	case MessageClose:
		// Handle close message
		return ch.handleClose(msg)

	case MessageAck:
		// Handle acknowledgment
		return ch.handleAck(msg)

	case MessagePing:
		// Handle ping
		return ch.handlePing(msg)

	case MessagePong:
		// Handle pong
		return ch.handlePong(msg)

	default:
		return fmt.Errorf("netchan: unknown message type: %d", msg.Type)
	}
}

// handleHandshake processes a handshake message.
func (ch *Chan[T]) handleHandshake(msg *Message) error {
	// Decode the type information
	var typeInfo TypeInfo
	if err := decodeValue(msg.Payload, &typeInfo); err != nil {
		return fmt.Errorf("netchan: failed to decode type information: %w", err)
	}

	// Check if the type is compatible
	localTypeInfo := TypeInfoFromType(ch.valueType)
	if !localTypeInfo.Compatible(typeInfo) {
		return fmt.Errorf("netchan: incompatible types: local %s, remote %s", localTypeInfo.TypeName, typeInfo.TypeName)
	}

	// Send an acknowledgment
	ackMsg := NewAckMessage(msg.Sequence)
	if err := ch.transport.SendMessage(ackMsg); err != nil {
		return fmt.Errorf("netchan: failed to send handshake acknowledgment: %w", err)
	}

	return nil
}

// handleData processes a data message.
func (ch *Chan[T]) handleData(msg *Message) error {
	// Deserialize the data
	value, err := ch.serializer.Deserialize(msg.Payload, msg.Flags, ch.valueType)
	if err != nil {
		return fmt.Errorf("netchan: failed to deserialize value: %w", err)
	}

	// Convert to the correct type
	typedValue, ok := value.(T)
	if !ok {
		return fmt.Errorf("netchan: type mismatch: expected %s, got %T", ch.valueType, value)
	}

	// Check if the channel is closed
	ch.closeLock.RLock()
	if ch.closed {
		ch.closeLock.RUnlock()
		return ErrChannelClosed
	}
	ch.closeLock.RUnlock()

	// Send an acknowledgment
	ackMsg := NewAckMessage(msg.Sequence)
	if err := ch.transport.SendMessage(ackMsg); err != nil {
		return fmt.Errorf("netchan: failed to send acknowledgment: %w", err)
	}

	// Try to put the value in the buffer
	select {
	case ch.buffer <- typedValue:
		// Value sent to buffer
		return nil

	default:
		// Buffer is full, store the value for later retrieval
		// In a real implementation, we would need to handle this case
		// For now, just drop the message
		return nil
	}
}

// handleClose processes a close message.
func (ch *Chan[T]) handleClose(msg *Message) error {
	// Set the closed flag
	ch.closeLock.Lock()
	if ch.closed {
		ch.closeLock.Unlock()
		return nil // Already closed
	}
	ch.closed = true
	ch.closeLock.Unlock()

	// Signal shutdown
	ch.doneOnce.Do(func() {
		close(ch.done)
	})

	// Close the buffer channel
	close(ch.buffer)

	// Send an acknowledgment
	ackMsg := NewAckMessage(msg.Sequence)
	_ = ch.transport.SendMessage(ackMsg)

	return nil
}

// handleAck processes an acknowledgment message.
func (ch *Chan[T]) handleAck(msg *Message) error {
	// Get the pending receive channel for this sequence number
	ch.pendingLock.RLock()
	respCh, exists := ch.pendingRecv[msg.Sequence]
	ch.pendingLock.RUnlock()

	if !exists {
		// No pending receive for this sequence number
		return nil
	}

	// Signal that the message was acknowledged
	select {
	case respCh <- nil: // Use nil to indicate acknowledgment
	default:
		// Channel is full or closed, ignore
	}

	// Remove the pending receive
	ch.pendingLock.Lock()
	delete(ch.pendingRecv, msg.Sequence)
	ch.pendingLock.Unlock()

	return nil
}

// handlePing processes a ping message.
func (ch *Chan[T]) handlePing(msg *Message) error {
	// Send a pong response
	pongMsg := NewPongMessage(msg.Sequence)
	return ch.transport.SendMessage(pongMsg)
}

// handlePong processes a pong message.
func (ch *Chan[T]) handlePong(_ *Message) error {
	// Nothing to do for pong messages, they just confirm the connection is alive
	return nil
}

// Dial connects to a listening channel.
func Dial[T any](addr string, prototype T, opts ...Option) (*Chan[T], error) {
	ch := Make(prototype, opts...)

	// Create a transport
	transport := NewTransport(ch.options)
	ch.transport = transport

	// Set up message handling
	transport.SetMessageHandler(func(msg *Message) error {
		return ch.handleMessage(msg)
	})

	transport.SetCloseHandler(func(err error) {
		// Handle connection closure
		// For now, just log it
		// log.Printf("netchan: connection closed: %v", err)
	})

	// Connect to the remote endpoint
	if err := transport.Dial(addr); err != nil {
		return nil, err
	}

	// Send handshake
	if err := ch.sendHandshake(); err != nil {
		ch.Close()
		return nil, err
	}

	return ch, nil
}

// sendHandshake sends a handshake message.
func (ch *Chan[T]) sendHandshake() error {
	// Create type information
	typeInfo := TypeInfoFromType(ch.valueType)

	// Create handshake message
	seq := ch.nextSequence()
	msg, err := NewHandshakeMessage(seq, typeInfo)
	if err != nil {
		return fmt.Errorf("netchan: failed to create handshake message: %w", err)
	}

	// Send the message
	return ch.transport.SendMessage(msg)
}

// nextSequence returns the next sequence number.
func (ch *Chan[T]) nextSequence() uint32 {
	ch.seqLock.Lock()
	defer ch.seqLock.Unlock()

	seq := ch.nextSeq
	ch.nextSeq++
	return seq
}

// Send sends a value on the channel.
func (ch *Chan[T]) Send(value T) error {
	// Check if the channel is closed
	ch.closeLock.RLock()
	if ch.closed {
		ch.closeLock.RUnlock()
		return ErrChannelClosed
	}
	ch.closeLock.RUnlock()

	// Serialize the value
	data, flags, err := ch.serializer.Serialize(value)
	if err != nil {
		return fmt.Errorf("netchan: failed to serialize value: %w", err)
	}

	// Create a data message
	seq := ch.nextSequence()
	msg := NewMessage(MessageData, seq, flags, data)

	// Send the message
	return ch.transport.SendMessage(msg)
}

// TrySend attempts to send a value on the channel without blocking.
func (ch *Chan[T]) TrySend(value T) (bool, error) {
	// This is a simplified implementation that just calls Send
	// In a real implementation, we would need to check if the channel is full
	err := ch.Send(value)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Recv receives a value from the channel.
func (ch *Chan[T]) Recv() (T, bool, error) {
	var zero T

	// Check if the channel is closed
	ch.closeLock.RLock()
	if ch.closed {
		ch.closeLock.RUnlock()
		return zero, false, ErrChannelClosed
	}
	ch.closeLock.RUnlock()

	// Create a channel to receive the response
	respCh := make(chan recvResult[T], 1)

	// Start a goroutine to wait for a message
	go func() {
		// Wait for a message from the buffer or network
		select {
		case val, ok := <-ch.buffer:
			if !ok {
				// Channel closed
				respCh <- recvResult[T]{zero, false, ErrChannelClosed}
				return
			}
			// Got a value from the buffer
			respCh <- recvResult[T]{val, true, nil}

		case <-ch.done:
			// Channel closed
			respCh <- recvResult[T]{zero, false, ErrChannelClosed}
		}
	}()

	// Wait for the result or timeout
	select {
	case result := <-respCh:
		return result.value, result.ok, result.err

	case <-time.After(ch.options.Timeout):
		return zero, false, ErrTimeout
	}
}

// recvResult holds the result of a receive operation.
type recvResult[T any] struct {
	value T
	ok    bool
	err   error
}

// TryRecv attempts to receive a value from the channel without blocking.
func (ch *Chan[T]) TryRecv() (T, bool, error) {
	var zero T

	// Check if the channel is closed
	ch.closeLock.RLock()
	if ch.closed {
		ch.closeLock.RUnlock()
		return zero, false, ErrChannelClosed
	}
	ch.closeLock.RUnlock()

	// Try to receive from the buffer
	select {
	case val, ok := <-ch.buffer:
		if !ok {
			// Channel closed
			return zero, false, ErrChannelClosed
		}
		// Got a value from the buffer
		return val, true, nil

	default:
		// No value available
		return zero, false, nil
	}
}

// Close closes the channel.
func (ch *Chan[T]) Close() error {
	// Set the closed flag
	ch.closeLock.Lock()
	if ch.closed {
		ch.closeLock.Unlock()
		return nil // Already closed
	}
	ch.closed = true
	ch.closeLock.Unlock()

	// Signal shutdown
	ch.doneOnce.Do(func() {
		close(ch.done)
	})

	// Close the buffer channel
	close(ch.buffer)

	// Send close message to the remote endpoint
	if ch.transport != nil {
		// Create a close message
		seq := ch.nextSequence()
		msg := NewCloseMessage(seq)

		// Send the message (ignore errors)
		_ = ch.transport.SendMessage(msg)

		// Close the transport
		_ = ch.transport.Close()
	}

	return nil
}

// IsClosed returns whether the channel is closed.
func (ch *Chan[T]) IsClosed() bool {
	ch.closeLock.RLock()
	defer ch.closeLock.RUnlock()
	return ch.closed
}

// Note: Select functionality is implemented in select.go
