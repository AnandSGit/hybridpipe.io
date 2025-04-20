package hybridpipe

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
)

// KafkaPacket implements the HybridPipe interface for Kafka messaging system.
type KafkaPacket struct {
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
}

// createTLSConfig creates the TLS configuration for Kafka connections.
// It loads client certificates and CA certificates for secure communication.
func createTLSConfig(clientCert, clientKey, caCert string) (*tls.Config, error) {
	// Create a basic TLS configuration
	tlsConfig := tls.Config{
		MinVersion: tls.VersionTLS12, // Enforce minimum TLS version for security
	}

	// Skip TLS setup if certificates are not provided
	if clientCert == "" || clientKey == "" || caCert == "" {
		log.Printf("Warning: TLS certificates not provided, using insecure connection")
		return nil, nil
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
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
	tlsConfig.RootCAs = caCertPool

	// Return the TLS config
	return &tlsConfig, nil
}

// Connect establishes a connection to the Kafka server using the configuration in HPipeConfig.
func (kp *KafkaPacket) Connect() error {
	// Build the Kafka server URL with port
	kp.Server = fmt.Sprintf("%s:%d", HPipeConfig.KServer, HPipeConfig.KLport)

	// Create TLS configuration if certificates are provided
	tlsConfig, err := createTLSConfig(
		HPipeConfig.KAFKACertFile,
		HPipeConfig.KAFKAKeyFile,
		HPipeConfig.KAFKACAFile,
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
		Timeout:   time.Duration(HPipeConfig.KTimeout) * time.Second,
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

// Dispatch publishes a message to the specified pipe.
func (kp *KafkaPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return fmt.Errorf("not connected to Kafka server")
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
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(pipe),
		Value: bytes,
		Time:  time.Now(),
	}

	// Write message to Kafka
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(HPipeConfig.KTimeout)*time.Second)
	defer cancel()

	if err = writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (kp *KafkaPacket) Accept(pipe string, fn Process) error {
	// Validate connection with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return fmt.Errorf("not connected to Kafka server")
	}
	kp.mutex.RUnlock()

	// Check if already subscribed
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	// Cancel existing reader if any
	if cancel, exists := kp.readerCancel[pipe]; exists {
		cancel()
		delete(kp.readerCancel, pipe)
	}

	// Create a new reader for the pipe
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kp.Server},
		GroupID:        fmt.Sprintf("%s-%d", pipe, os.Getpid()), // Unique group ID for this process
		Topic:          pipe,
		MinBytes:       10e1, // 100 bytes
		MaxBytes:       10e6, // 10MB
		CommitInterval: 1 * time.Second,
		Dialer:         kp.DialerConn,
		StartOffset:    kafka.LastOffset, // Start reading from the latest message
	})

	// Store the reader
	kp.Readers[pipe] = reader

	// Create a context with cancel function for the reader goroutine
	ctx, cancel := context.WithCancel(context.Background())
	kp.readerCancel[pipe] = cancel

	// Start reading messages in a goroutine
	go kp.readMessages(ctx, pipe, reader, fn)

	log.Printf("Subscribed to Kafka pipe %s", pipe)
	return nil
}

// readMessages reads messages from Kafka in a loop until the context is canceled.
func (kp *KafkaPacket) readMessages(ctx context.Context, pipe string, reader *kafka.Reader, fn Process) {
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
			// Read message with timeout
			readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			message, err := reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				// Check if context was canceled
				if ctx.Err() != nil {
					return
				}

				// Log other errors but continue
				if err != context.DeadlineExceeded {
					log.Printf("Error reading from Kafka pipe %s: %v", pipe, err)
				}
				continue
			}

			// Decode the message
			var data interface{}
			if err := Decode(message.Value, &data); err != nil {
				log.Printf("Failed to decode message from pipe %s: %v", pipe, err)
				continue
			}

			// Process the message
			fn(data)
		}
	}
}

// Remove unsubscribes from the specified pipe.
func (kp *KafkaPacket) Remove(pipe string) error {
	// Validate connection with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return fmt.Errorf("not connected to Kafka server")
	}
	kp.mutex.RUnlock()

	// Lock for thread safety
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	// Check if the pipe is subscribed
	reader, exists := kp.Readers[pipe]
	if !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Cancel the reader goroutine
	if cancel, exists := kp.readerCancel[pipe]; exists {
		cancel()
		delete(kp.readerCancel, pipe)
	}

	// Close the reader
	if err := reader.Close(); err != nil {
		return fmt.Errorf("failed to close reader for pipe %s: %w", pipe, err)
	}

	// Remove the reader from the map
	delete(kp.Readers, pipe)

	log.Printf("Unsubscribed from Kafka pipe %s", pipe)
	return nil
}

// Close terminates all connections to the Kafka server.
func (kp *KafkaPacket) Close() {
	// Check if connected with read lock
	kp.mutex.RLock()
	if kp.DialerConn == nil {
		kp.mutex.RUnlock()
		return
	}
	kp.mutex.RUnlock()

	// Lock for thread safety
	kp.mutex.Lock()
	defer kp.mutex.Unlock()

	// Cancel all reader goroutines
	for pipe, cancel := range kp.readerCancel {
		cancel()
		log.Printf("Canceled reader for pipe %s", pipe)
	}
	kp.readerCancel = make(map[string]context.CancelFunc)

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

	log.Printf("Closed all Kafka connections")
}
