// Mock implementation of Kafka client for testing
//
// This module provides a mock implementation of the Kafka client for testing.

use futures::Stream;
use std::collections::HashMap;
use std::collections::VecDeque;
use std::pin::Pin;
use std::sync::Arc;
use std::sync::Mutex;
use std::task::{Context, Poll};

use super::common::{CallbackRegistry, ConnectionState, MessageStore, MockMessage};

/// Mock Kafka client
#[derive(Debug, Clone)]
pub struct MockKafkaClient {
    /// Message store for sent messages
    pub message_store: Arc<MessageStore>,
    /// Callback registry for subscribers
    pub callback_registry: Arc<CallbackRegistry>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
    /// Subscriptions
    pub subscriptions: Arc<Mutex<HashMap<String, MockKafkaConsumer>>>,
}

impl MockKafkaClient {
    /// Create a new mock Kafka client
    pub fn new() -> Self {
        Self {
            message_store: Arc::new(MessageStore::new()),
            callback_registry: Arc::new(CallbackRegistry::new()),
            connection_state: Arc::new(ConnectionState::new()),
            subscriptions: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    /// Connect to the mock Kafka broker
    pub async fn connect(&self) -> Result<(), String> {
        self.connection_state.set_connected(true);
        Ok(())
    }

    /// Disconnect from the mock Kafka broker
    pub async fn disconnect(&self) -> Result<(), String> {
        self.connection_state.set_connected(false);
        self.callback_registry.clear();
        self.message_store.clear();
        Ok(())
    }

    /// Publish a message to a topic
    pub async fn publish(&self, topic: &str, payload: Vec<u8>) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        // Store the message
        self.message_store.add_message(topic, payload.clone());

        // Call callbacks for the topic
        self.callback_registry.call_callbacks(topic, payload);

        Ok(())
    }

    /// Subscribe to a topic
    pub async fn subscribe(&self, topic: &str) -> Result<MockKafkaConsumer, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        let consumer = MockKafkaConsumer {
            topic: topic.to_string(),
            messages: Arc::new(Mutex::new(VecDeque::new())),
            connection_state: self.connection_state.clone(),
        };

        // Store the subscription
        let mut subscriptions = self.subscriptions.lock().unwrap();
        subscriptions.insert(topic.to_string(), consumer.clone());

        Ok(consumer)
    }

    /// Unsubscribe from a topic
    pub async fn unsubscribe(&self, topic: &str) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        // Remove the subscription
        let mut subscriptions = self.subscriptions.lock().unwrap();
        subscriptions.remove(topic);

        // Remove callbacks
        self.callback_registry.remove_callbacks(topic);

        Ok(())
    }

    /// Check if the client is connected
    pub fn is_connected(&self) -> bool {
        self.connection_state.is_connected()
    }
}

/// Mock Kafka consumer
#[derive(Debug, Clone)]
pub struct MockKafkaConsumer {
    /// Topic this consumer is subscribed to
    pub topic: String,
    /// Messages queue
    pub messages: Arc<Mutex<VecDeque<MockMessage>>>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
}

impl Stream for MockKafkaConsumer {
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

/// Mock Kafka producer config
#[derive(Debug, Clone)]
pub struct MockKafkaProducerConfig {
    /// Broker list
    pub brokers: Vec<String>,
    /// Client ID
    pub client_id: String,
    /// Acks
    pub acks: i16,
}

impl MockKafkaProducerConfig {
    /// Create new producer config
    pub fn new(brokers: Vec<String>) -> Self {
        Self {
            brokers,
            client_id: "mock-kafka-producer".to_string(),
            acks: 1,
        }
    }

    /// Set client ID
    pub fn set_client_id(mut self, client_id: &str) -> Self {
        self.client_id = client_id.to_string();
        self
    }

    /// Set acks
    pub fn set_acks(mut self, acks: i16) -> Self {
        self.acks = acks;
        self
    }
}

/// Mock Kafka consumer config
#[derive(Debug, Clone)]
pub struct MockKafkaConsumerConfig {
    /// Broker list
    pub brokers: Vec<String>,
    /// Group ID
    pub group_id: String,
    /// Client ID
    pub client_id: String,
}

impl MockKafkaConsumerConfig {
    /// Create new consumer config
    pub fn new(brokers: Vec<String>, group_id: &str) -> Self {
        Self {
            brokers,
            group_id: group_id.to_string(),
            client_id: "mock-kafka-consumer".to_string(),
        }
    }

    /// Set client ID
    pub fn set_client_id(mut self, client_id: &str) -> Self {
        self.client_id = client_id.to_string();
        self
    }
}

/// Create a new mock Kafka client
pub async fn create_client(brokers: Vec<String>) -> Result<MockKafkaClient, String> {
    let client = MockKafkaClient::new();
    client.connect().await?;
    Ok(client)
}
