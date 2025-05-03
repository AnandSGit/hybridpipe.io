package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)

// Config holds the configuration for HybridPipe.
type Config struct {
	// General configuration
	General struct {
		LogLevel string `toml:"LogLevel"`
	} `toml:"GENERAL"`

	// Kafka configuration
	Kafka struct {
		Brokers          []string `toml:"KafkaBrokers"`
		ClientID         string   `toml:"KafkaClientID"`
		ConsumerGroup    string   `toml:"KafkaConsumerGroup"`
		SecurityProtocol string   `toml:"KafkaSecurityProtocol"`
		SASLMechanism    string   `toml:"KafkaSASLMechanism"`
		SASLUsername     string   `toml:"KafkaSASLUsername"`
		SASLPassword     string   `toml:"KafkaSASLPassword"`
		CertFile         string   `toml:"KafkaCertFile"`
		KeyFile          string   `toml:"KafkaKeyFile"`
		CAFile           string   `toml:"KafkaCAFile"`
		VerifySSL        bool     `toml:"KafkaVerifySSL"`
		ConnectTimeout   int      `toml:"KafkaConnectTimeout"`
	} `toml:"KAFKA"`

	// NATS configuration
	NATS struct {
		Servers        []string `toml:"NATSServers"`
		Username       string   `toml:"NATSUsername"`
		Password       string   `toml:"NATSPassword"`
		Token          string   `toml:"NATSToken"`
		CertFile       string   `toml:"NATSCertFile"`
		KeyFile        string   `toml:"NATSKeyFile"`
		CAFile         string   `toml:"NATSCAFile"`
		ConnectTimeout int      `toml:"NATSConnectTimeout"`
	} `toml:"NATS"`

	// RabbitMQ configuration
	RabbitMQ struct {
		URL            string `toml:"RabbitMQURL"`
		Username       string `toml:"RabbitMQUsername"`
		Password       string `toml:"RabbitMQPassword"`
		VHost          string `toml:"RabbitMQVHost"`
		CertFile       string `toml:"RabbitMQCertFile"`
		KeyFile        string `toml:"RabbitMQKeyFile"`
		CAFile         string `toml:"RabbitMQCAFile"`
		ConnectTimeout int    `toml:"RabbitMQConnectTimeout"`
	} `toml:"RABBITMQ"`

	// AMQP 1.0 configuration
	AMQP1 struct {
		URL            string `toml:"AMQP1URL"`
		Username       string `toml:"AMQP1Username"`
		Password       string `toml:"AMQP1Password"`
		CertFile       string `toml:"AMQP1CertFile"`
		KeyFile        string `toml:"AMQP1KeyFile"`
		CAFile         string `toml:"AMQP1CAFile"`
		ConnectTimeout int    `toml:"AMQP1ConnectTimeout"`
	} `toml:"AMQP1"`

	// MQTT configuration
	MQTT struct {
		Broker         string `toml:"MQTTBroker"`
		ClientID       string `toml:"MQTTClientID"`
		Username       string `toml:"MQTTUsername"`
		Password       string `toml:"MQTTPassword"`
		CleanSession   bool   `toml:"MQTTCleanSession"`
		QoS            int    `toml:"MQTTQoS"`
		Retained       bool   `toml:"MQTTRetained"`
		CertFile       string `toml:"MQTTCertFile"`
		KeyFile        string `toml:"MQTTKeyFile"`
		CAFile         string `toml:"MQTTCAFile"`
		ConnectTimeout int    `toml:"MQTTConnectTimeout"`
	} `toml:"MQTT"`

	// Apache Qpid configuration
	Qpid struct {
		URL            string `toml:"QpidURL"`
		Username       string `toml:"QpidUsername"`
		Password       string `toml:"QpidPassword"`
		CertFile       string `toml:"QpidCertFile"`
		KeyFile        string `toml:"QpidKeyFile"`
		CAFile         string `toml:"QpidCAFile"`
		ConnectTimeout int    `toml:"QpidConnectTimeout"`
	} `toml:"QPID"`

	// NSQ configuration
	NSQ struct {
		LookupdAddresses []string `toml:"NSQLookupdAddresses"`
		NSQDAddress      string   `toml:"NSQDAddress"`
		NSQDPort         int      `toml:"NSQDPort"`
		ClientID         string   `toml:"NSQClientID"`
		AuthSecret       string   `toml:"NSQAuthSecret"`
		MaxInFlight      int      `toml:"NSQMaxInFlight"`
		MaxAttempts      int      `toml:"NSQMaxAttempts"`
		RequeueDelay     int      `toml:"NSQRequeueDelay"`
		ConnectTimeout   int      `toml:"NSQConnectTimeout"`
	} `toml:"NSQ"`

	// TCP configuration
	TCP struct {
		Host           string `toml:"TCPHost"`
		Port           int    `toml:"TCPPort"`
		CertFile       string `toml:"TCPCertFile"`
		KeyFile        string `toml:"TCPKeyFile"`
		CAFile         string `toml:"TCPCAFile"`
		ConnectTimeout int    `toml:"TCPConnectTimeout"`
	} `toml:"TCP"`

	// Redis configuration
	Redis struct {
		Address        string `toml:"RedisAddress"`
		Password       string `toml:"RedisPassword"`
		DB             int    `toml:"RedisDB"`
		ConnectTimeout int    `toml:"RedisConnectTimeout"`
	} `toml:"REDIS"`

	// NetChan configuration
	NetChan struct {
		Host           string `toml:"NetChanHost"`
		Port           int    `toml:"NetChanPort"`
		CertFile       string `toml:"NetChanCertFile"`
		KeyFile        string `toml:"NetChanKeyFile"`
		CAFile         string `toml:"NetChanCAFile"`
		ConnectTimeout int    `toml:"NetChanConnectTimeout"`
	} `toml:"NETCHAN"`

	// Tracing configuration
	Tracing struct {
		Enabled          bool    `toml:"Enabled"`
		ServiceName      string  `toml:"ServiceName"`
		SamplingRate     float64 `toml:"SamplingRate"`
		ExporterType     string  `toml:"ExporterType"`
		ExporterEndpoint string  `toml:"ExporterEndpoint"`
	} `toml:"TRACING"`

	// Monitoring configuration
	Monitoring struct {
		Enabled          bool   `toml:"Enabled"`
		MetricsInterval  string `toml:"MetricsInterval"`
		ExporterType     string `toml:"ExporterType"`
		ExporterEndpoint string `toml:"ExporterEndpoint"`
	} `toml:"MONITORING"`
}

