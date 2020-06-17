package hybridpipe

// ALL THE MESSAGE QUEUE CONFIGURATION FILE RELATED DEFINITIONS WOULD BE DONE
// HERE. IF ANY TLS or USER CREDENTIAL CONFIGURATION FILES DEFINED IN FUTURE,
// THOSE WOULD BE UPDATED IN THE DEFINED CONFIGURATION FILE AND HERE.

// -----------------------------------------------------------------------------
// IMPORT Section
// -----------------------------------------------------------------------------
import (
	"log"

	"github.com/BurntSushi/toml"
)

// MQCONFIGFILE define the file name and location of the Client config file for
// MQ Platform (contains NATS, RabbitMQ and KAFKA related client configurations) in
// TOML format. This configuration file should be placed under common Program config
// files location to make it easier to deploy
const MQCONFIGFILE = "./drouter_db.toml"

// -----------------------------------------------------------------------------
// CLIENT CONFIGURATION FILE HANDLING
// -----------------------------------------------------------------------------
/**
Sample Configuration File below

[NATS]
# Nats Server List. DEFAULT = 0.0.0.0
NServers           = "localhost"
# Listening port. DEFAULT = 4222
NLPort             = 4222
# Monitoring Port (Http Port) DEFAULT = 8222
NMPort             = 8222
# Cluster Family Port DEFAULT = 6222
NCPort             = 6222
# TLS Configuration (Please specify the Fully qualified name)
NATSCertFile       = "./platforms/tls/nats-client.pem"
NATSKeyFile        = "./platforms/tls/nats-client.key"
NATSCAFile         = "./platforms/tls/nats-server-ca.pem"
# Allows reconnect in case of drop. DEFAULT = true
NAllow_Reconnect   = true
# Number of time re-connect attempt to be done. DEFAULT = 10
NMax_Attempt       = 10
# Wait time before trying to re-connect. DEFAULT = 5
NReconnect_Wait    = 5
# Time (in seconds) before timeout event raised and handled. DEFAULT = 2
NTimeout           = 2

[KAFKA]
# Kafka Server List. DEFAULT = 0.0.0.0
KServers   = "Anand-iMac"
# Listening port. DEFAULT = 9092
KLPort     = 9093
# Timeout of KAFKA Server connection drop / Keepalive.
KTimeout   = 10
# TLS Configuration Data
KAFKACertFile       = "./platforms/tls/mytestclient.cert.pem"
KAFKAKeyFile        = "./platforms/tls/mytestclient.key.pem"
KAFKACAFile         = "./platforms/tls/rootca.cert.pem"

[RABBITMQ]
# RABBIT MQ Server and Port details.
RServers    = "amqp://guest:guest@localhost:5672/"
**/

// MQF define the configuration File content for NATS, RabbitMQ and KAFKA in Golang
// structure format. These configurations are embedded into MQF structure for direct
// access to the data.
type MQF struct {

	// NATS Input Data Embedded
	NatsF `toml:"NATS"`
	// KAFKA Input Data Embedded
	KafkaF `toml:"KAFKA"`
	// RabbitMQ Input Data Embedded
	RabbitMQF `toml:"RABBITMQ"`
}

// RabbitMQF defines the Rabbit MQ Server connection configurations. This struct
// would be extended with User authentication during next phase of HybridPipe
type RabbitMQF struct {
	// RServer defines the RabbitMQ Server
	RServerPort string `toml:"RServerPort"`
}

// KafkaF defines the KAFKA Server connection configurations. This structure
// will be extended once we are adding the TLS Authentication and Message
// encoding capability.
type KafkaF struct {

	// KServer defines the Kafka Server URI/Nodename. DEFAULT = localhost
	KServer string `toml:"KServers"`
	// KLport defines the Server listening port for Kafka. DEFAULT = 9092
	KLport int `toml:"KLPort"`
	// KTimeout defines the timeout for Kafka Server connection.
	// DEFAULT = 2 (in seconds)
	KTimeout int `toml:"KTimeout"`
	// KAFKACertFile defines the TLS Certificate File for KAFKA. No DEFAULT
	KAFKACertFile string `toml:"KAFKACertFile"`
	// KAFKAKeyFile defines the TLS Key File for KAFKA. No DEFAULT
	KAFKAKeyFile string `toml:"KAFKAKeyFile"`
	// KAFKACAFile defines the KAFKA Certification Authority. No DEFAULT
	KAFKACAFile string `toml:"KAFKACAFile"`
}

// NatsF defines the NATS Server connection configurations. It will be extended
// once we are adding TLS feature and Message encoding along with user
// authentication.
type NatsF struct {

	// NServer defines the NATS Server URI/Node name. DEFAULT = localhost
	NServer string `toml:"NServers"`
	// NLport defines the NATS Server listening port. DEFAULT = 4222
	NLport int `toml:"NLPort"`
	// NMport defines the NATS Server Monitoring HTTP port. DEFAULT = 8222
	NMport int `toml:"NMPort"`
	// NCport defines the NATS Server Cluster joining port. DEFAULT = 6222
	NCport int `toml:"NCPort"`
	// NATSCertFile defines the TLS Certification File
	NATSCertFile string `toml:"NATSCertFile"`
	// NATSKeyFile defines the TLS Public Key File
	NATSKeyFile string `toml:"NATSKeyFile"`
	//  NATSCAFile defines the TLS Certification Authority File
	NATSCAFile string `toml:"NATSCAFile"`
	// NAllowReconnect defines the flag if this client should be allowed to
	// re-connect in case of drop of connection. DEFAULT = true
	NAllowReconnect bool `toml:"NAllow_Reconnect"`
	// NMaxAttempt defines the number of times this client should be allowed
	// to attempt to re-connect, if Reconnect flag is enabled. DEFAULT = 10
	NMaxAttempt int `toml:"NMax_Attempt"`
	// NReconnectWait defines the interval in seconds after re-connect
	// attempt should be done after conn drop. DEFAULT = 5 (in seconds)
	NReconnectWait int `toml:"NReconnect_Wait"`
	// NTimeout defines the timeout for NATS connection.
	NTimeout int `toml:"NTimeout"`
}

// ReadConfig defines the function to read the client side configuration file any
// configuration data, which need / should be provided by MQ user would be taken
// directly from the user by asking to fill a structure.  THIS DATA DETAILS
// SHOULD BE DEFINED AS PART OF INTERFACE DEFINITION.
func ReadConfig(fc interface{}) error {

	if _, e := toml.DecodeFile(MQCONFIGFILE, fc); e != nil {
		log.Printf("Configuration File - %v Read Error: %v", MQCONFIGFILE, e)
		return e
	}
	return nil
}
