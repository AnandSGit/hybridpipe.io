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
