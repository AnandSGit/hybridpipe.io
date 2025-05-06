package core

// KafkaConfig holds the configuration for Kafka protocol.
type KafkaConfig struct {
	// KServer is the Kafka server address
	KServer string
	// KLport is the Kafka server port
	KLport int
	// KTimeout is the Kafka connection timeout in seconds
	KTimeout int
	// KAFKACertFile is the path to the client certificate file
	KAFKACertFile string
	// KAFKAKeyFile is the path to the client key file
	KAFKAKeyFile string
	// KAFKACAFile is the path to the CA certificate file
	KAFKACAFile string
}

// NATSConfig holds the configuration for NATS protocol.
type NATSConfig struct {
	// NServer is the NATS server address
	NServer string
	// NLport is the NATS server port
	NLport int
	// NReconnectWait is the reconnect wait time in seconds
	NReconnectWait int
	// NMaxReconnects is the maximum number of reconnect attempts
	NMaxReconnects int
	// NATSCertFile is the path to the client certificate file
	NATSCertFile string
	// NATSKeyFile is the path to the client key file
	NATSKeyFile string
	// NATSCAFile is the path to the CA certificate file
	NATSCAFile string
}

// RabbitMQConfig holds the configuration for RabbitMQ protocol.
type RabbitMQConfig struct {
	// RServerPort is the RabbitMQ server address with port
	RServerPort string
}

// MQTTConfig holds the configuration for MQTT protocol.
type MQTTConfig struct {
	// MQTTBroker is the MQTT broker address
	MQTTBroker string
	// MQTTClientID is the MQTT client ID
	MQTTClientID string
	// MQTTUsername is the MQTT username
	MQTTUsername string
	// MQTTPassword is the MQTT password
	MQTTPassword string
	// MQTTCleanSession indicates whether to use a clean session
	MQTTCleanSession bool
	// MQTTQoS is the MQTT quality of service level
	MQTTQoS int
	// MQTTRetain indicates whether to retain messages
	MQTTRetain bool
	// MQTTTLSEnabled indicates whether to use TLS
	MQTTTLSEnabled bool
	// MQTTCertFile is the path to the client certificate file
	MQTTCertFile string
	// MQTTKeyFile is the path to the client key file
	MQTTKeyFile string
	// MQTTCAFile is the path to the CA certificate file
	MQTTCAFile string
	// MQTTConnectTimeout is the connection timeout in seconds
	MQTTConnectTimeout int
	// MQTTKeepAlive is the keep alive interval in seconds
	MQTTKeepAlive int
	// MQTTPublishTimeout is the publish timeout in seconds
	MQTTPublishTimeout int
}

// QpidConfig holds the configuration for Apache Qpid protocol.
type QpidConfig struct {
	// QpidServer is the Qpid server address
	QpidServer string
	// QpidPort is the Qpid server port
	QpidPort int
	// QpidTLSEnabled indicates whether to use TLS
	QpidTLSEnabled bool
	// QpidCertFile is the path to the client certificate file
	QpidCertFile string
	// QpidKeyFile is the path to the client key file
	QpidKeyFile string
	// QpidCAFile is the path to the CA certificate file
	QpidCAFile string
}

// NSQConfig holds the configuration for NSQ protocol.
type NSQConfig struct {
	// NSQLookupdAddresses is a comma-separated list of NSQ lookupd addresses
	NSQLookupdAddresses string
	// NSQDAddress is the NSQD address
	NSQDAddress string
	// NSQDPort is the NSQD port
	NSQDPort int
	// NSQClientID is the NSQ client ID
	NSQClientID string
	// NSQAuthSecret is the NSQ authentication secret
	NSQAuthSecret string
	// NSQMaxInFlight is the maximum number of in-flight messages
	NSQMaxInFlight int
	// NSQRequeueDelay is the requeue delay in seconds
	NSQRequeueDelay int
	// NSQConnectTimeout is the connection timeout in seconds
	NSQConnectTimeout int
}

// TCPConfig holds the configuration for TCP protocol.
type TCPConfig struct {
	// TCPMode can be "client" or "server"
	TCPMode string
	// TCPHost is the TCP host address
	TCPHost string
	// TCPPort is the TCP port
	TCPPort int
	// TCPTLSEnabled indicates whether to use TLS
	TCPTLSEnabled bool
	// TCPCertFile is the path to the certificate file
	TCPCertFile string
	// TCPKeyFile is the path to the key file
	TCPKeyFile string
	// TCPCAFile is the path to the CA certificate file
	TCPCAFile string
	// TCPConnectTimeout is the connection timeout in seconds
	TCPConnectTimeout int
	// TCPKeepAlive indicates whether to use TCP keep-alive
	TCPKeepAlive bool
	// TCPKeepAlivePeriod is the keep-alive period in seconds
	TCPKeepAlivePeriod int
}

// RedisConfig holds the configuration for Redis protocol.
type RedisConfig struct {
	// RedisAddress is the Redis server address
	RedisAddress string
	// RedisPort is the Redis server port
	RedisPort int
	// RedisUsername is the Redis username
	RedisUsername string
	// RedisPassword is the Redis password
	RedisPassword string
	// RedisDB is the Redis database number
	RedisDB int
	// RedisTLSEnabled indicates whether to use TLS
	RedisTLSEnabled bool
	// RedisCertFile is the path to the client certificate file
	RedisCertFile string
	// RedisKeyFile is the path to the client key file
	RedisKeyFile string
	// RedisCAFile is the path to the CA certificate file
	RedisCAFile string
	// RedisConnectTimeout is the connection timeout in seconds
	RedisConnectTimeout int
	// RedisReadTimeout is the read timeout in seconds
	RedisReadTimeout int
	// RedisWriteTimeout is the write timeout in seconds
	RedisWriteTimeout int
	// RedisPoolSize is the connection pool size
	RedisPoolSize int
	// RedisMinIdleConns is the minimum number of idle connections
	RedisMinIdleConns int
}

