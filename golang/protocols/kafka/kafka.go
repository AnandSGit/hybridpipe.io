// Package kafka provides an implementation of the HybridPipe interface for Kafka messaging system.
package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"hybridpipe.io/core"
)

// Packet implements the HybridPipe interface for Kafka messaging system.
type Packet struct {
	// Readers maps pipe names to their Kafka readers
	Readers map[string]*kafka.Reader
	// Writers maps pipe names to their Kafka writers
	Writers map[string]*kafka.Writer
	// DialerConn is the Kafka connection dialer
	DialerConn *kafka.Dialer
	// Server is the Kafka server address with port
	Server string
	// readerCancel maps pipe names to their context cancel functions
	readerCancel map[string]context.CancelFunc
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// config holds the Kafka configuration
	config *core.KafkaConfig
}

// New creates a new Kafka packet with the specified configuration.
func New(config *core.KafkaConfig) *Packet {
	return &Packet{
		Readers:      make(map[string]*kafka.Reader),
		Writers:      make(map[string]*kafka.Writer),
		readerCancel: make(map[string]context.CancelFunc),
		config:       config,
	}
}

// createTLSConfig creates a TLS configuration from certificate files.
func createTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, nil
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
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
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}

// Connect establishes a connection to the Kafka server using the configuration.
func (kp *Packet) Connect() error {
	// Get configuration if not already set
	if kp.config == nil {
		config, err := core.GetProtocolConfig(core.KAFKA)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		kafkaConfig := config.(core.KafkaConfig)
		kp.config = &kafkaConfig
	}

	// Build the Kafka server URL with port
	kp.Server = fmt.Sprintf("%s:%d", kp.config.KServer, kp.config.KLport)

	// Create TLS configuration if certificates are provided
	tlsConfig, err := createTLSConfig(
		kp.config.KAFKACertFile,
		kp.config.KAFKAKeyFile,
		kp.config.KAFKACAFile,
	)
	if err != nil {
		return fmt.Errorf("failed to create TLS configuration: %w", err)
	}

	// Get hostname for client ID
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	// Create Kafka dialer
	kp.DialerConn = &kafka.Dialer{
		Timeout:   time.Duration(kp.config.KTimeout) * time.Second,
		DualStack: true,
		TLS:       tlsConfig,
		ClientID:  fmt.Sprintf("hybridpipe-%s-%d", hostname, os.Getpid()),
	}

	// Initialize maps
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	if kp.Readers == nil {
		kp.Readers = make(map[string]*kafka.Reader)
	}
	if kp.Writers == nil {
		kp.Writers = make(map[string]*kafka.Writer)
	}
	if kp.readerCancel == nil {
		kp.readerCancel = make(map[string]context.CancelFunc)
	}

	log.Printf("Connected to Kafka server at %s", kp.Server)
	return nil
}

// Dispatch sends a message to the specified pipe.
func (kp *Packet) Dispatch(pipe string, data any) error {
	// Validate connection with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	kp.mutex.RUnlock()

	// Get or create writer for the pipe
	kp.mutex.Lock()
	writer, exists := kp.Writers[pipe]
	if !exists {
		writer = kafka.NewWriter(kafka.WriterConfig{
			Brokers:       []string{kp.Server},
			Topic:         pipe,
			Balancer:      &kafka.LeastBytes{},
			BatchSize:     1,
			QueueCapacity: 10,
			Async:         true,
			Dialer:        kp.DialerConn,
			// Ensure at least one broker acknowledges the message
			// Note: RequiredAcks is an int in older versions of kafka-go
			RequiredAcks: 1,
		})
		kp.Writers[pipe] = writer
	}
	kp.mutex.Unlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(pipe),
		Value: bytes,
		Time:  time.Now(),
	}

	// Write the message
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(kp.config.KTimeout)*time.Second)
	defer cancel()

	if err := writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write message to pipe %s: %w", pipe, err)
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (kp *Packet) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Validate connection with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	kp.mutex.RUnlock()

	// Get or create writer for the pipe
	kp.mutex.Lock()
	writer, exists := kp.Writers[pipe]
	if !exists {
		writer = kafka.NewWriter(kafka.WriterConfig{
			Brokers:       []string{kp.Server},
			Topic:         pipe,
			Balancer:      &kafka.LeastBytes{},
			BatchSize:     1,
			QueueCapacity: 10,
			Async:         true,
			Dialer:        kp.DialerConn,
			RequiredAcks:  1,
		})
		kp.Writers[pipe] = writer
	}
	kp.mutex.Unlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(pipe),
		Value: bytes,
		Time:  time.Now(),
	}

	// Write the message with the provided context
	if err := writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (kp *Packet) Accept(pipe string, fn func(any)) error {
	// Create a wrapper that converts the function to the expected type
	wrapper := func(data []byte) error {
		var decoded any
		if err := core.Decode(data, &decoded); err != nil {
			return err
		}
		fn(decoded)
		return nil
	}

	return kp.Subscribe(pipe, wrapper)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (kp *Packet) Subscribe(pipe string, callback core.Process) error {
	// Validate connection with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	kp.mutex.RUnlock()

	// Lock for thread safety
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	// Cancel existing reader if any
	if cancel, exists := kp.readerCancel[pipe]; exists {
		cancel()
		delete(kp.readerCancel, pipe)
	}

	// Close existing reader if any
	if reader, exists := kp.Readers[pipe]; exists {
		reader.Close()
		delete(kp.Readers, pipe)
	}

	// Create a new reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kp.Server},
		Topic:          pipe,
		GroupID:        fmt.Sprintf("hybridpipe-%s", pipe),
		MinBytes:       10,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		Dialer:         kp.DialerConn,
	})

	// Store the reader
	kp.Readers[pipe] = reader

	// Create a context with cancel function for the reader goroutine
	ctx, cancel := context.WithCancel(context.Background())
	kp.readerCancel[pipe] = cancel

	// Start a goroutine to handle messages
	go kp.readMessages(ctx, pipe, reader, callback)

	log.Printf("Subscribed to Kafka pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (kp *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Validate connection with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	kp.mutex.RUnlock()

	// Lock for thread safety
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	// Cancel existing reader if any
	if cancel, exists := kp.readerCancel[pipe]; exists {
		cancel()
		delete(kp.readerCancel, pipe)
	}

	// Close existing reader if any
	if reader, exists := kp.Readers[pipe]; exists {
		reader.Close()
		delete(kp.Readers, pipe)
	}

	// Create a new reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kp.Server},
		Topic:          pipe,
		GroupID:        fmt.Sprintf("hybridpipe-%s", pipe),
		MinBytes:       10,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		Dialer:         kp.DialerConn,
	})

	// Store the reader
	kp.Readers[pipe] = reader

	// Create a context with cancel function for the reader goroutine
	readerCtx, cancel := context.WithCancel(ctx)
	kp.readerCancel[pipe] = cancel

	// Start a goroutine to handle messages
	go kp.readMessages(readerCtx, pipe, reader, callback)

	log.Printf("Subscribed to Kafka pipe %s with context", pipe)
	return nil
}

