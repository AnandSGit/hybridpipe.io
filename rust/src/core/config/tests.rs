#[cfg(test)]
mod tests {
    use super::*;
    use std::env;
    use std::fs;
    use std::path::Path;
    use tempfile::tempdir;

    #[test]
    fn test_default_config() {
        // Test that the default configuration can be loaded
        let config = load_config("nonexistent_file.toml").expect("Failed to load default config");
        
        // Check that the default values are set
        assert_eq!(config.nats.nats_server, "nats://localhost:4222");
        assert_eq!(config.kafka.kafka_brokers, "localhost:9092");
        assert_eq!(config.rabbitmq.rabbitmq_server, "amqp://guest:guest@localhost:5672/");
        assert_eq!(config.amqp1.amqp_server, "amqp://0.0.0.0:5672/");
        assert_eq!(config.mqtt.mqtt_broker, "tcp://localhost:1883");
        assert_eq!(config.qpid.qpid_server, "localhost");
        assert_eq!(config.nsq.nsq_server, "localhost");
        assert_eq!(config.tcp.tcp_host, "localhost");
        assert_eq!(config.redis.redis_server, "redis://localhost:6379/");
        assert_eq!(config.netchan.netchan_server, "localhost:9000");
    }

    #[test]
    fn test_load_config_from_file() {
        // Create a temporary directory
        let dir = tempdir().expect("Failed to create temporary directory");
        let config_path = dir.path().join("test_config.toml");
        
        // Create a test configuration file
        let config_content = r#"
            [nats]
            nats_server = "nats://example.com:4222"
            
            [kafka]
            kafka_brokers = "example.com:9092"
            
            [rabbitmq]
            rabbitmq_server = "amqp://user:pass@example.com:5672/"
        "#;
        fs::write(&config_path, config_content).expect("Failed to write test config file");
        
        // Load the configuration from the file
        let config = load_config(config_path.to_str().unwrap()).expect("Failed to load config from file");
        
        // Check that the values from the file are used
        assert_eq!(config.nats.nats_server, "nats://example.com:4222");
        assert_eq!(config.kafka.kafka_brokers, "example.com:9092");
        assert_eq!(config.rabbitmq.rabbitmq_server, "amqp://user:pass@example.com:5672/");
        
        // Check that default values are used for fields not in the file
        assert_eq!(config.amqp1.amqp_server, "amqp://0.0.0.0:5672/");
        assert_eq!(config.mqtt.mqtt_broker, "tcp://localhost:1883");
    }

    #[test]
    fn test_load_config_from_env() {
        // Set environment variables
        env::set_var("HYBRIDPIPE_NATS_NATS_SERVER", "nats://env.example.com:4222");
        env::set_var("HYBRIDPIPE_KAFKA_KAFKA_BROKERS", "env.example.com:9092");
        
        // Load the configuration
        let config = load_config("nonexistent_file.toml").expect("Failed to load config with env vars");
        
        // Check that the values from environment variables are used
        assert_eq!(config.nats.nats_server, "nats://env.example.com:4222");
        assert_eq!(config.kafka.kafka_brokers, "env.example.com:9092");
        
        // Clean up
        env::remove_var("HYBRIDPIPE_NATS_NATS_SERVER");
        env::remove_var("HYBRIDPIPE_KAFKA_KAFKA_BROKERS");
    }

    #[test]
    fn test_get_config() {
        // Initialize the configuration
        init_config().expect("Failed to initialize config");
        
        // Get the configuration
        let config = get_config().expect("Failed to get config");
        
        // Check that the configuration is valid
        assert_eq!(config.nats.nats_server, "nats://localhost:4222");
    }

    #[test]
    fn test_get_protocol_config() {
        // Initialize the configuration
        init_config().expect("Failed to initialize config");
        
        // Get the configuration for a specific protocol
        let nats_config = get_protocol_config(BrokerType::NATS).expect("Failed to get NATS config");
        let nats_config = nats_config.downcast_ref::<NATSConfig>().expect("Failed to downcast to NATSConfig");
        
        // Check that the configuration is valid
        assert_eq!(nats_config.nats_server, "nats://localhost:4222");
        
        // Get the configuration for another protocol
        let kafka_config = get_protocol_config(BrokerType::KAFKA).expect("Failed to get Kafka config");
        let kafka_config = kafka_config.downcast_ref::<KafkaConfig>().expect("Failed to downcast to KafkaConfig");
        
        // Check that the configuration is valid
        assert_eq!(kafka_config.kafka_brokers, "localhost:9092");
        
        // Try to get the configuration for an unregistered protocol
        let result = get_protocol_config(BrokerType::RESERVED1);
        assert!(result.is_err());
        match result {
            Err(Error::BrokerTypeNotRegistered(broker_type)) => {
                assert_eq!(broker_type, "RESERVED1");
            }
            _ => panic!("Expected BrokerTypeNotRegistered error"),
        }
    }
}
