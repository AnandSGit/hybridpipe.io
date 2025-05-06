// Core module for HybridPipe
//
// This module defines the core interfaces and functionality for the HybridPipe messaging system.

use std::fmt;
use std::sync::Arc;
use async_trait::async_trait;
use thiserror::Error;
use tokio::sync::RwLock;
use std::collections::HashMap;
use std::any::Any;

pub mod broker_types;
pub mod config;
pub mod registry;
pub mod context;
pub mod errors;

// Re-export the broker types
pub use broker_types::BrokerType;
pub use errors::{Error, Result};
pub use registry::{deploy_router, register_router_factory};

/// Process defines the callback function that should be called when a client receives a message.
pub type Process = Box<dyn Fn(Vec<u8>) -> Result<()> + Send + Sync>;

/// HybridPipe defines the core interface for all messaging implementations.
/// Any messaging system implementation must satisfy this interface.
#[async_trait]
pub trait HybridPipe: Send + Sync {
    /// Connect establishes a connection to the messaging system.
    async fn connect(&self) -> Result<()>;

    /// Disconnect closes the connection to the messaging system.
    async fn disconnect(&self) -> Result<()>;

    /// Dispatch sends a message to the specified pipe.
    async fn dispatch(&self, pipe: &str, data: Box<dyn Any + Send + Sync>) -> Result<()>;

    /// DispatchWithContext sends a message with context for cancellation and timeouts.
    async fn dispatch_with_context(
        &self, 
        ctx: &context::Context, 
        pipe: &str, 
        data: Box<dyn Any + Send + Sync>
    ) -> Result<()>;

    /// Accept subscribes to messages from the specified pipe and processes them with the provided function.
    /// This is an alias for Subscribe for backward compatibility.
    async fn accept<F>(&self, pipe: &str, callback: F) -> Result<()>
    where
        F: Fn(Box<dyn Any + Send + Sync>) + Send + Sync + 'static;

    /// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
    async fn subscribe(&self, pipe: &str, callback: Process) -> Result<()>;

    /// SubscribeWithContext registers a callback with context for cancellation.
    async fn subscribe_with_context(
        &self, 
        ctx: &context::Context, 
        pipe: &str, 
        callback: Process
    ) -> Result<()>;

    /// Remove unsubscribes from the specified pipe.
    /// This is an alias for Unsubscribe for backward compatibility.
    async fn remove(&self, pipe: &str) -> Result<()>;

    /// Unsubscribe removes a subscription from the specified pipe.
    async fn unsubscribe(&self, pipe: &str) -> Result<()>;

    /// Close terminates the connection to the messaging system.
    /// This is an alias for Disconnect for backward compatibility.
    async fn close(&self) -> Result<()>;

    /// IsConnected returns true if the connection to the messaging system is active.
    fn is_connected(&self) -> bool;
}

/// RouterFactory is a function that creates a new instance of a HybridPipe implementation.
pub type RouterFactory = Box<dyn Fn() -> Arc<dyn HybridPipe> + Send + Sync>;

// Type alias for a boxed HybridPipe trait object
pub type BoxedHybridPipe = Arc<dyn HybridPipe>;
