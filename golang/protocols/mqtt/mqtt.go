// Package mqtt provides an implementation of the HybridPipe interface for MQTT messaging system.
package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"hybridpipe.io/core"
)

// Packet implements the HybridPipe interface for MQTT messaging system.
type Packet struct {
	// client is the MQTT client
	client mqtt.Client
	// subscriptions maps pipe names to their subscription tokens
	subscriptions map[string]mqtt.Token
	// handlers maps pipe names to their message handlers
	handlers map[string]mqtt.MessageHandler
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
	// config holds the MQTT configuration
	config *core.MQTTConfig
}

// New creates a new MQTT packet with the specified configuration.
func New(config *core.MQTTConfig) *Packet {
	return &Packet{
		subscriptions: make(map[string]mqtt.Token),
		handlers:      make(map[string]mqtt.MessageHandler),
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

// Connect establishes a connection to the MQTT broker using the configuration.
func (mp *Packet) Connect() error {
	// Get configuration if not already set
	if mp.config == nil {
		config, err := core.GetProtocolConfig(core.MQTT)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}
		mqttConfig := config.(core.MQTTConfig)
		mp.config = &mqttConfig
	}

	// Initialize maps if needed
	mp.mutex.Lock()
	if mp.subscriptions == nil {
		mp.subscriptions = make(map[string]mqtt.Token)
	}
	if mp.handlers == nil {
		mp.handlers = make(map[string]mqtt.MessageHandler)
	}
	mp.mutex.Unlock()

	// Create MQTT client options
	opts := mqtt.NewClientOptions()

	// Set broker URL
	opts.AddBroker(mp.config.MQTTBroker)

	// Set client ID with random suffix if not provided
	clientID := mp.config.MQTTClientID
	if clientID == "" {
		clientID = fmt.Sprintf("hybridpipe-%d", time.Now().UnixNano())
	}
	opts.SetClientID(clientID)

	// Set credentials if provided
	if mp.config.MQTTUsername != "" && mp.config.MQTTPassword != "" {
		opts.SetUsername(mp.config.MQTTUsername)
		opts.SetPassword(mp.config.MQTTPassword)
	}

	// Set TLS if enabled
	if mp.config.MQTTTLSEnabled {
		tlsConfig, err := createTLSConfig(
			mp.config.MQTTCertFile,
			mp.config.MQTTKeyFile,
			mp.config.MQTTCAFile,
		)
		if err != nil {
			return fmt.Errorf("failed to create TLS configuration: %w", err)
		}
		if tlsConfig != nil {
			opts.SetTLSConfig(tlsConfig)
		}
	}

	// Set connection timeout
	connectTimeout := time.Duration(mp.config.MQTTConnectTimeout) * time.Second
	if connectTimeout == 0 {
		connectTimeout = 30 * time.Second // Default timeout
	}
	opts.SetConnectTimeout(connectTimeout)

	// Set keep alive interval
	if mp.config.MQTTKeepAlive > 0 {
		opts.SetKeepAlive(time.Duration(mp.config.MQTTKeepAlive) * time.Second)
	}

	// Set clean session
	opts.SetCleanSession(mp.config.MQTTCleanSession)

	// Set connection lost handler
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("Connection to MQTT broker lost: %v", err)
	})

	// Create and connect the client
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(connectTimeout) {
		return fmt.Errorf("connection to MQTT broker timed out")
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	// Store the client
	mp.client = client

	log.Printf("Connected to MQTT broker at %s with client ID %s", mp.config.MQTTBroker, clientID)
	return nil
}

