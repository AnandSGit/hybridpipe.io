// Mock implementation of RabbitMQ client for testing
//
// This module provides a mock implementation of the RabbitMQ client for testing.

use std::sync::Arc;
use std::collections::HashMap;
use std::sync::Mutex;
use futures::Stream;
use std::pin::Pin;
use std::task::{Context, Poll};
use std::collections::VecDeque;

use super::common::{MessageStore, CallbackRegistry, ConnectionState, MockMessage};

/// Mock RabbitMQ connection
#[derive(Debug, Clone)]
pub struct MockRabbitMQConnection {
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
}

impl MockRabbitMQConnection {
    /// Create a new mock RabbitMQ connection
    pub fn new() -> Self {
        Self {
            connection_state: Arc::new(ConnectionState::new()),
        }
    }
    
    /// Connect to the mock RabbitMQ broker
    pub async fn connect(&self) -> Result<(), String> {
        self.connection_state.set_connected(true);
        Ok(())
    }
    
    /// Disconnect from the mock RabbitMQ broker
    pub async fn disconnect(&self) -> Result<(), String> {
        self.connection_state.set_connected(false);
        Ok(())
    }
    
    /// Create a channel
    pub async fn create_channel(&self) -> Result<MockRabbitMQChannel, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        Ok(MockRabbitMQChannel {
            message_store: Arc::new(MessageStore::new()),
            callback_registry: Arc::new(CallbackRegistry::new()),
            connection_state: self.connection_state.clone(),
            subscriptions: Arc::new(Mutex::new(HashMap::new())),
        })
    }
    
    /// Check if the connection is connected
    pub fn is_connected(&self) -> bool {
        self.connection_state.is_connected()
    }
}

/// Mock RabbitMQ channel
#[derive(Debug, Clone)]
pub struct MockRabbitMQChannel {
    /// Message store for sent messages
    pub message_store: Arc<MessageStore>,
    /// Callback registry for subscribers
    pub callback_registry: Arc<CallbackRegistry>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
    /// Subscriptions
    pub subscriptions: Arc<Mutex<HashMap<String, MockRabbitMQConsumer>>>,
}

impl MockRabbitMQChannel {
    /// Declare a queue
    pub async fn queue_declare(
        &self,
        queue: &str,
        _durable: bool,
        _exclusive: bool,
        _auto_delete: bool,
        _arguments: HashMap<String, String>,
    ) -> Result<MockRabbitMQQueue, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        Ok(MockRabbitMQQueue {
            name: queue.to_string(),
        })
    }
    
    /// Publish a message to an exchange
    pub async fn basic_publish(
        &self,
        exchange: &str,
        routing_key: &str,
        _mandatory: bool,
        _immediate: bool,
        payload: Vec<u8>,
    ) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        // Store the message
        let topic = if exchange.is_empty() {
            routing_key.to_string()
        } else {
            format!("{}.{}", exchange, routing_key)
        };
        
        self.message_store.add_message(&topic, payload.clone());
        
        // Call callbacks for the topic
        self.callback_registry.call_callbacks(&topic, payload);
        
        Ok(())
    }
    
    /// Consume messages from a queue
    pub async fn basic_consume(
        &self,
        queue: &str,
        _consumer_tag: &str,
        _no_local: bool,
        _no_ack: bool,
        _exclusive: bool,
        _arguments: HashMap<String, String>,
    ) -> Result<MockRabbitMQConsumer, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        let consumer = MockRabbitMQConsumer {
            queue: queue.to_string(),
            messages: Arc::new(Mutex::new(VecDeque::new())),
            connection_state: self.connection_state.clone(),
        };
        
        // Store the subscription
        let mut subscriptions = self.subscriptions.lock().unwrap();
        subscriptions.insert(queue.to_string(), consumer.clone());
        
        Ok(consumer)
    }
    
    /// Cancel a consumer
    pub async fn basic_cancel(&self, queue: &str) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }
        
        // Remove the subscription
        let mut subscriptions = self.subscriptions.lock().unwrap();
        subscriptions.remove(queue);
        
        // Remove callbacks
        self.callback_registry.remove_callbacks(queue);
        
        Ok(())
    }
}

/// Mock RabbitMQ queue
#[derive(Debug, Clone)]
pub struct MockRabbitMQQueue {
    /// Queue name
    pub name: String,
}

/// Mock RabbitMQ consumer
#[derive(Debug, Clone)]
pub struct MockRabbitMQConsumer {
    /// Queue this consumer is consuming from
    pub queue: String,
    /// Messages queue
    pub messages: Arc<Mutex<VecDeque<MockMessage>>>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
}

impl Stream for MockRabbitMQConsumer {
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

/// Connect to a RabbitMQ server
pub async fn connect(url: &str) -> Result<MockRabbitMQConnection, String> {
    let connection = MockRabbitMQConnection::new();
    connection.connect().await?;
    Ok(connection)
}
