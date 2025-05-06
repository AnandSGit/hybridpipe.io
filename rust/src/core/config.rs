// Configuration module for HybridPipe
//
// This module defines the configuration system for HybridPipe.

use config::{Config, ConfigError, Environment, File};
use serde::{Deserialize, Serialize};
use std::any::Any;
use std::env;
use std::path::Path;
use std::sync::OnceLock;

use super::Error;

// Global configuration instance
static CONFIG: OnceLock<HybridPipeConfig> = OnceLock::new();

/// HybridPipeConfig is the main configuration struct for HybridPipe.
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct HybridPipeConfig {
    /// NATS configuration
    #[serde(default)]
    pub nats: NATSConfig,

    /// Kafka configuration
    #[serde(default)]
    pub kafka: KafkaConfig,

    /// RabbitMQ configuration
    #[serde(default)]
    pub rabbitmq: RabbitMQConfig,

    /// AMQP 1.0 configuration
    #[serde(default)]
    pub amqp1: AMQP1Config,

    /// MQTT configuration
    #[serde(default)]
    pub mqtt: MQTTConfig,

    /// Qpid configuration
    #[serde(default)]
    pub qpid: QpidConfig,

    /// NSQ configuration
    #[serde(default)]
    pub nsq: NSQConfig,

    /// TCP configuration
    #[serde(default)]
    pub tcp: TCPConfig,

    /// Redis configuration
    #[serde(default)]
    pub redis: RedisConfig,

    /// NetChan configuration
    #[serde(default)]
    pub netchan: NetChanConfig,

    /// General configuration
    #[serde(default)]
    pub general: GeneralConfig,
}

/// NATS configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct NATSConfig {
    /// NATS server address
    #[serde(default = "default_nats_server")]
    pub n_servers: String,

    /// NATS client port
    #[serde(default = "default_nats_client_port")]
    pub n_lport: u16,

    /// NATS monitoring port
    #[serde(default = "default_nats_monitoring_port")]
    pub n_mport: u16,

    /// NATS cluster port
    #[serde(default = "default_nats_cluster_port")]
    pub n_cport: u16,

    /// NATS client certificate file
    #[serde(default)]
    pub nats_cert_file: String,

    /// NATS client key file
    #[serde(default)]
    pub nats_key_file: String,

    /// NATS CA certificate file
    #[serde(default)]
    pub nats_ca_file: String,

    /// Allow reconnection
    #[serde(default = "default_true")]
    pub n_allow_reconnect: bool,

    /// Maximum reconnection attempts
    #[serde(default = "default_max_attempts")]
    pub n_max_attempt: u32,

    /// Reconnection wait time in seconds
    #[serde(default = "default_reconnect_wait")]
    pub n_reconnect_wait: u32,

    /// Connection timeout in seconds
    #[serde(default = "default_timeout")]
    pub n_timeout: u32,
}

/// Kafka configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct KafkaConfig {
    /// Kafka server address
    #[serde(default = "default_kafka_server")]
    pub k_servers: String,

    /// Kafka port
    #[serde(default = "default_kafka_port")]
    pub k_lport: u16,

    /// Connection timeout in seconds
    #[serde(default = "default_timeout")]
    pub k_timeout: u32,

    /// Kafka client certificate file
    #[serde(default)]
    pub kafka_cert_file: String,

    /// Kafka client key file
    #[serde(default)]
    pub kafka_key_file: String,

    /// Kafka CA certificate file
    #[serde(default)]
    pub kafka_ca_file: String,
}

// Default values for configuration
fn default_nats_server() -> String {
    "localhost".to_string()
}
fn default_nats_client_port() -> u16 {
    4222
}
fn default_nats_monitoring_port() -> u16 {
    8222
}
fn default_nats_cluster_port() -> u16 {
    6222
}
fn default_kafka_server() -> String {
    "localhost".to_string()
}
fn default_kafka_port() -> u16 {
    9093
}
fn default_true() -> bool {
    true
}
fn default_max_attempts() -> u32 {
    10
}
fn default_reconnect_wait() -> u32 {
    5
}
fn default_timeout() -> u32 {
    10
}

// Additional configuration structs would be defined here

/// RabbitMQ configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct RabbitMQConfig {
    // Fields would be defined here
    #[serde(default = "default_rabbitmq_server")]
    pub r_server_port: String,
}

fn default_rabbitmq_server() -> String {
    "amqp://guest:guest@localhost:5672/".to_string()
}

/// AMQP 1.0 configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct AMQP1Config {
    // Fields would be defined here
    #[serde(default = "default_amqp_server")]
    pub amqp_server: String,

    #[serde(default = "default_amqp_port")]
    pub amqp_port: u16,
}