// Dispatch sends a message to the specified pipe.
func (mp *Packet) Dispatch(pipe string, data any) error {
	// Validate connection
	if !mp.IsConnected() {
		return core.ErrNotConnected
	}

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Set QoS level
	qos := byte(mp.config.MQTTQoS)
	if qos > 2 {
		qos = 1 // Default to QoS 1 if invalid
	}

	// Publish the message
	token := mp.client.Publish(pipe, qos, mp.config.MQTTRetain, bytes)
	if !token.WaitTimeout(time.Duration(mp.config.MQTTPublishTimeout) * time.Second) {
		return fmt.Errorf("publish to pipe %s timed out", pipe)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to publish to pipe %s: %w", pipe, err)
	}

	return nil
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (mp *Packet) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with dispatch
	}

	// Validate connection
	if !mp.IsConnected() {
		return core.ErrNotConnected
	}

	// Encode the data
	bytes, err := core.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Set QoS level
	qos := byte(mp.config.MQTTQoS)
	if qos > 2 {
		qos = 1 // Default to QoS 1 if invalid
	}

	// Create a channel to signal completion
	done := make(chan struct{})
	var publishErr error

	// Publish the message
	token := mp.client.Publish(pipe, qos, mp.config.MQTTRetain, bytes)

	// Wait for the token to complete
	if !token.WaitTimeout(time.Duration(mp.config.MQTTPublishTimeout) * time.Second) {
		return fmt.Errorf("publish to pipe %s timed out", pipe)
	}

	// Check for errors
	publishErr = token.Error()
	close(done)

	// Wait for either context cancellation or publish completion
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if publishErr != nil {
			return fmt.Errorf("failed to publish to pipe %s: %w", pipe, publishErr)
		}
		return nil
	}
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (mp *Packet) Accept(pipe string, fn func(any)) error {
	// Create a wrapper that converts the function to the expected type
	wrapper := func(client mqtt.Client, msg mqtt.Message) {
		var decoded any
		if err := core.Decode(msg.Payload(), &decoded); err != nil {
			log.Printf("Failed to decode message from pipe %s: %v", pipe, err)
			return
		}
		fn(decoded)
	}

	// Subscribe using the wrapper
	return mp.subscribeInternal(pipe, wrapper)
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (mp *Packet) Subscribe(pipe string, callback core.Process) error {
	// Create a message handler that calls the callback
	handler := func(client mqtt.Client, msg mqtt.Message) {
		if err := callback(msg.Payload()); err != nil {
			log.Printf("Error processing message from pipe %s: %v", pipe, err)
		}
	}

	// Subscribe using the handler
	return mp.subscribeInternal(pipe, handler)
}

// SubscribeWithContext registers a callback with context for cancellation.
func (mp *Packet) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with subscription
	}

	// Create a message handler that calls the callback
	handler := func(client mqtt.Client, msg mqtt.Message) {
		// Check for context cancellation before processing
		select {
		case <-ctx.Done():
			return
		default:
			if err := callback(msg.Payload()); err != nil {
				log.Printf("Error processing message from pipe %s: %v", pipe, err)
			}
		}
	}

	// Subscribe using the handler
	if err := mp.subscribeInternal(pipe, handler); err != nil {
		return err
	}

	// Set up a goroutine to unsubscribe when the context is canceled
	go func() {
		<-ctx.Done()
		mp.Unsubscribe(pipe)
	}()

	return nil
}

// subscribeInternal is a helper function for subscribing to a pipe.
func (mp *Packet) subscribeInternal(pipe string, handler mqtt.MessageHandler) error {
	// Validate connection
	if !mp.IsConnected() {
		return core.ErrNotConnected
	}

	// Lock for thread safety
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Set QoS level
	qos := byte(mp.config.MQTTQoS)
	if qos > 2 {
		qos = 1 // Default to QoS 1 if invalid
	}

	// Store the handler
	mp.handlers[pipe] = handler

	// Subscribe to the pipe
	token := mp.client.Subscribe(pipe, qos, handler)
	if !token.WaitTimeout(time.Duration(mp.config.MQTTConnectTimeout) * time.Second) {
		delete(mp.handlers, pipe)
		return fmt.Errorf("subscription to pipe %s timed out", pipe)
	}

	if err := token.Error(); err != nil {
		delete(mp.handlers, pipe)
		return fmt.Errorf("failed to subscribe to pipe %s: %w", pipe, err)
	}

	// Store the subscription token
	mp.subscriptions[pipe] = token

	log.Printf("Subscribed to MQTT pipe %s with QoS %d", pipe, qos)
	return nil
}

// Remove unsubscribes from the specified pipe.
func (mp *Packet) Remove(pipe string) error {
	return mp.Unsubscribe(pipe)
}

// Unsubscribe removes a subscription from the specified pipe.
func (mp *Packet) Unsubscribe(pipe string) error {
	// Validate connection
	if !mp.IsConnected() {
		return core.ErrNotConnected
	}

	// Lock for thread safety
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Check if the pipe is subscribed
	if _, exists := mp.handlers[pipe]; !exists {
		return core.ErrPipeNotFound
	}

	// Unsubscribe from the pipe
	token := mp.client.Unsubscribe(pipe)
	if !token.WaitTimeout(time.Duration(mp.config.MQTTConnectTimeout) * time.Second) {
		return fmt.Errorf("unsubscription from pipe %s timed out", pipe)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to unsubscribe from pipe %s: %w", pipe, err)
	}

	// Remove the subscription and handler
	delete(mp.subscriptions, pipe)
	delete(mp.handlers, pipe)

	log.Printf("Unsubscribed from MQTT pipe %s", pipe)
	return nil
}

// Disconnect closes the connection to the MQTT broker.
func (mp *Packet) Disconnect() error {
	// Check if connected
	if !mp.IsConnected() {
		return nil
	}

	// Disconnect with a timeout
	mp.client.Disconnect(250) // 250ms timeout

	// Clear the client and subscriptions
	mp.mutex.Lock()
	mp.subscriptions = make(map[string]mqtt.Token)
	mp.handlers = make(map[string]mqtt.MessageHandler)
	mp.mutex.Unlock()

	log.Printf("Disconnected from MQTT broker")
	return nil
}

// Close terminates the connection to the MQTT broker.
func (mp *Packet) Close() error {
	return mp.Disconnect()
}

// IsConnected returns true if the connection to the MQTT broker is active.
func (mp *Packet) IsConnected() bool {
	return mp.client != nil && mp.client.IsConnected()
}
