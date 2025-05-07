// Mock implementation of Redis client for testing
//
// This module provides a mock implementation of the Redis client for testing.

use std::sync::Arc;
use std::collections::HashMap;
use std::sync::Mutex;
use futures::Stream;
use std::pin::Pin;
use std::task::{Context, Poll};
use std::collections::VecDeque;

use super::common::{MessageStore, CallbackRegistry, ConnectionState, MockMessage};

/// Mock Redis client
#[derive(Debug, Clone)]
pub struct MockRedisClient {
    /// Message store for sent messages
    pub message_store: Arc<MessageStore>,
    /// Callback registry for subscribers
    pub callback_registry: Arc<CallbackRegistry>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
    /// Subscriptions
    pub subscriptions: Arc<Mutex<HashMap<String, MockRedisSubscriber>>>,
    /// Key-value store
    pub kv_store: Arc<Mutex<HashMap<String, Vec<u8>>>>,
}

impl MockRedisClient {
    /// Create a new mock Redis client
    pub fn new() -> Self {
        Self {
            message_store: Arc::new(MessageStore::new()),
            callback_registry: Arc::new(CallbackRegistry::new()),
            connection_state: Arc::new(ConnectionState::new()),
            subscriptions: Arc::new(Mutex::new(HashMap::new())),
            kv_store: Arc::new(Mutex::new(HashMap::new())),
        }
    }
    
    /// Connect to the mock Redis server
    pub async fn connect(&self) -> Result<(), String> {
        self.connection_state.set_connected(true);
        Ok(())
    }
    
    /// Disconnect from the mock Redis server
    pub async fn disconnect(&self) -> Result<(), String> {
        self.connection_state.set_connected(false);
        self.callback_registry.clear();
        self.message_store.clear();
        Ok(())
    }
    
    /// Publish a message to a channel
    pub async fn publish(&self, channel: &str, payload: Vec<u8>) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        // Store the message
        self.message_store.add_message(channel, payload.clone());
        
        // Call callbacks for the channel
        self.callback_registry.call_callbacks(channel, payload);
        
        Ok(())
    }
    
    /// Subscribe to a channel
    pub async fn subscribe(&self, channel: &str) -> Result<MockRedisSubscriber, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        let subscriber = MockRedisSubscriber {
            channel: channel.to_string(),
            messages: Arc::new(Mutex::new(VecDeque::new())),
            connection_state: self.connection_state.clone(),
        };
        
        // Store the subscription
        let mut subscriptions = self.subscriptions.lock().unwrap();
        subscriptions.insert(channel.to_string(), subscriber.clone());
        
        Ok(subscriber)
    }
    
    /// Unsubscribe from a channel
    pub async fn unsubscribe(&self, channel: &str) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        // Remove the subscription
        let mut subscriptions = self.subscriptions.lock().unwrap();
        subscriptions.remove(channel);
        
        // Remove callbacks
        self.callback_registry.remove_callbacks(channel);
        
        Ok(())
    }
    
    /// Set a key-value pair
    pub async fn set(&self, key: &str, value: Vec<u8>) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        let mut kv_store = self.kv_store.lock().unwrap();
        kv_store.insert(key.to_string(), value);
        
        Ok(())
    }
    
    /// Get a value by key
    pub async fn get(&self, key: &str) -> Result<Option<Vec<u8>>, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        let kv_store = self.kv_store.lock().unwrap();
        Ok(kv_store.get(key).cloned())
    }
    
    /// Delete a key
    pub async fn del(&self, key: &str) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        let mut kv_store = self.kv_store.lock().unwrap();
        kv_store.remove(key);
        
        Ok(())
    }
    
    /// Check if the client is connected
    pub fn is_connected(&self) -> bool {
        self.connection_state.is_connected()
    }
}

/// Mock Redis subscriber
#[derive(Debug, Clone)]
pub struct MockRedisSubscriber {
    /// Channel this subscriber is subscribed to
    pub channel: String,
    /// Messages queue
    pub messages: Arc<Mutex<VecDeque<MockMessage>>>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
}

impl Stream for MockRedisSubscriber {
    type Item = MockMessage;
    
    fn poll_next(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        if !self.connection_state.is_connected() {
            return Poll::Ready(None);
        }
        
        let mut messages = self.messages.lock().unwrap();
        if let Some(message) = messages.pop_front() {
            Poll::Ready(Some(message))
        } else {
            Poll::Pending
        }
    }
}

/// Connect to a Redis server
pub async fn connect(url: &str) -> Result<MockRedisClient, String> {
    let client = MockRedisClient::new();
    client.connect().await?;
    Ok(client)
}