// NetChanConfig holds the configuration for NetChan protocol.
type NetChanConfig struct {
	// ChannelBufferSize is the buffer size for channels
	ChannelBufferSize int
	// SendTimeout is the send timeout in milliseconds
	SendTimeout int
}

// GetKafkaConfig returns the Kafka configuration from the global config.
func GetKafkaConfig() KafkaConfig {
	config := GetConfig()
	return KafkaConfig{
		KServer:       config.Kafka.Brokers[0],
		KLport:        8080, // Default port
		KTimeout:      config.Kafka.ConnectTimeout,
		KAFKACertFile: config.Kafka.CertFile,
		KAFKAKeyFile:  config.Kafka.KeyFile,
		KAFKACAFile:   config.Kafka.CAFile,
	}
}

// GetNATSConfig returns the NATS configuration from the global config.
func GetNATSConfig() NATSConfig {
	config := GetConfig()
	return NATSConfig{
		NServer:        config.NATS.Servers[0],
		NLport:         4222, // Default port
		NReconnectWait: 1,    // Default reconnect wait
		NMaxReconnects: 10,   // Default max reconnects
		NATSCertFile:   config.NATS.CertFile,
		NATSKeyFile:    config.NATS.KeyFile,
		NATSCAFile:     config.NATS.CAFile,
	}
}

// GetRabbitMQConfig returns the RabbitMQ configuration from the global config.
func GetRabbitMQConfig() RabbitMQConfig {
	config := GetConfig()
	return RabbitMQConfig{
		RServerPort: config.RabbitMQ.URL,
	}
}

// GetMQTTConfig returns the MQTT configuration from the global config.
func GetMQTTConfig() MQTTConfig {
	config := GetConfig()
	return MQTTConfig{
		MQTTBroker:         config.MQTT.Broker,
		MQTTClientID:       config.MQTT.ClientID,
		MQTTUsername:       config.MQTT.Username,
		MQTTPassword:       config.MQTT.Password,
		MQTTCleanSession:   config.MQTT.CleanSession,
		MQTTQoS:            config.MQTT.QoS,
		MQTTRetain:         config.MQTT.Retained,
		MQTTTLSEnabled:     true,
		MQTTCertFile:       config.MQTT.CertFile,
		MQTTKeyFile:        config.MQTT.KeyFile,
		MQTTCAFile:         config.MQTT.CAFile,
		MQTTConnectTimeout: config.MQTT.ConnectTimeout,
		MQTTKeepAlive:      60, // Default keep alive
		MQTTPublishTimeout: 5,  // Default publish timeout
	}
}

// GetQpidConfig returns the Qpid configuration from the global config.
func GetQpidConfig() QpidConfig {
	config := GetConfig()
	return QpidConfig{
		QpidServer:     "localhost", // Default server
		QpidPort:       5672,        // Default port
		QpidTLSEnabled: true,
		QpidCertFile:   config.Qpid.CertFile,
		QpidKeyFile:    config.Qpid.KeyFile,
		QpidCAFile:     config.Qpid.CAFile,
	}
}

// GetNSQConfig returns the NSQ configuration from the global config.
func GetNSQConfig() NSQConfig {
	config := GetConfig()
	return NSQConfig{
		NSQLookupdAddresses: "",                  // Default to empty
		NSQDAddress:         config.NSQ.NSQDAddress,
		NSQDPort:            config.NSQ.NSQDPort,
		NSQClientID:         config.NSQ.ClientID,
		NSQAuthSecret:       config.NSQ.AuthSecret,
		NSQMaxInFlight:      config.NSQ.MaxInFlight,
		NSQRequeueDelay:     config.NSQ.RequeueDelay,
		NSQConnectTimeout:   config.NSQ.ConnectTimeout,
	}
}

// GetTCPConfig returns the TCP configuration from the global config.
func GetTCPConfig() TCPConfig {
	config := GetConfig()
	return TCPConfig{
		TCPMode:           "client", // Default mode
		TCPHost:           config.TCP.Host,
		TCPPort:           config.TCP.Port,
		TCPTLSEnabled:     true,
		TCPCertFile:       config.TCP.CertFile,
		TCPKeyFile:        config.TCP.KeyFile,
		TCPCAFile:         config.TCP.CAFile,
		TCPConnectTimeout: config.TCP.ConnectTimeout,
		TCPKeepAlive:      true,
		TCPKeepAlivePeriod: 60, // Default keep alive period
	}
}

// GetRedisConfig returns the Redis configuration from the global config.
func GetRedisConfig() RedisConfig {
	config := GetConfig()
	return RedisConfig{
		RedisAddress:       config.Redis.Address,
		RedisPort:          6379, // Default port
		RedisUsername:      "",   // Default to empty
		RedisPassword:      config.Redis.Password,
		RedisDB:            config.Redis.DB,
		RedisTLSEnabled:    false,
		RedisCertFile:      "",
		RedisKeyFile:       "",
		RedisCAFile:        "",
		RedisConnectTimeout: config.Redis.ConnectTimeout,
		RedisReadTimeout:    30, // Default read timeout
		RedisWriteTimeout:   30, // Default write timeout
		RedisPoolSize:       10, // Default pool size
		RedisMinIdleConns:   2,  // Default min idle connections
	}
}

// GetNetChanConfig returns the NetChan configuration from the global config.
func GetNetChanConfig() NetChanConfig {
	return NetChanConfig{
		ChannelBufferSize: 10,   // Default buffer size
		SendTimeout:       1000, // Default send timeout
	}
}
