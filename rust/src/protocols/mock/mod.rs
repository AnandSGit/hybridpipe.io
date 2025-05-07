// Mock protocol implementation for HybridPipe
//
// This module provides an in-memory implementation of the HybridPipe interface for testing.

#[cfg(test)]
mod tests;

use async_trait::async_trait;
use log::{debug, info};
use std::any::Any;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tokio::sync::RwLock;

use crate::core::registry::register_router_factory;
use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};

/// Packet implements the HybridPipe interface for in-memory messaging.
pub struct Packet {
    /// Subscribers maps pipe names to their callback functions
    subscribers: RwLock<HashMap<String, Process>>,
    /// Messages maps pipe names to their messages
    messages: RwLock<HashMap<String, Vec<Vec<u8>>>>,
    /// Connected flag indicates if the packet is connected
    connected: Mutex<bool>,
}

impl Packet {
    /// Create a new mock packet.
    pub fn new() -> Self {
        Self {
            subscribers: RwLock::new(HashMap::new()),
            messages: RwLock::new(HashMap::new()),
            connected: Mutex::new(false),
        }
    }
}

#[async_trait]
impl HybridPipe for Packet {
    async fn connect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if *connected {
            return Err(Error::AlreadyConnected);
        }

        *connected = true;
        info!("Connected to mock messaging system");
        Ok(())
    }

    async fn disconnect(&self) -> Result<()> {
        // Check if connected and set flag to false
        {
            let mut connected = self.connected.lock().unwrap();
            if !*connected {
                return Err(Error::NotConnected);
            }
            *connected = false;
        }

        // Clear subscribers and messages
        let mut subscribers = self.subscribers.write().await;
        subscribers.clear();

        let mut messages = self.messages.write().await;
        messages.clear();

        info!("Disconnected from mock messaging system");
        Ok(())
    }

    async fn dispatch(&self, pipe: &str, data: Box<dyn Any + Send + Sync>) -> Result<()> {
        // Check if connected
        {
            let connected = self.connected.lock().unwrap();
            if !*connected {
                return Err(Error::NotConnected);
            }
        }

        // Extract the bytes from the data
        let bytes = if let Some(bytes) = data.downcast_ref::<Vec<u8>>() {
            bytes.clone()
        } else {
            // If it's not already bytes, try to serialize it
            serde_json::to_vec(&format!("{:?}", data)).map_err(|e| {
                Error::SerializationError(format!("Failed to encode message: {}", e))
            })?
        };

        // Clone the bytes for storage
        let bytes_clone = bytes.clone();

        // Store the message
        let mut messages = self.messages.write().await;
        messages
            .entry(pipe.to_string())
            .or_insert_with(Vec::new)
            .push(bytes_clone);
        drop(messages);

        // Deliver the message to subscribers
        let subscribers = self.subscribers.read().await;
        if let Some(callback) = subscribers.get(pipe) {
            if let Err(e) = callback(bytes.to_vec()) {
                debug!("Error processing message from pipe {}: {}", pipe, e);
            }
        }

        Ok(())
    }

    async fn dispatch_with_context(
        &self,
        ctx: &context::Context,
        pipe: &str,
        data: Box<dyn Any + Send + Sync>,
    ) -> Result<()> {
        // Use the context to run the dispatch with a timeout
        ctx.run(self.dispatch(pipe, data)).await
    }

    async fn subscribe(&self, pipe: &str, callback: Process) -> Result<()> {
        // Check if connected
        {
            let connected = self.connected.lock().unwrap();
            if !*connected {
                return Err(Error::NotConnected);
            }
        }

        // Check if already subscribed
        let subscribers = self.subscribers.read().await;
        if subscribers.contains_key(pipe) {
            return Err(Error::AlreadySubscribed(pipe.to_string()));
        }
        drop(subscribers);

        // Store the callback
        let mut subscribers = self.subscribers.write().await;
        subscribers.insert(pipe.to_string(), callback);

        // Deliver any existing messages
        let messages = self.messages.read().await;
        if let Some(pipe_messages) = messages.get(pipe) {
            let callback = subscribers.get(pipe).unwrap();
            for message in pipe_messages {
                if let Err(e) = callback(message.clone()) {
                    debug!("Error processing message from pipe {}: {}", pipe, e);
                }
            }
        }

        info!("Subscribed to mock pipe: {}", pipe);
        Ok(())
    }

    async fn subscribe_with_context(
        &self,
        ctx: &context::Context,
        pipe: &str,
        callback: Process,
    ) -> Result<()> {
        // Use the context to run the subscribe with a timeout
        ctx.run(self.subscribe(pipe, callback)).await
    }

    async fn remove(&self, pipe: &str) -> Result<()> {
        self.unsubscribe(pipe).await
    }

    async fn unsubscribe(&self, pipe: &str) -> Result<()> {
        // Check if connected
        {
            let connected = self.connected.lock().unwrap();
            if !*connected {
                return Err(Error::NotConnected);
            }
        }

        // Check if subscribed
        let subscribers = self.subscribers.read().await;
        if !subscribers.contains_key(pipe) {
            return Err(Error::NotSubscribed(pipe.to_string()));
        }
        drop(subscribers);

        // Remove the subscriber
        let mut subscribers = self.subscribers.write().await;
        subscribers.remove(pipe);

        info!("Unsubscribed from mock pipe: {}", pipe);
        Ok(())
    }

    async fn close(&self) -> Result<()> {
        self.disconnect().await
    }

    fn is_connected(&self) -> bool {
        let connected = self.connected.lock().unwrap();
        *connected
    }
}

/// Register the mock protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::MOCK, Box::new(|| Arc::new(Packet::new())));
}
