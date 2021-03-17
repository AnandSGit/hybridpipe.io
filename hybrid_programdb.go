package hybridpipe

import (
	"log"
	
	"github.com/BurntSushi/toml"
)

const HPDBFILE = "/hybridpipe_db.toml"

type HybridDB struct {
	NatsF     `toml:"NATS"`
	KafkaF    `toml:"KAFKA"`
	RabbitMQF `toml:"RABBITMQ"`
	AMQPF     `toml:"AMQP1"`
	GeneralF  `toml:"GENERAL"`
}

type RabbitMQF struct {
	// TODO: Future expansion would include TLS and other required modules
	RServerPort string `toml:"RServerPort"`
}

type KafkaF struct {
	KServer       string `toml:"KServers"`
	KLport        int    `toml:"KLPort"`
	KTimeout      int    `toml:"KTimeout"`
	KAFKACertFile string `toml:"KAFKACertFile"`
	KAFKAKeyFile  string `toml:"KAFKAKeyFile"`
	KAFKACAFile   string `toml:"KAFKACAFile"`
}

type NatsF struct {
	NServer         string `toml:"NServers"`
	NLport          int    `toml:"NLPort"`
	NMport          int    `toml:"NMPort"`
	NCport          int    `toml:"NCPort"`
	NATSCertFile    string `toml:"NATSCertFile"`
	NATSKeyFile     string `toml:"NATSKeyFile"`
	NATSCAFile      string `toml:"NATSCAFile"`
	NAllowReconnect bool   `toml:"NAllow_Reconnect"`
	NMaxAttempt     int    `toml:"NMax_Attempt"`
	NReconnectWait  int    `toml:"NReconnect_Wait"`
	NTimeout        int    `toml:"NTimeout"`
}

type AMQPF struct {
	AMQPServer string `toml:"AMQPServer"`
	AMQPPort   string `toml:"AMQPPort"`
}

type GeneralF struct {
	DBPath string `toml:"DBLocation"`
}

var (
	HPipeConfig = new(HybridDB)
)

// ReadConfig defines the function to read the HybridPipe configuration file.
func ReadConfig() error {
	
	if _, e := toml.DecodeFile(HPDBFILE, &HPipeConfig); e != nil {
		log.Printf("Configuration File - %v Read Error: %v", HPDBFILE, e)
		return e
	}
	return nil
}
