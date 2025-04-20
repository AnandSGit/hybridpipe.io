package hybridpipe

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Default configuration file path
const defaultConfigFile = "hybridpipe_db.toml"

// ConfigPaths defines the locations where the configuration file will be searched
var ConfigPaths = []string{
	"",                          // Current directory
	"/etc/hybridpipe/",          // System-wide configuration
	"/opt/hybridpipe/config/",   // Application-specific configuration
	"$HOME/.config/hybridpipe/", // User-specific configuration
}

// HybridDB represents the complete configuration for HybridPipe
type HybridDB struct {
	NatsF     `toml:"NATS"`
	KafkaF    `toml:"KAFKA"`
	RabbitMQF `toml:"RABBITMQ"`
	AMQPF     `toml:"AMQP1"`
	QpidF     `toml:"QPID"`
	MQTTF     `toml:"MQTT"`
	NSQF      `toml:"NSQ"`
	TCPF      `toml:"TCP"`
	RedisF    `toml:"REDIS"`
	ZMQF      `toml:"ZEROMQ"`
	GeneralF  `toml:"GENERAL"`
}

// RabbitMQF contains RabbitMQ specific configuration
type RabbitMQF struct {
	// RServerPort is the connection string for RabbitMQ
	RServerPort string `toml:"RServerPort"`
}

// KafkaF contains Kafka specific configuration
type KafkaF struct {
	// KServer is the Kafka server hostname
	KServer string `toml:"KServers"`
	// KLport is the Kafka listener port
	KLport int `toml:"KLPort"`
	// KTimeout is the Kafka connection timeout in seconds
	KTimeout int `toml:"KTimeout"`
	// KAFKACertFile is the path to the client certificate file
	KAFKACertFile string `toml:"KAFKACertFile"`
	// KAFKAKeyFile is the path to the client key file
	KAFKAKeyFile string `toml:"KAFKAKeyFile"`
	// KAFKACAFile is the path to the CA certificate file
	KAFKACAFile string `toml:"KAFKACAFile"`
}

// NatsF contains NATS specific configuration
type NatsF struct {
	// NServer is the NATS server hostname
	NServer string `toml:"NServers"`
	// NLport is the NATS listener port
	NLport int `toml:"NLPort"`
	// NMport is the NATS monitoring port
	NMport int `toml:"NMPort"`
	// NCport is the NATS cluster port
	NCport int `toml:"NCPort"`
	// NATSCertFile is the path to the client certificate file
	NATSCertFile string `toml:"NATSCertFile"`
	// NATSKeyFile is the path to the client key file
	NATSKeyFile string `toml:"NATSKeyFile"`
	// NATSCAFile is the path to the CA certificate file
	NATSCAFile string `toml:"NATSCAFile"`
	// NAllowReconnect enables automatic reconnection
	NAllowReconnect bool `toml:"NAllow_Reconnect"`
	// NMaxAttempt is the maximum number of reconnection attempts
	NMaxAttempt int `toml:"NMax_Attempt"`
	// NReconnectWait is the time to wait between reconnection attempts in seconds
	NReconnectWait int `toml:"NReconnect_Wait"`
	// NTimeout is the connection timeout in seconds
	NTimeout int `toml:"NTimeout"`
}

// AMQPF contains AMQP 1.0 specific configuration
type AMQPF struct {
	// AMQPServer is the AMQP server connection string
	AMQPServer string `toml:"AMQPServer"`
	// AMQPPort is the AMQP server port
	AMQPPort string `toml:"AMQPPort"`
}

// QpidF contains Apache Qpid specific configuration
type QpidF struct {
	// QpidServer is the Qpid server hostname or IP address
	QpidServer string `toml:"QpidServer"`
	// QpidPort is the Qpid server port
	QpidPort int `toml:"QpidPort"`
	// QpidUsername is the username for authentication
	QpidUsername string `toml:"QpidUsername"`
	// QpidPassword is the password for authentication
	QpidPassword string `toml:"QpidPassword"`
	// QpidCertFile is the path to the client certificate file
	QpidCertFile string `toml:"QpidCertFile"`
	// QpidKeyFile is the path to the client key file
	QpidKeyFile string `toml:"QpidKeyFile"`
	// QpidCAFile is the path to the CA certificate file
	QpidCAFile string `toml:"QpidCAFile"`
	// QpidTimeout is the connection timeout in seconds
	QpidTimeout int `toml:"QpidTimeout"`
}

