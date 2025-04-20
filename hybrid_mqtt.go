package hybridpipe

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTPacket implements the HybridPipe interface for MQTT messaging system.
type MQTTPacket struct {
	// client is the MQTT client
	client mqtt.Client
	// subscriptions maps pipe names to their subscription tokens
	subscriptions map[string]mqtt.Token
	// handlers maps pipe names to their message handlers
	handlers map[string]mqtt.MessageHandler
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
}

// createMQTTTLSConfig creates a TLS configuration for MQTT connections.
func createMQTTTLSConfig(clientCert, clientKey, caCert string) (*tls.Config, error) {
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

// Connect establishes a connection to the MQTT broker using the configuration in HPipeConfig.
func (mp *MQTTPacket) Connect() error {
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
	opts.AddBroker(HPipeConfig.MQTTF.MQTTBroker)

	// Set client ID with random suffix if not provided
	clientID := HPipeConfig.MQTTF.MQTTClientID
	if clientID == "" {
		clientID = fmt.Sprintf("hybridpipe-%d", time.Now().UnixNano())
	}
	opts.SetClientID(clientID)

	// Set credentials if provided
	if HPipeConfig.MQTTF.MQTTUsername != "" && HPipeConfig.MQTTF.MQTTPassword != "" {
		opts.SetUsername(HPipeConfig.MQTTF.MQTTUsername)
		opts.SetPassword(HPipeConfig.MQTTF.MQTTPassword)
	}

	// Set connection timeout
	connectTimeout := time.Duration(HPipeConfig.MQTTF.MQTTConnectTimeout) * time.Second
	if connectTimeout == 0 {
		connectTimeout = 30 * time.Second // Default timeout
	}
	opts.SetConnectTimeout(connectTimeout)

	// Set clean session flag
	opts.SetCleanSession(HPipeConfig.MQTTF.MQTTCleanSession)

	// Set up TLS if certificates are provided
	if HPipeConfig.MQTTF.MQTTCertFile != "" && HPipeConfig.MQTTF.MQTTKeyFile != "" && HPipeConfig.MQTTF.MQTTCAFile != "" {
		tlsConfig, err := createMQTTTLSConfig(
			HPipeConfig.MQTTF.MQTTCertFile,
			HPipeConfig.MQTTF.MQTTKeyFile,
			HPipeConfig.MQTTF.MQTTCAFile,
		)
		if err != nil {
			return fmt.Errorf("failed to create TLS configuration: %w", err)
		}

		if tlsConfig != nil {
			opts.SetTLSConfig(tlsConfig)
		}
	}

	// Set connection callbacks
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("Connected to MQTT broker at %s", HPipeConfig.MQTTF.MQTTBroker)
	})

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

	log.Printf("Connected to MQTT broker at %s with client ID %s", HPipeConfig.MQTTF.MQTTBroker, clientID)
	return nil
}

// Dispatch publishes a message to the specified pipe.
func (mp *MQTTPacket) Dispatch(pipe string, data interface{}) error {
	// Validate connection
	mp.mutex.RLock()
	if mp.client == nil {
		mp.mutex.RUnlock()
		return fmt.Errorf("not connected to MQTT broker")
	}
	mp.mutex.RUnlock()

	// Encode the data
	bytes, err := Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Get QoS level from configuration
	qos := byte(HPipeConfig.MQTTF.MQTTQoS)
	if qos > 2 {
		qos = 1 // Default to QoS 1 if invalid
	}

	// Publish the message
	token := mp.client.Publish(pipe, qos, HPipeConfig.MQTTF.MQTTRetained, bytes)
	if !token.WaitTimeout(time.Duration(HPipeConfig.MQTTF.MQTTConnectTimeout) * time.Second) {
		return fmt.Errorf("publish to pipe %s timed out", pipe)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to publish message to pipe %s: %w", pipe, err)
	}

	return nil
}

// Accept subscribes to messages from the specified pipe and processes them with the provided function.
func (mp *MQTTPacket) Accept(pipe string, fn Process) error {
	// Validate connection
	mp.mutex.RLock()
	if mp.client == nil {
		mp.mutex.RUnlock()
		return fmt.Errorf("not connected to MQTT broker")
	}
	mp.mutex.RUnlock()

	// Lock for thread safety during setup
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Check if already subscribed
	if _, exists := mp.subscriptions[pipe]; exists {
		return fmt.Errorf("already subscribed to pipe %s", pipe)
	}

	// Get QoS level from configuration
	qos := byte(HPipeConfig.MQTTF.MQTTQoS)
	if qos > 2 {
		qos = 1 // Default to QoS 1 if invalid
	}

	// Create a message handler
	handler := func(client mqtt.Client, msg mqtt.Message) {
		// Decode the message
		var data interface{}
		if err := Decode(msg.Payload(), &data); err != nil {
			log.Printf("Failed to decode message from pipe %s: %v", pipe, err)
			return
		}

		// Process the message
		fn(data)
	}

	// Store the handler
	mp.handlers[pipe] = handler

	// Subscribe to the pipe
	token := mp.client.Subscribe(pipe, qos, handler)
	if !token.WaitTimeout(time.Duration(HPipeConfig.MQTTF.MQTTConnectTimeout) * time.Second) {
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
func (mp *MQTTPacket) Remove(pipe string) error {
	// Validate connection
	mp.mutex.RLock()
	if mp.client == nil {
		mp.mutex.RUnlock()
		return fmt.Errorf("not connected to MQTT broker")
	}
	mp.mutex.RUnlock()

	// Lock for thread safety
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Check if the pipe is subscribed
	if _, exists := mp.subscriptions[pipe]; !exists {
		return fmt.Errorf("pipe %s is not subscribed", pipe)
	}

	// Unsubscribe from the pipe
	token := mp.client.Unsubscribe(pipe)
	if !token.WaitTimeout(time.Duration(HPipeConfig.MQTTF.MQTTConnectTimeout) * time.Second) {
		return fmt.Errorf("unsubscribe from pipe %s timed out", pipe)
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

// Close terminates the connection to the MQTT broker.
func (mp *MQTTPacket) Close() {
	// Lock for thread safety
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Check if connected
	if mp.client == nil {
		return
	}

	// Unsubscribe from all pipes
	for pipe := range mp.subscriptions {
		token := mp.client.Unsubscribe(pipe)
		token.Wait()
		if err := token.Error(); err != nil {
			log.Printf("Error unsubscribing from pipe %s: %v", pipe, err)
		}
	}

	// Clear the subscriptions and handlers maps
	mp.subscriptions = make(map[string]mqtt.Token)
	mp.handlers = make(map[string]mqtt.MessageHandler)

	// Disconnect from the broker
	mp.client.Disconnect(250) // Wait 250ms for the broker to clean up
	mp.client = nil

	log.Printf("Disconnected from MQTT broker")
}
