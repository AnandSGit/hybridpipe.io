// AMQP protocol implementation for HybridPipe
//
// This module provides a placeholder implementation of the HybridPipe interface for AMQP 1.0.
// The actual implementation is temporarily disabled.

use async_trait::async_trait;
use log::warn;
use std::any::Any;
use std::sync::{Arc, Mutex};

use crate::core::registry::register_router_factory;
use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};

/// Packet implements a placeholder for the HybridPipe interface for AMQP 1.0.
pub struct Packet {
    /// Connected flag indicates if the packet is connected
    connected: Mutex<bool>,
}

impl Packet {
    /// Create a new AMQP packet.
    pub fn new() -> Self {
        Self {
            connected: Mutex::new(false),
        }
    }
}

#[async_trait]
impl HybridPipe for Packet {
    async fn connect(&self) -> Result<()> {
        warn!("AMQP implementation is temporarily disabled. Using placeholder implementation.");
        let mut connected = self.connected.lock().unwrap();
        if *connected {
            return Err(Error::AlreadyConnected);
        }
        *connected = true;
        Ok(())
    }

    async fn disconnect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }
        *connected = false;
        Ok(())
    }

    async fn dispatch(&self, _pipe: &str, _data: Box<dyn Any + Send + Sync>) -> Result<()> {
        Err(Error::Other(
            "AMQP implementation is temporarily disabled".to_string(),
        ))
    }

    async fn dispatch_with_context(
        &self,
        _ctx: &context::Context,
        _pipe: &str,
        _data: Box<dyn Any + Send + Sync>,
    ) -> Result<()> {
        Err(Error::Other(
            "AMQP implementation is temporarily disabled".to_string(),
        ))
    }

    async fn subscribe(&self, _pipe: &str, _callback: Process) -> Result<()> {
        Err(Error::Other(
            "AMQP implementation is temporarily disabled".to_string(),
        ))
    }

    async fn subscribe_with_context(
        &self,
        _ctx: &context::Context,
        _pipe: &str,
        _callback: Process,
    ) -> Result<()> {
        Err(Error::Other(
            "AMQP implementation is temporarily disabled".to_string(),
        ))
    }

    async fn remove(&self, _pipe: &str) -> Result<()> {
        Err(Error::Other(
            "AMQP implementation is temporarily disabled".to_string(),
        ))
    }

    async fn unsubscribe(&self, _pipe: &str) -> Result<()> {
        Err(Error::Other(
            "AMQP implementation is temporarily disabled".to_string(),
        ))
    }

    async fn close(&self) -> Result<()> {
        self.disconnect().await
    }

    fn is_connected(&self) -> bool {
        let connected = self.connected.lock().unwrap();
        *connected
    }
}

/// Register the AMQP protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::AMQP1, Box::new(|| Arc::new(Packet::new())));
}