// MQTTF contains MQTT specific configuration
type MQTTF struct {
	// MQTTBroker is the MQTT broker URL (e.g., tcp://localhost:1883)
	MQTTBroker string `toml:"MQTTBroker"`
	// MQTTClientID is the client ID for the MQTT connection
	MQTTClientID string `toml:"MQTTClientID"`
	// MQTTUsername is the username for authentication
	MQTTUsername string `toml:"MQTTUsername"`
	// MQTTPassword is the password for authentication
	MQTTPassword string `toml:"MQTTPassword"`
	// MQTTCleanSession determines if the client has a persistent session
	MQTTCleanSession bool `toml:"MQTTCleanSession"`
	// MQTTQoS is the Quality of Service level (0, 1, or 2)
	MQTTQoS int `toml:"MQTTQoS"`
	// MQTTRetained determines if published messages are retained
	MQTTRetained bool `toml:"MQTTRetained"`
	// MQTTCertFile is the path to the client certificate file
	MQTTCertFile string `toml:"MQTTCertFile"`
	// MQTTKeyFile is the path to the client key file
	MQTTKeyFile string `toml:"MQTTKeyFile"`
	// MQTTCAFile is the path to the CA certificate file
	MQTTCAFile string `toml:"MQTTCAFile"`
	// MQTTConnectTimeout is the connection timeout in seconds
	MQTTConnectTimeout int `toml:"MQTTConnectTimeout"`
}

// NSQF contains NSQ specific configuration
type NSQF struct {
	// NSQLookupdAddresses is a comma-separated list of nsqlookupd addresses
	NSQLookupdAddresses string `toml:"NSQLookupdAddresses"`
	// NSQDAddress is the address of the nsqd instance
	NSQDAddress string `toml:"NSQDAddress"`
	// NSQDPort is the port of the nsqd instance
	NSQDPort int `toml:"NSQDPort"`
	// NSQClientID is the client ID for the NSQ connection
	NSQClientID string `toml:"NSQClientID"`
	// NSQAuthSecret is the secret for authentication
	NSQAuthSecret string `toml:"NSQAuthSecret"`
	// NSQMaxInFlight is the maximum number of messages to process at once
	NSQMaxInFlight int `toml:"NSQMaxInFlight"`
	// NSQMaxAttempts is the maximum number of attempts to process a message
	NSQMaxAttempts int `toml:"NSQMaxAttempts"`
	// NSQRequeueDelay is the delay before requeuing a failed message (in seconds)
	NSQRequeueDelay int `toml:"NSQRequeueDelay"`
	// NSQConnectTimeout is the connection timeout in seconds
	NSQConnectTimeout int `toml:"NSQConnectTimeout"`
}

// TCPF contains TCP specific configuration
type TCPF struct {
	// TCPHost is the host to connect to or listen on
	TCPHost string `toml:"TCPHost"`
	// TCPPort is the port to connect to or listen on
	TCPPort int `toml:"TCPPort"`
	// TCPMode can be "server" or "client"
	TCPMode string `toml:"TCPMode"`
	// TCPBufferSize is the size of the read/write buffer
	TCPBufferSize int `toml:"TCPBufferSize"`
	// TCPKeepAlive enables TCP keep-alive
	TCPKeepAlive bool `toml:"TCPKeepAlive"`
	// TCPKeepAlivePeriod is the keep-alive period in seconds
	TCPKeepAlivePeriod int `toml:"TCPKeepAlivePeriod"`
	// TCPConnectTimeout is the connection timeout in seconds
	TCPConnectTimeout int `toml:"TCPConnectTimeout"`
	// TCPTLSEnabled enables TLS for the connection
	TCPTLSEnabled bool `toml:"TCPTLSEnabled"`
	// TCPCertFile is the path to the client/server certificate file
	TCPCertFile string `toml:"TCPCertFile"`
	// TCPKeyFile is the path to the client/server key file
	TCPKeyFile string `toml:"TCPKeyFile"`
	// TCPCAFile is the path to the CA certificate file
	TCPCAFile string `toml:"TCPCAFile"`
}