// readMessages reads messages from Kafka in a loop until the context is canceled.
func (kp *Packet) readMessages(ctx context.Context, pipe string, reader *kafka.Reader, callback core.Process) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in Kafka reader for pipe %s: %v", pipe, r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping Kafka reader for pipe %s", pipe)
			return
		default:
			// Create a context with timeout for reading messages
			readCtx, cancel := context.WithTimeout(ctx, time.Duration(kp.config.KTimeout)*time.Second)
			message, err := reader.ReadMessage(readCtx)
			cancel()

			// Check for context cancellation
			if ctx.Err() != nil {
				return
			}

			// Handle timeout and continue
			if err == context.DeadlineExceeded {
				continue
			}

			// Handle other errors
			if err != nil {
				log.Printf("Error reading message from pipe %s: %v", pipe, err)
				// If the error is fatal, break the loop
				if err != context.Canceled {
					time.Sleep(1 * time.Second) // Wait before retrying
				}
				continue
			}

			// Process the message
			if err := callback(message.Value); err != nil {
				log.Printf("Error processing message from pipe %s: %v", pipe, err)
			}
		}
	}
}

// Remove unsubscribes from the specified pipe.
func (kp *Packet) Remove(pipe string) error {
	return kp.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (kp *Packet) Unsubscribe(pipe string) error {
	// Validate connection with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	kp.mutex.RUnlock()

	// Lock for thread safety
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	// Check if the pipe is subscribed
	reader, exists := kp.Readers[pipe]
	if !exists {
		return core.ErrPipeNotFound
	}

	// Cancel the reader goroutine
	if cancel, exists := kp.readerCancel[pipe]; exists {
		cancel()
		delete(kp.readerCancel, pipe)
	}

	// Close the reader
	if err := reader.Close(); err != nil {
		log.Printf("Error closing reader for pipe %s: %v", pipe, err)
	}
	delete(kp.Readers, pipe)

	log.Printf("Unsubscribed from Kafka pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the Kafka server.
func (kp *Packet) Disconnect() error {
	// Check if connected with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return nil
	}
	kp.mutex.RUnlock()

	// Lock for thread safety
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	// Cancel all reader goroutines
	for pipe, cancel := range kp.readerCancel {
		cancel()
		delete(kp.readerCancel, pipe)
		log.Printf("Canceled reader for pipe %s", pipe)
	}

	// Close all readers
	for pipe, reader := range kp.Readers {
		if err := reader.Close(); err != nil {
			log.Printf("Error closing reader for pipe %s: %v", pipe, err)
		}
	}
	kp.Readers = make(map[string]*kafka.Reader)

	// Close all writers
	for pipe, writer := range kp.Writers {
		if err := writer.Close(); err != nil {
			log.Printf("Error closing writer for pipe %s: %v", pipe, err)
		}
	}
	kp.Writers = make(map[string]*kafka.Writer)

	// Clear the dialer
	kp.DialerConn = nil

	log.Printf("Closed Kafka connection")
	return nil
}

// Close terminates the connection to the Kafka server.
func (kp *Packet) Close() error {
	return kp.Disconnect()
}

// IsConnected returns true if the connection to the Kafka server is active.
func (kp *Packet) IsConnected() bool {
	kp.mutex.RLock()
	defer kp.mutex.RUnlock()
	return kp.DialerConn != nil
}
