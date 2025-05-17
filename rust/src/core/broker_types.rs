// Broker types for HybridPipe
//
// This module defines the broker types supported by HybridPipe.

use std::fmt;

/// BrokerType represents the type of messaging system to use.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum BrokerType {
    /// NATS messaging system
    NATS,
    /// Apache Kafka
    KAFKA,
    /// RabbitMQ (AMQP 0.9.1)
    RABBITMQ,
    /// ZeroMQ messaging system
    ZEROMQ,
    /// AMQP 1.0
    AMQP1,
    /// MQTT protocol
    MQTT,
    /// Apache Qpid
    QPID,
    /// NSQ messaging platform
    NSQ,
    /// Direct TCP/IP communication
    TCP,
    /// Redis pub/sub
    REDIS,
    /// Go channels over network
    NETCHAN,
    /// In-memory mock for testing
    MOCK,
}

impl fmt::Display for BrokerType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            BrokerType::NATS => write!(f, "NATS"),
            BrokerType::KAFKA => write!(f, "KAFKA"),
            BrokerType::RABBITMQ => write!(f, "RABBITMQ"),
            BrokerType::ZEROMQ => write!(f, "ZEROMQ"),
            BrokerType::AMQP1 => write!(f, "AMQP1"),
            BrokerType::MQTT => write!(f, "MQTT"),
            BrokerType::QPID => write!(f, "QPID"),
            BrokerType::NSQ => write!(f, "NSQ"),
            BrokerType::TCP => write!(f, "TCP"),
            BrokerType::REDIS => write!(f, "REDIS"),
            BrokerType::NETCHAN => write!(f, "NETCHAN"),
            BrokerType::MOCK => write!(f, "MOCK"),
        }
    }
}

// Constants for broker types
pub const NATS: BrokerType = BrokerType::NATS;
pub const KAFKA: BrokerType = BrokerType::KAFKA;
pub const RABBITMQ: BrokerType = BrokerType::RABBITMQ;
pub const ZEROMQ: BrokerType = BrokerType::ZEROMQ;
pub const AMQP1: BrokerType = BrokerType::AMQP1;
pub const MQTT: BrokerType = BrokerType::MQTT;
pub const QPID: BrokerType = BrokerType::QPID;
pub const NSQ: BrokerType = BrokerType::NSQ;
pub const TCP: BrokerType = BrokerType::TCP;
pub const REDIS: BrokerType = BrokerType::REDIS;
pub const NETCHAN: BrokerType = BrokerType::NETCHAN;
pub const MOCK: BrokerType = BrokerType::MOCK;
