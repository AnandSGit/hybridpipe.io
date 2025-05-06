// Error types for HybridPipe
//
// This module defines the error types used by HybridPipe.

use thiserror::Error;

/// Result type for HybridPipe operations
pub type Result<T> = std::result::Result<T, Error>;

/// Error represents all possible errors returned by HybridPipe.
#[derive(Error, Debug)]
pub enum Error {
    /// NotConnected is returned when an operation is attempted on a disconnected HybridPipe.
    #[error("not connected to messaging system")]
    NotConnected,

    /// PipeNotFound is returned when an operation is attempted on a non-existent pipe.
    #[error("pipe not found: {0}")]
    PipeNotFound(String),

    /// AlreadyConnected is returned when Connect is called on an already connected HybridPipe.
    #[error("already connected to messaging system")]
    AlreadyConnected,

    /// AlreadySubscribed is returned when Subscribe is called on an already subscribed pipe.
    #[error("already subscribed to pipe: {0}")]
    AlreadySubscribed(String),

    /// NotSubscribed is returned when Unsubscribe is called on a non-subscribed pipe.
    #[error("not subscribed to pipe: {0}")]
    NotSubscribed(String),

    /// InvalidConfiguration is returned when the configuration is invalid.
    #[error("invalid configuration: {0}")]
    InvalidConfiguration(String),

    /// SerializationError is returned when serialization fails.
    #[error("serialization error: {0}")]
    SerializationError(String),

    /// DeserializationError is returned when deserialization fails.
    #[error("deserialization error: {0}")]
    DeserializationError(String),

    /// ConnectionError is returned when a connection error occurs.
    #[error("connection error: {0}")]
    ConnectionError(String),

    /// DispatchError is returned when a dispatch error occurs.
    #[error("dispatch error: {0}")]
    DispatchError(String),

    /// SubscriptionError is returned when a subscription error occurs.
    #[error("subscription error: {0}")]
    SubscriptionError(String),

    /// UnsubscriptionError is returned when an unsubscription error occurs.
    #[error("unsubscription error: {0}")]
    UnsubscriptionError(String),

    /// TimeoutError is returned when an operation times out.
    #[error("timeout error: {0}")]
    TimeoutError(String),

    /// BrokerTypeNotRegistered is returned when a broker type is not registered.
    #[error("broker type not registered: {0}")]
    BrokerTypeNotRegistered(String),

    /// InternalError is returned when an internal error occurs.
    #[error("internal error: {0}")]
    InternalError(String),

    /// IOError is returned when an I/O error occurs.
    #[error("I/O error: {0}")]
    IOError(#[from] std::io::Error),

    /// Other is returned when an error doesn't fit into any of the above categories.
    #[error("other error: {0}")]
    Other(String),
}
