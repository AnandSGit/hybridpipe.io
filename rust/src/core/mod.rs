// Core module for HybridPipe
//
// This module defines the core interfaces and functionality for the HybridPipe messaging system.

use async_trait::async_trait;
use std::any::Any;
use std::sync::Arc;
use thiserror::Error;

pub mod broker_types;
pub mod config;
pub mod context;
pub mod errors;
pub mod registry;

// Re-export the broker types
pub use broker_types::BrokerType;
pub use errors::{Error, Result};
pub use registry::{deploy_router, register_router_factory};

/// Process defines the callback function that should be called when a client receives a message.
pub type Process = Box<dyn Fn(Vec<u8>) -> Result<()> + Send + Sync>;

/// Extension methods for HybridPipe that are not object-safe.
/// These are implemented as standalone functions.
pub mod ext {
    use super::*;
    use crate::serialization::decode;

    /// Accept subscribes to messages from the specified pipe and processes them with the provided function.
    /// This is an alias for Subscribe for backward compatibility.
    pub async fn accept<T, F>(pipe: &dyn HybridPipe, pipe_name: &str, callback: F) -> Result<()>
    where
        T: 'static + Send + Sync + serde::de::DeserializeOwned,
        F: Fn(Box<dyn Any + Send + Sync>) + Send + Sync + 'static,
    {
        // Create a Process wrapper for the callback
        let process: Process = Box::new(move |data| {
            let decoded: Box<dyn Any + Send + Sync> = match decode::<T>(&data) {
                Ok(decoded) => Box::new(decoded),
                Err(e) => return Err(e),
            };
            callback(decoded);
            Ok(())
        });

        pipe.subscribe(pipe_name, process).await
    }
}

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
        data: Box<dyn Any + Send + Sync>,
    ) -> Result<()>;

    /// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
    async fn subscribe(&self, pipe: &str, callback: Process) -> Result<()>;

    /// SubscribeWithContext registers a callback with context for cancellation.
    async fn subscribe_with_context(
        &self,
        ctx: &context::Context,
        pipe: &str,
        callback: Process,
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

// Tests for the core module
#[cfg(test)]
mod tests;
