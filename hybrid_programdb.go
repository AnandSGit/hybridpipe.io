package hybridpipe

import (
	"log"

	"github.com/BurntSushi/toml"
)

// MQ Platform Configuration file is in TOML Format. Design decision should be made to place this
// file in target system design. Now it is been put in local folder for ease of use. Please refer
// this file for TOML file format and data.
const MQCONFIGFILE = "./hybridpipe_db.toml"

// MQF define the configuration File content for NATS, RabbitMQ and KAFKA in Golang
// structure format. These configurations are embedded into MQF structure for direct
// access to the data.
type MQF struct {
	NatsF `toml:"NATS"`
	KafkaF `toml:"KAFKA"`
	RabbitMQF `toml:"RABBITMQ"`
}

// RabbitMQF defines the Rabbit MQ Server connection configurations. This struct
// would be extended with User authentication during next phase of HybridPipe
type RabbitMQF struct {
	// TODO: Future expansion would include TLS and other required modules
	RServerPort string `toml:"RServerPort"`
}

// KafkaF defines the KAFKA Server connection configurations. This structure
// will be extended once we are adding the TLS Authentication and Message
// encoding capability.
type KafkaF struct {
	KServer string `toml:"KServers"`
	KLport int `toml:"KLPort"`
	KTimeout int `toml:"KTimeout"`
	KAFKACertFile string `toml:"KAFKACertFile"`
	KAFKAKeyFile string `toml:"KAFKAKeyFile"`
	KAFKACAFile string `toml:"KAFKACAFile"`
}

// NatsF defines the NATS Server connection configurations. It will be extended
// once we are adding TLS feature and Message encoding along with user
// authentication.
type NatsF struct {
	NServer string `toml:"NServers"`
	NLport int `toml:"NLPort"`
	NMport int `toml:"NMPort"`
	NCport int `toml:"NCPort"`
	NATSCertFile string `toml:"NATSCertFile"`
	NATSKeyFile string `toml:"NATSKeyFile"`
	NATSCAFile string `toml:"NATSCAFile"`
	NAllowReconnect bool `toml:"NAllow_Reconnect"`
	NMaxAttempt int `toml:"NMax_Attempt"`
	NReconnectWait int `toml:"NReconnect_Wait"`
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