var (
	// globalConfig holds the global configuration.
	globalConfig Config
	// configMutex protects concurrent access to the configuration.
	configMutex sync.RWMutex
	// configLoaded indicates whether the configuration has been loaded.
	configLoaded bool
)

// ReadConfig reads the configuration from a file.
func ReadConfig() error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// Check if the configuration has already been loaded
	if configLoaded {
		return nil
	}

	// Get the configuration file path from the environment variable
	configPath := os.Getenv("HYBRIDPIPE_CONFIG")
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return readConfigFile(configPath)
		}
	}

	// Try to find the configuration file in standard locations
	locations := []string{
		"./hybridpipe_db.toml",
		"/etc/hybridpipe/hybridpipe_db.toml",
		"/opt/hybridpipe/config/hybridpipe_db.toml",
	}

	// Add user-specific location
	if homeDir, err := os.UserHomeDir(); err == nil {
		locations = append(locations, filepath.Join(homeDir, ".config/hybridpipe/hybridpipe_db.toml"))
	}

	// Try each location
	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			return readConfigFile(location)
		}
	}

	// No configuration file found, use default values
	configLoaded = true
	return nil
}

// readConfigFile reads the configuration from the specified file.
func readConfigFile(path string) error {
	_, err := toml.DecodeFile(path, &globalConfig)
	if err != nil {
		return fmt.Errorf("failed to decode config file: %w", err)
	}

	configLoaded = true
	return nil
}

// GetConfig returns a copy of the global configuration.
func GetConfig() Config {
	configMutex.RLock()
	defer configMutex.RUnlock()

	// Return a copy to prevent modification of the global configuration
	return globalConfig
}

// SetConfig sets the global configuration.
func SetConfig(config Config) {
	configMutex.Lock()
	defer configMutex.Unlock()

	globalConfig = config
	configLoaded = true
}
