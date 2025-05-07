// Mock implementation of NATS client for testing
//
// This module provides a mock implementation of the NATS client for testing.

// No need for async_trait in this mock
use futures::Stream;
use std::collections::VecDeque;
use std::pin::Pin;
use std::sync::Arc;
use std::sync::Mutex;
use std::task::{Context, Poll};
use std::time::Duration;

use super::common::{CallbackRegistry, ConnectionState, MessageStore, MockMessage};

/// Mock NATS client
#[derive(Debug)]
pub struct MockNatsClient {
    /// Message store for sent messages
    pub message_store: Arc<MessageStore>,
    /// Callback registry for subscribers
    pub callback_registry: Arc<CallbackRegistry>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
}

impl MockNatsClient {
    /// Create a new mock NATS client
    pub fn new() -> Self {
        Self {
            message_store: Arc::new(MessageStore::new()),
            callback_registry: Arc::new(CallbackRegistry::new()),
            connection_state: Arc::new(ConnectionState::new()),
        }
    }

    /// Connect to the mock NATS server
    pub async fn connect(&self) -> Result<(), String> {
        self.connection_state.set_connected(true);
        Ok(())
    }

    /// Disconnect from the mock NATS server
    pub async fn disconnect(&self) -> Result<(), String> {
        self.connection_state.set_connected(false);
        self.callback_registry.clear();
        self.message_store.clear();
        Ok(())
    }

    /// Publish a message to a subject
    pub async fn publish(&self, subject: String, payload: Vec<u8>) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        // Store the message
        self.message_store.add_message(&subject, payload.clone());

        // Call callbacks for the subject
        self.callback_registry.call_callbacks(&subject, payload);

        Ok(())
    }

    /// Subscribe to a subject
    pub async fn subscribe(&self, subject: String) -> Result<MockSubscriber, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        Ok(MockSubscriber {
            subject: subject.clone(),
            message_store: self.message_store.clone(),
            connection_state: self.connection_state.clone(),
            messages: Mutex::new(VecDeque::new()),
        })
    }

    /// Check if the client is connected
    pub fn is_connected(&self) -> bool {
        self.connection_state.is_connected()
    }
}

impl Clone for MockNatsClient {
    fn clone(&self) -> Self {
        Self {
            message_store: self.message_store.clone(),
            callback_registry: self.callback_registry.clone(),
            connection_state: self.connection_state.clone(),
        }
    }
}

/// Mock NATS subscriber
pub struct MockSubscriber {
    /// The subject this subscriber is subscribed to
    pub subject: String,
    /// Message store for retrieving messages
    pub message_store: Arc<MessageStore>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
    /// Queue of messages to return
    pub messages: Mutex<VecDeque<MockMessage>>,
}

impl MockSubscriber {
    /// Add a message to this subscriber's queue
    pub fn add_message(&self, payload: Vec<u8>) {
        let mut messages = self.messages.lock().unwrap();
        messages.push_back(MockMessage {
            subject: self.subject.clone(),
            payload,
        });
    }
}

impl Stream for MockSubscriber {
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

// We use the common MockMessage type

/// Mock NATS connection options
#[derive(Debug, Clone)]
pub struct MockConnectOptions {
    /// Connection timeout
    pub connection_timeout: Duration,
    /// Whether to retry on initial connect
    pub retry_on_initial_connect: bool,
    /// Whether to require TLS
    pub require_tls: bool,
}

impl MockConnectOptions {
    /// Create new connection options
    pub fn new() -> Self {
        Self {
            connection_timeout: Duration::from_secs(5),
            retry_on_initial_connect: false,
            require_tls: false,
        }
    }

    /// Set connection timeout
    pub fn connection_timeout(mut self, timeout: Duration) -> Self {
        self.connection_timeout = timeout;
        self
    }

    /// Set retry on initial connect
    pub fn retry_on_initial_connect(mut self) -> Self {
        self.retry_on_initial_connect = true;
        self
    }

    /// Set reconnect delay callback
    pub fn reconnect_delay_callback<F>(self, _callback: F) -> Self
    where
        F: Fn(usize) -> Duration + Send + Sync + 'static,
    {
        // We don't actually use the callback in the mock
        self
    }

    /// Require TLS
    pub fn require_tls(mut self, require: bool) -> Self {
        self.require_tls = require;
        self
    }
}

/// Connect to a mock NATS server
pub async fn connect(_server: &str) -> Result<MockNatsClient, String> {
    let client = MockNatsClient::new();
    client.connect().await?;
    Ok(client)
}

/// Connect to a mock NATS server with options
pub async fn connect_with_options(
    _server: &str,
    _options: MockConnectOptions,
) -> Result<MockNatsClient, String> {
    let client = MockNatsClient::new();
    client.connect().await?;
    Ok(client)
}