// RedisF contains Redis specific configuration
type RedisF struct {
	// RedisAddress is the Redis server address
	RedisAddress string `toml:"RedisAddress"`
	// RedisPort is the Redis server port
	RedisPort int `toml:"RedisPort"`
	// RedisPassword is the password for authentication
	RedisPassword string `toml:"RedisPassword"`
	// RedisDB is the Redis database number
	RedisDB int `toml:"RedisDB"`
	// RedisUsername is the username for Redis 6+ ACL authentication
	RedisUsername string `toml:"RedisUsername"`
	// RedisPoolSize is the maximum number of connections in the pool
	RedisPoolSize int `toml:"RedisPoolSize"`
	// RedisMinIdleConns is the minimum number of idle connections in the pool
	RedisMinIdleConns int `toml:"RedisMinIdleConns"`
	// RedisConnectTimeout is the connection timeout in seconds
	RedisConnectTimeout int `toml:"RedisConnectTimeout"`
	// RedisReadTimeout is the read timeout in seconds
	RedisReadTimeout int `toml:"RedisReadTimeout"`
	// RedisWriteTimeout is the write timeout in seconds
	RedisWriteTimeout int `toml:"RedisWriteTimeout"`
	// RedisTLSEnabled enables TLS for the connection
	RedisTLSEnabled bool `toml:"RedisTLSEnabled"`
	// RedisCertFile is the path to the client certificate file
	RedisCertFile string `toml:"RedisCertFile"`
	// RedisKeyFile is the path to the client key file
	RedisKeyFile string `toml:"RedisKeyFile"`
	// RedisCAFile is the path to the CA certificate file
	RedisCAFile string `toml:"RedisCAFile"`
}

// ZMQF contains ZeroMQ specific configuration
type ZMQF struct {
	// ZMQEndpoints is a comma-separated list of ZeroMQ endpoints (e.g., tcp://localhost:5555)
	ZMQEndpoints string `toml:"ZMQEndpoints"`
	// ZMQPublishEndpoint is the endpoint for publishing messages
	ZMQPublishEndpoint string `toml:"ZMQPublishEndpoint"`
	// ZMQSubscribeEndpoint is the endpoint for subscribing to messages
	ZMQSubscribeEndpoint string `toml:"ZMQSubscribeEndpoint"`
	// ZMQSocketType defines the socket type (PUB, SUB, PUSH, PULL, etc.)
	ZMQSocketType string `toml:"ZMQSocketType"`
	// ZMQHighWaterMark sets the high water mark for sockets
	ZMQHighWaterMark int `toml:"ZMQHighWaterMark"`
	// ZMQLinger sets the linger period for sockets in milliseconds
	ZMQLinger int `toml:"ZMQLinger"`
	// ZMQConnectTimeout is the connection timeout in seconds
	ZMQConnectTimeout int `toml:"ZMQConnectTimeout"`
	// ZMQIOThreads sets the number of I/O threads for the ZeroMQ context
	ZMQIOThreads int `toml:"ZMQIOThreads"`
	// ZMQSecurityMechanism sets the security mechanism (NULL, PLAIN, CURVE)
	ZMQSecurityMechanism string `toml:"ZMQSecurityMechanism"`
	// ZMQCurveServerKey is the server's public key for CURVE security
	ZMQCurveServerKey string `toml:"ZMQCurveServerKey"`
	// ZMQCurvePublicKey is the client's public key for CURVE security
	ZMQCurvePublicKey string `toml:"ZMQCurvePublicKey"`
	// ZMQCurveSecretKey is the client's secret key for CURVE security
	ZMQCurveSecretKey string `toml:"ZMQCurveSecretKey"`
}

// GeneralF contains general configuration options
type GeneralF struct {
	// DBPath is the path to the database location
	DBPath string `toml:"DBLocation"`
}

// HPipeConfig holds the global configuration instance
var HPipeConfig = new(HybridDB)

// ReadConfig reads the HybridPipe configuration file from one of the predefined locations
// or from the location specified by the HYBRIDPIPE_CONFIG environment variable.
func ReadConfig() error {
	// Check for configuration file path in environment variable
	configPath := os.Getenv("HYBRIDPIPE_CONFIG")
	if configPath != "" {
		return loadConfigFile(configPath)
	}

	// Try all predefined paths
	for _, basePath := range ConfigPaths {
		// Expand environment variables in the path
		expandedPath := os.ExpandEnv(basePath)
		filePath := filepath.Join(expandedPath, defaultConfigFile)

		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			return loadConfigFile(filePath)
		}
	}

	// If we get here, try the current working directory as a last resort
	return loadConfigFile(defaultConfigFile)
}

