// Package redis provides an implementation of the HybridPipe interface for Redis messaging system.
package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"hybridpipe.io/core"
)

// Packet implements the HybridPipe interface for Redis messaging system.
type Packet struct {
	// client is the Redis client
	client *redis.Client
	// pubsub is the Redis pubsub client
	pubsub *redis.PubSub
	// subscriptions maps pipe names to their subscription channels
	subscriptions map[string]<-chan *redis.Message
	// handlers maps pipe names to their message handlers
	handlers map[string]core.Process
	// ctx is the context for Redis operations
	ctx context.Context
	// cancel is the cancel function for the context
	cancel context.CancelFunc
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// config holds the Redis configuration
	config *core.RedisConfig
}

// New creates a new Redis packet with the specified configuration.
func New(config *core.RedisConfig) *Packet {
	ctx, cancel := context.WithCancel(context.Background())
	return &Packet{
		subscriptions: make(map[string]<-chan *redis.Message),
		handlers:      make(map[string]core.Process),
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
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

// Connect establishes a connection to the Redis server using the configuration.
func (rp *Packet) Connect() error {
	// Get configuration if not already set
	if rp.config == nil {
		config, err := core.GetProtocolConfig(core.REDIS)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		redisConfig := config.(core.RedisConfig)
		rp.config = &redisConfig
	}

	// Initialize maps and context
	rp.mutex.Lock()
	if rp.subscriptions == nil {
		rp.subscriptions = make(map[string]<-chan *redis.Message)
	}
	if rp.handlers == nil {
		rp.handlers = make(map[string]core.Process)
	}

	// Create a new context with cancel function
	if rp.ctx == nil || rp.ctx.Err() != nil {
		rp.ctx, rp.cancel = context.WithCancel(context.Background())
	}
	rp.mutex.Unlock()

	// Create Redis options
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", rp.config.RedisAddress, rp.config.RedisPort),
		Password: rp.config.RedisPassword,
		DB:       rp.config.RedisDB,
	}

	// Set username if provided
	if rp.config.RedisUsername != "" {
		options.Username = rp.config.RedisUsername
	}

	// Set connection pool options
	if rp.config.RedisPoolSize > 0 {
		options.PoolSize = rp.config.RedisPoolSize
	}
	if rp.config.RedisMinIdleConns > 0 {
		options.MinIdleConns = rp.config.RedisMinIdleConns
	}

	// Set timeouts
	if rp.config.RedisConnectTimeout > 0 {
		options.DialTimeout = time.Duration(rp.config.RedisConnectTimeout) * time.Second
	}
	if rp.config.RedisReadTimeout > 0 {
		options.ReadTimeout = time.Duration(rp.config.RedisReadTimeout) * time.Second
	}
	if rp.config.RedisWriteTimeout > 0 {
		options.WriteTimeout = time.Duration(rp.config.RedisWriteTimeout) * time.Second
	}

	// Set TLS if enabled
	if rp.config.RedisTLSEnabled {
		tlsConfig, err := createTLSConfig(
			rp.config.RedisCertFile,
			rp.config.RedisKeyFile,
			rp.config.RedisCAFile,
		)
		if err != nil {
			return fmt.Errorf("failed to create TLS configuration: %w", err)
		}
		if tlsConfig != nil {
			options.TLSConfig = tlsConfig
		}
	}

	// Create Redis client
	client := redis.NewClient(options)

	// Test the connection
	ctx, cancel := context.WithTimeout(rp.ctx, time.Duration(rp.config.RedisConnectTimeout)*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis server: %w", err)
	}

	// Create PubSub client
	pubsub := client.Subscribe(rp.ctx)

	// Store the clients
	rp.mutex.Lock()
	rp.client = client
	rp.pubsub = pubsub
	rp.mutex.Unlock()

	log.Printf("Connected to Redis server at %s:%d", rp.config.RedisAddress, rp.config.RedisPort)
	return nil
}

// Dispatch sends a message to the specified pipe.
func (rp *Packet) Dispatch(pipe string, data any) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.client == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	client := rp.client
	ctx := rp.ctx
	rp.mutex.RUnlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message
	if err := client.Publish(ctx, pipe, bytes).Err(); err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (rp *Packet) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.client == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	client := rp.client
	rp.mutex.RUnlock()

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message with the provided context
	if err := client.Publish(ctx, pipe, bytes).Err(); err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (rp *Packet) Accept(pipe string, fn func(any)) error {
	// Create a wrapper that converts the function to the expected type
	wrapper := func(data []byte) error {
		var decoded any
		if err := core.Decode(data, &decoded); err != nil {
			return err
		}
		fn(decoded)
		return nil
	}

	return rp.Subscribe(pipe, wrapper)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (rp *Packet) Subscribe(pipe string, callback core.Process) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.client == nil || rp.pubsub == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	pubsub := rp.pubsub
	ctx := rp.ctx
	rp.mutex.RUnlock()

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Subscribe to the pipe
	err := pubsub.Subscribe(ctx, pipe)
	if err != nil {
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Get the channel for receiving messages
	ch := pubsub.Channel()

	// Store the subscription and handler
	rp.subscriptions[pipe] = ch
	rp.handlers[pipe] = callback

	// Start a goroutine to handle messages
	go rp.readMessages(pipe, ch, callback)

	log.Printf("Subscribed to Redis pipe %s", pipe)
	return nil
}

// SubscribeWithContext registers a callback with context for cancellation.
func (rp *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with subscription
	}

	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.client == nil || rp.pubsub == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	pubsub := rp.pubsub
	rp.mutex.RUnlock()

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Subscribe to the pipe
	err := pubsub.Subscribe(ctx, pipe)
	if err != nil {
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Get the channel for receiving messages
	ch := pubsub.Channel()

	// Store the subscription and handler
	rp.subscriptions[pipe] = ch
	rp.handlers[pipe] = callback

	// Start a goroutine to handle messages
	go rp.readMessagesWithContext(ctx, pipe, ch, callback)

	log.Printf("Subscribed to Redis pipe %s with context", pipe)
	return nil
}

// readMessages reads messages from Redis in a loop until the context is canceled.
func (rp *Packet) readMessages(pipe string, ch <-chan *redis.Message, callback core.Process) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in Redis reader for pipe %s: %v", pipe, r)
		}
	}()

	for {
		select {
		case <-rp.ctx.Done():
			log.Printf("Stopping Redis reader for pipe %s", pipe)
			return
		case msg, ok := <-ch:
			if !ok {
				log.Printf("Redis channel closed for pipe %s", pipe)
				return
			}

			// Skip messages for other pipes
			if msg.Channel != pipe {
				continue
			}

			// Process the message
			if err := callback([]byte(msg.Payload)); err != nil {
				log.Printf("Error processing message from pipe %s: %v", pipe, err)
			}
		}
	}
}