fn default_amqp_server() -> String {
    "amqp://0.0.0.0:5672/".to_string()
}
fn default_amqp_port() -> u16 {
    5672
}

/// MQTT configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct MQTTConfig {
    // Fields would be defined here
    #[serde(default = "default_mqtt_broker")]
    pub mqtt_broker: String,
}

fn default_mqtt_broker() -> String {
    "tcp://localhost:1883".to_string()
}

/// Qpid configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct QpidConfig {
    // Fields would be defined here
    #[serde(default = "default_qpid_server")]
    pub qpid_server: String,
}

fn default_qpid_server() -> String {
    "localhost".to_string()
}

/// NSQ configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct NSQConfig {
    // Fields would be defined here
    #[serde(default = "default_nsq_server")]
    pub nsq_server: String,
}

fn default_nsq_server() -> String {
    "localhost".to_string()
}

/// TCP configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct TCPConfig {
    // Fields would be defined here
    #[serde(default = "default_tcp_host")]
    pub tcp_host: String,
}

fn default_tcp_host() -> String {
    "localhost".to_string()
}

/// Redis configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct RedisConfig {
    // Fields would be defined here
    #[serde(default = "default_redis_address")]
    pub redis_address: String,
}

fn default_redis_address() -> String {
    "localhost".to_string()
}

/// NetChan configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct NetChanConfig {
    // Fields would be defined here
    #[serde(default = "default_netchan_server")]
    pub netchan_server: String,
}

fn default_netchan_server() -> String {
    "localhost".to_string()
}

/// General configuration
#[derive(Debug, Clone, Deserialize, Serialize, Default)]
pub struct GeneralConfig {
    // Fields would be defined here
    #[serde(default = "default_db_location")]
    pub db_location: String,
}

fn default_db_location() -> String {
    "/etc/hybridpipe".to_string()
}

/// Initialize the configuration system.
pub fn init_config() -> Result<(), Error> {
    if CONFIG.get().is_some() {
        return Ok(());
    }

    let config_path =
        env::var("HYBRIDPIPE_CONFIG").unwrap_or_else(|_| "hybridpipe_db.toml".to_string());

    let config = load_config(&config_path)
        .map_err(|e| Error::InvalidConfiguration(format!("Failed to load config: {}", e)))?;

    CONFIG
        .set(config)
        .map_err(|_| Error::InternalError("Failed to set global configuration".to_string()))?;

    Ok(())
}

/// Load the configuration from a file.
fn load_config(config_path: &str) -> Result<HybridPipeConfig, ConfigError> {
    let mut builder = Config::builder();

    // Add default configuration
    builder = builder.add_source(File::from_str(
        include_str!("../../default_config.toml"),
        config::FileFormat::Toml,
    ));

    // Add configuration from file if it exists
    if Path::new(config_path).exists() {
        builder = builder.add_source(File::with_name(config_path));
    }

    // Add configuration from environment variables
    builder = builder.add_source(Environment::with_prefix("HYBRIDPIPE").separator("_"));

    // Build the configuration
    let config = builder.build()?;

    // Deserialize the configuration
    config.try_deserialize()
}

/// Get the global configuration instance.
pub fn get_config() -> Result<&'static HybridPipeConfig, Error> {
    if let Some(config) = CONFIG.get() {
        Ok(config)
    } else {
        init_config()?;
        CONFIG
            .get()
            .ok_or_else(|| Error::InternalError("Failed to get global configuration".to_string()))
    }
}

/// Get the configuration for a specific protocol.
pub fn get_protocol_config(broker_type: super::BrokerType) -> Result<&'static dyn Any, Error> {
    let config = get_config()?;

    match broker_type {
        super::BrokerType::NATS => Ok(&config.nats as &dyn Any),
        super::BrokerType::KAFKA => Ok(&config.kafka as &dyn Any),
        super::BrokerType::RABBITMQ => Ok(&config.rabbitmq as &dyn Any),
        super::BrokerType::AMQP1 => Ok(&config.amqp1 as &dyn Any),
        super::BrokerType::MQTT => Ok(&config.mqtt as &dyn Any),
        super::BrokerType::QPID => Ok(&config.qpid as &dyn Any),
        super::BrokerType::NSQ => Ok(&config.nsq as &dyn Any),
        super::BrokerType::TCP => Ok(&config.tcp as &dyn Any),
        super::BrokerType::REDIS => Ok(&config.redis as &dyn Any),
        super::BrokerType::NETCHAN => Ok(&config.netchan as &dyn Any),
        _ => Err(Error::BrokerTypeNotRegistered(broker_type.to_string())),
    }
}
