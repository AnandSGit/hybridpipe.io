// HybridPipe - A unified messaging interface for microservices and distributed systems
//
// This is the main entry point for the HybridPipe library.

pub mod core;
pub mod protocols;
pub mod serialization;
pub mod utils;
pub mod middleware;
pub mod monitoring;
pub mod tracing;

// Re-export the most commonly used types and functions
pub use core::{
    HybridPipe, 
    BrokerType, 
    Process, 
    Error, 
    Result,
    deploy_router,
    register_router_factory,
};

pub use serialization::{
    SerializationFormat,
    SerializationOptions,
    encode,
    decode,
    encode_with_options,
    decode_with_options,
    register_type,
};

// Constants for broker types
pub use core::broker_types::{
    NATS,
    KAFKA,
    RABBITMQ,
    AMQP1,
    MQTT,
    QPID,
    NSQ,
    TCP,
    REDIS,
    NETCHAN,
    MOCK,
};