// loadConfigFile loads the configuration from the specified file path
func loadConfigFile(filePath string) error {
	// Expand environment variables in the file path
	expandedPath := os.ExpandEnv(filePath)

	// Try to decode the configuration file
	_, err := toml.DecodeFile(expandedPath, HPipeConfig)
	if err != nil {
		return fmt.Errorf("failed to read configuration file %s: %w", expandedPath, err)
	}

	// Process the configuration (expand environment variables in paths)
	processConfig()

	log.Printf("Loaded configuration from %s", expandedPath)
	return nil
}

// processConfig processes the loaded configuration to expand environment variables
// and perform any other necessary transformations
func processConfig() {
	// Expand environment variables in file paths
	HPipeConfig.KafkaF.KAFKACertFile = expandEnvPath(HPipeConfig.KafkaF.KAFKACertFile)
	HPipeConfig.KafkaF.KAFKAKeyFile = expandEnvPath(HPipeConfig.KafkaF.KAFKAKeyFile)
	HPipeConfig.KafkaF.KAFKACAFile = expandEnvPath(HPipeConfig.KafkaF.KAFKACAFile)
	HPipeConfig.NatsF.NATSCertFile = expandEnvPath(HPipeConfig.NatsF.NATSCertFile)
	HPipeConfig.NatsF.NATSKeyFile = expandEnvPath(HPipeConfig.NatsF.NATSKeyFile)
	HPipeConfig.NatsF.NATSCAFile = expandEnvPath(HPipeConfig.NatsF.NATSCAFile)
	HPipeConfig.QpidF.QpidCertFile = expandEnvPath(HPipeConfig.QpidF.QpidCertFile)
	HPipeConfig.QpidF.QpidKeyFile = expandEnvPath(HPipeConfig.QpidF.QpidKeyFile)
	HPipeConfig.QpidF.QpidCAFile = expandEnvPath(HPipeConfig.QpidF.QpidCAFile)
	HPipeConfig.MQTTF.MQTTCertFile = expandEnvPath(HPipeConfig.MQTTF.MQTTCertFile)
	HPipeConfig.MQTTF.MQTTKeyFile = expandEnvPath(HPipeConfig.MQTTF.MQTTKeyFile)
	HPipeConfig.MQTTF.MQTTCAFile = expandEnvPath(HPipeConfig.MQTTF.MQTTCAFile)
	HPipeConfig.TCPF.TCPCertFile = expandEnvPath(HPipeConfig.TCPF.TCPCertFile)
	HPipeConfig.TCPF.TCPKeyFile = expandEnvPath(HPipeConfig.TCPF.TCPKeyFile)
	HPipeConfig.TCPF.TCPCAFile = expandEnvPath(HPipeConfig.TCPF.TCPCAFile)
	HPipeConfig.RedisF.RedisCertFile = expandEnvPath(HPipeConfig.RedisF.RedisCertFile)
	HPipeConfig.RedisF.RedisKeyFile = expandEnvPath(HPipeConfig.RedisF.RedisKeyFile)
	HPipeConfig.RedisF.RedisCAFile = expandEnvPath(HPipeConfig.RedisF.RedisCAFile)
	HPipeConfig.ZMQF.ZMQCurveServerKey = expandEnvPath(HPipeConfig.ZMQF.ZMQCurveServerKey)
	HPipeConfig.ZMQF.ZMQCurvePublicKey = expandEnvPath(HPipeConfig.ZMQF.ZMQCurvePublicKey)
	HPipeConfig.ZMQF.ZMQCurveSecretKey = expandEnvPath(HPipeConfig.ZMQF.ZMQCurveSecretKey)
	HPipeConfig.GeneralF.DBPath = expandEnvPath(HPipeConfig.GeneralF.DBPath)
}

// expandEnvPath expands environment variables in a file path
func expandEnvPath(path string) string {
	// Skip empty paths
	if path == "" {
		return path
	}

	// Replace environment variables
	expanded := os.ExpandEnv(path)

	// Convert to absolute path if it's not already
	if !filepath.IsAbs(expanded) && !strings.HasPrefix(expanded, "$") {
		if absPath, err := filepath.Abs(expanded); err == nil {
			expanded = absPath
		}
	}

	return expanded
}
