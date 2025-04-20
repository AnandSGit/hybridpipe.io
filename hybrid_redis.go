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

	"github.com/go-redis/redis/v8"
)

// RedisPacket implements the HybridPipe interface for Redis messaging system.
type RedisPacket struct {
	// client is the Redis client
	client *redis.Client
	// pubsub is the Redis pubsub client
	pubsub *redis.PubSub
	// subscriptions maps pipe names to their subscription channels
	subscriptions map[string]<-chan *redis.Message
	// handlers maps pipe names to their message handlers
	handlers map[string]Process
	// ctx is the context for Redis operations
	ctx context.Context
	// cancel is the cancel function for the context
	cancel context.CancelFunc
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
}

// createRedisTLSConfig creates a TLS configuration for Redis connections.
func createRedisTLSConfig(clientCert, clientKey, caCert string) (*tls.Config, error) {
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

	return &tlsConfig, nil
}

// Connect establishes a connection to the Redis server using the configuration in HPipeConfig.
func (rp *RedisPacket) Connect() error {
	// Initialize maps and context
	rp.mutex.Lock()
	if rp.subscriptions == nil {
		rp.subscriptions = make(map[string]<-chan *redis.Message)
	}
	if rp.handlers == nil {
		rp.handlers = make(map[string]Process)
	}
	
	// Create a new context with cancel function
	rp.ctx, rp.cancel = context.WithCancel(context.Background())
	rp.mutex.Unlock()

	// Create Redis options
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", HPipeConfig.RedisF.RedisAddress, HPipeConfig.RedisF.RedisPort),
		Password: HPipeConfig.RedisF.RedisPassword,
		DB:       HPipeConfig.RedisF.RedisDB,
	}

	// Set username if provided (Redis 6+ ACL)
	if HPipeConfig.RedisF.RedisUsername != "" {
		options.Username = HPipeConfig.RedisF.RedisUsername
	}

	// Set connection pool options
	if HPipeConfig.RedisF.RedisPoolSize > 0 {
		options.PoolSize = HPipeConfig.RedisF.RedisPoolSize
	}
	if HPipeConfig.RedisF.RedisMinIdleConns > 0 {
		options.MinIdleConns = HPipeConfig.RedisF.RedisMinIdleConns
	}

	// Set timeouts
	if HPipeConfig.RedisF.RedisConnectTimeout > 0 {
		options.DialTimeout = time.Duration(HPipeConfig.RedisF.RedisConnectTimeout) * time.Second
	}
	if HPipeConfig.RedisF.RedisReadTimeout > 0 {
		options.ReadTimeout = time.Duration(HPipeConfig.RedisF.RedisReadTimeout) * time.Second
	}
	if HPipeConfig.RedisF.RedisWriteTimeout > 0 {
		options.WriteTimeout = time.Duration(HPipeConfig.RedisF.RedisWriteTimeout) * time.Second
	}

	// Set up TLS if enabled
	if HPipeConfig.RedisF.RedisTLSEnabled {
		tlsConfig, err := createRedisTLSConfig(
			HPipeConfig.RedisF.RedisCertFile,
			HPipeConfig.RedisF.RedisKeyFile,
			HPipeConfig.RedisF.RedisCAFile,
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
	ctx, cancel := context.WithTimeout(rp.ctx, time.Duration(HPipeConfig.RedisF.RedisConnectTimeout)*time.Second)
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

	log.Printf("Connected to Redis server at %s:%d", HPipeConfig.RedisF.RedisAddress, HPipeConfig.RedisF.RedisPort)
	return nil
}

// Dispatch publishes a message to the specified pipe.
func (rp *RedisPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.client == nil {
		rp.mutex.RUnlock()
		return fmt.Errorf("not connected to Redis server")
	}
	client := rp.client
	ctx := rp.ctx
	rp.mutex.RUnlock()

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Publish the message
	err = client.Publish(ctx, pipe, bytes).Err()
	if err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (rp *RedisPacket) Accept(pipe string, fn Process) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.client == nil || rp.pubsub == nil {
		rp.mutex.RUnlock()
		return fmt.Errorf("not connected to Redis server")
	}
	pubsub := rp.pubsub
	ctx := rp.ctx
	rp.mutex.RUnlock()

	// Lock for thread safety during setup
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Check if already subscribed
	if _, exists := rp.subscriptions[pipe]; exists {
		return fmt.Errorf("already subscribed to pipe %s", pipe)
	}

	// Subscribe to the pipe
	err := pubsub.Subscribe(ctx, pipe)
	if err != nil {
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Get the channel for this subscription
	channel := pubsub.Channel()

	// Store the subscription and handler
	rp.subscriptions[pipe] = channel
	rp.handlers[pipe] = fn

	// Start a goroutine to process messages
	go rp.processMessages(pipe, channel)

	log.Printf("Subscribed to Redis pipe %s", pipe)
	return nil
}

// processMessages processes messages from a Redis subscription channel.
func (rp *RedisPacket) processMessages(pipe string, channel <-chan *redis.Message) {
	for {
		// Check if context is done
		rp.mutex.RLock()
		ctx := rp.ctx
		rp.mutex.RUnlock()

		select {
		case <-ctx.Done():
			// Context canceled, stop processing
			return
		case msg, ok := <-channel:
			if !ok {
				// Channel closed
				log.Printf("Redis subscription channel closed for pipe %s", pipe)
				return
			}

			// Get the handler for this pipe
			rp.mutex.RLock()
			handler, exists := rp.handlers[pipe]
			rp.mutex.RUnlock()

			if !exists {
				// Handler removed, stop processing
				log.Printf("No handler for pipe %s", pipe)
				return
			}

			// Decode the message
			var data interface{}
			if err := Decode([]byte(msg.Payload), &data); err != nil {
				log.Printf("Failed to decode message from pipe %s: %v", pipe, err)
				continue
			}

			// Process the message
			handler(data)
		}
	}
}

// Remove unsubscribes from the specified pipe.
func (rp *RedisPacket) Remove(pipe string) error {
	// Validate connection with read lock
	rp.mutex.RLock()
	if rp.client == nil || rp.pubsub == nil {
		rp.mutex.RUnlock()
		return fmt.Errorf("not connected to Redis server")
	}
	pubsub := rp.pubsub
	ctx := rp.ctx
	rp.mutex.RUnlock()

	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Check if the pipe is subscribed
	if _, exists := rp.subscriptions[pipe]; !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
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

// Close terminates the connection to the Redis server.
func (rp *RedisPacket) Close() {
	// Lock for thread safety
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Check if connected
	if rp.client == nil {
		return
	}

	// Cancel the context to stop all goroutines
	if rp.cancel != nil {
		rp.cancel()
	}

	// Close the PubSub client
	if rp.pubsub != nil {
		rp.pubsub.Close()
		rp.pubsub = nil
	}

	// Close the Redis client
	rp.client.Close()
	rp.client = nil

	// Clear the maps
	rp.subscriptions = make(map[string]<-chan *redis.Message)
	rp.handlers = make(map[string]Process)

	log.Printf("Disconnected from Redis server")
}