// readMessagesWithContext reads messages from Redis in a loop until the context is canceled.
func (rp *Packet) readMessagesWithContext(ctx context.Context, pipe string, ch <-chan *redis.Message, callback core.Process) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in Redis reader for pipe %s: %v", pipe, r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context canceled for Redis reader for pipe %s", pipe)
			return
		case <-rp.ctx.Done():
			log.Printf("Stopping Redis reader for pipe %s", pipe)
			return
		case msg, ok := <-ch:
			if !ok {
				log.Printf("Redis channel closed for pipe %s", pipe)
				return
			}

			// Skip messages for other pipes
			if msg.Channel != pipe {
				continue
			}

			// Check context before processing
			select {
			case <-ctx.Done():
				return
			default:
				// Process the message
				if err := callback([]byte(msg.Payload)); err != nil {
					log.Printf("Error processing message from pipe %s: %v", pipe, err)
				}
			}
		}
	}
}

// Remove unsubscribes from the specified pipe.
func (rp *Packet) Remove(pipe string) error {
	return rp.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (rp *Packet) Unsubscribe(pipe string) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.client == nil || rp.pubsub == nil {
		rp.mutex.RUnlock()
		return core.ErrNotConnected
	}
	pubsub := rp.pubsub
	ctx := rp.ctx
	rp.mutex.RUnlock()

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Check if the pipe is subscribed
	if _, exists := rp.handlers[pipe]; !exists {
		return core.ErrPipeNotFound
	}

	// Unsubscribe from the pipe
	err := pubsub.Unsubscribe(ctx, pipe)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from pipe %s: %w", pipe, err)
	}

	// Remove the subscription and handler
	delete(rp.subscriptions, pipe)
	delete(rp.handlers, pipe)

	log.Printf("Unsubscribed from Redis pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the Redis server.
func (rp *Packet) Disconnect() error {
	// Check if connected with read lock
	rp.mutex.RLock()
	if rp.client == nil {
		rp.mutex.RUnlock()
		return nil
	}
	client := rp.client
	pubsub := rp.pubsub
	rp.mutex.RUnlock()

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Cancel the context to signal shutdown
	if rp.cancel != nil {
		rp.cancel()
	}

	// Close the PubSub client
	if pubsub != nil {
		if err := pubsub.Close(); err != nil {
			log.Printf("Error closing Redis PubSub: %v", err)
		}
		rp.pubsub = nil
	}

	// Close the Redis client
	if err := client.Close(); err != nil {
		log.Printf("Error closing Redis client: %v", err)
		return err
	}
	rp.client = nil

	// Clear the maps
	rp.subscriptions = make(map[string]<-chan *redis.Message)
	rp.handlers = make(map[string]core.Process)

	log.Printf("Disconnected from Redis server")
	return nil
}

// Close terminates the connection to the Redis server.
func (rp *Packet) Close() error {
	return rp.Disconnect()
}

// IsConnected returns true if the connection to the Redis server is active.
func (rp *Packet) IsConnected() bool {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()
	return rp.client != nil && rp.pubsub != nil
}
