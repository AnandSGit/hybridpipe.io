// HybridPipe - A unified messaging interface for microservices and distributed systems
//
// This is the main entry point for the HybridPipe library.

pub mod core;
pub mod middleware;
pub mod monitoring;
pub mod protocols;
pub mod serialization;
pub mod tracing;
pub mod utils;

// Re-export the most commonly used types and functions
pub use core::ext;
pub use core::{
    deploy_router, register_router_factory, BrokerType, Error, HybridPipe, Process, Result,
};

// Re-export serialization functionality
pub use serialization::{
    decode, decode_with_options, encode, encode_with_options, register_type, SerializationFormat,
    SerializationOptions,
};

// Protocol Buffers serialization
pub use serialization::protobuf::{
    create_message, decode_protobuf, encode_protobuf, register_protobuf_type,
};

// Re-export middleware functionality
pub use middleware::{LoggingMiddleware, Middleware, MiddlewareStack, MonitoringMiddleware};

// Re-export monitoring functionality
pub use monitoring::{new_message_stats, MessageStats, PipeMetrics};

// Re-export tracing functionality
pub use tracing::{Span, Trace, Tracer, TracingMiddleware};

// Constants for broker types
pub use core::broker_types::{
    AMQP1, KAFKA, MOCK, MQTT, NATS, NETCHAN, NSQ, QPID, RABBITMQ, REDIS, TCP,
};

// Register all protocol implementations
pub fn init() {
    protocols::register_all();
}
