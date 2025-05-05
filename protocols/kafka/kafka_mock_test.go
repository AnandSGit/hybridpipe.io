package kafka

import (
	"context"
	"errors"
	"sync"
	"time"
	"unsafe"

	"github.com/segmentio/kafka-go"
)

// MockReader is a mock implementation of kafka.Reader for testing
type MockReader struct {
	messages     []kafka.Message
	index        int
	closed       bool
	readError    error
	closeError   error
	mu           sync.Mutex
	readCallback func(ctx context.Context) (kafka.Message, error)
}

// NewMockReader creates a new mock reader with predefined messages
func NewMockReader(messages []kafka.Message) *MockReader {
	return &MockReader{
		messages: messages,
		index:    0,
		closed:   false,
	}
}

// ReadMessage implements the Reader interface
func (m *MockReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If a custom callback is set, use it
	if m.readCallback != nil {
		return m.readCallback(ctx)
	}

	// Check if reader is closed
	if m.closed {
		return kafka.Message{}, errors.New("reader closed")
	}

	// Check for custom error
	if m.readError != nil {
		return kafka.Message{}, m.readError
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	default:
		// Continue
	}

	// Check if we have messages
	if len(m.messages) == 0 {
		// Simulate waiting for messages
		select {
		case <-ctx.Done():
			return kafka.Message{}, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return kafka.Message{}, context.DeadlineExceeded
		}
	}

	// Check if we've read all messages
	if m.index >= len(m.messages) {
		// Simulate waiting for more messages
		select {
		case <-ctx.Done():
			return kafka.Message{}, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return kafka.Message{}, context.DeadlineExceeded
		}
	}

	// Return the next message
	msg := m.messages[m.index]
	m.index++
	return msg, nil
}

// SetReadCallback sets a custom callback for ReadMessage
func (m *MockReader) SetReadCallback(callback func(ctx context.Context) (kafka.Message, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readCallback = callback
}

// SetReadError sets an error to be returned by ReadMessage
func (m *MockReader) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readError = err
}

// SetCloseError sets an error to be returned by Close
func (m *MockReader) SetCloseError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeError = err
}

// Close implements the Reader interface
func (m *MockReader) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeError
}

// MockWriter is a mock implementation of kafka.Writer for testing
type MockWriter struct {
	messages      []kafka.Message
	writeError    error
	closeError    error
	closed        bool
	mu            sync.Mutex
	writeCallback func(ctx context.Context, msgs ...kafka.Message) error
}

// NewMockWriter creates a new mock writer
func NewMockWriter() *MockWriter {
	return &MockWriter{
		messages: make([]kafka.Message, 0),
		closed:   false,
	}
}

// WriteMessages implements the Writer interface
func (m *MockWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If a custom callback is set, use it
	if m.writeCallback != nil {
		return m.writeCallback(ctx, msgs...)
	}

	// Check if writer is closed
	if m.closed {
		return errors.New("writer closed")
	}

	// Check for custom error
	if m.writeError != nil {
		return m.writeError
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue
	}

	// Store the messages
	m.messages = append(m.messages, msgs...)
	return nil
}

// SetWriteCallback sets a custom callback for WriteMessages
func (m *MockWriter) SetWriteCallback(callback func(ctx context.Context, msgs ...kafka.Message) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeCallback = callback
}

// SetWriteError sets an error to be returned by WriteMessages
func (m *MockWriter) SetWriteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeError = err
}

// SetCloseError sets an error to be returned by Close
func (m *MockWriter) SetCloseError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeError = err
}

// Close implements the Writer interface
func (m *MockWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeError
}

// GetMessages returns the messages written to the writer
func (m *MockWriter) GetMessages() []kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messages
}

// ReaderConstructor is a function type for creating readers
type ReaderConstructor func(kafka.ReaderConfig) *kafka.Reader

// WriterConstructor is a function type for creating writers
type WriterConstructor func(kafka.WriterConfig) *kafka.Writer

// MockReaderConstructor returns a mock reader constructor
func MockReaderConstructor(mockReader *MockReader) ReaderConstructor {
	return func(config kafka.ReaderConfig) *kafka.Reader {
		// Type assertion to convert MockReader to kafka.Reader
		// This requires MockReader to fully implement the kafka.Reader interface
		return (*kafka.Reader)(unsafe.Pointer(mockReader))
	}
}

// MockWriterConstructor returns a mock writer constructor
func MockWriterConstructor(mockWriter *MockWriter) WriterConstructor {
	return func(config kafka.WriterConfig) *kafka.Writer {
		return (*kafka.Writer)(unsafe.Pointer(mockWriter))
	}
}
