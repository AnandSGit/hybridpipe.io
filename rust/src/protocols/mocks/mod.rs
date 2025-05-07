// Mocking framework for protocol tests
//
// This module provides mock implementations of external messaging systems for testing.

pub mod amqp;
pub mod kafka;
pub mod mqtt;
pub mod nats;
pub mod rabbitmq;
pub mod redis;
pub mod tcp;

/// Common traits and utilities for mocks
pub mod common {
    use std::collections::HashMap;
    use std::sync::{Arc, Mutex};

    /// A generic message type for mocks
    #[derive(Debug, Clone)]
    pub struct MockMessage {
        /// The subject or topic the message was sent to
        pub subject: String,
        /// The message payload
        pub payload: Vec<u8>,
    }

    /// A shared message store for mocks
    #[derive(Debug, Default)]
    pub struct MessageStore {
        /// Messages stored by subject/topic
        messages: Mutex<HashMap<String, Vec<MockMessage>>>,
    }

    impl MessageStore {
        /// Create a new message store
        pub fn new() -> Self {
            Self {
                messages: Mutex::new(HashMap::new()),
            }
        }

        /// Add a message to the store
        pub fn add_message(&self, subject: &str, payload: Vec<u8>) {
            let mut messages = self.messages.lock().unwrap();
            let subject_messages = messages.entry(subject.to_string()).or_insert_with(Vec::new);
            subject_messages.push(MockMessage {
                subject: subject.to_string(),
                payload,
            });
        }

        /// Get messages for a subject
        pub fn get_messages(&self, subject: &str) -> Vec<MockMessage> {
            let messages = self.messages.lock().unwrap();
            if let Some(subject_messages) = messages.get(subject) {
                subject_messages.clone()
            } else {
                Vec::new()
            }
        }

        /// Clear all messages
        pub fn clear(&self) {
            let mut messages = self.messages.lock().unwrap();
            messages.clear();
        }
    }

    /// A callback registry for mocks
    #[derive(Default)]
    pub struct CallbackRegistry {
        /// Callbacks stored by subject/topic
        callbacks:
            Mutex<HashMap<String, Vec<Arc<dyn Fn(Vec<u8>) -> Result<(), String> + Send + Sync>>>>,
    }

    impl std::fmt::Debug for CallbackRegistry {
        fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            f.debug_struct("CallbackRegistry")
                .field("callbacks", &"<function pointers>")
                .finish()
        }
    }

    impl CallbackRegistry {
        /// Create a new callback registry
        pub fn new() -> Self {
            Self {
                callbacks: Mutex::new(HashMap::new()),
            }
        }

        /// Register a callback for a subject
        pub fn register_callback<F>(&self, subject: &str, callback: F)
        where
            F: Fn(Vec<u8>) -> Result<(), String> + Send + Sync + 'static,
        {
            let mut callbacks = self.callbacks.lock().unwrap();
            let subject_callbacks = callbacks
                .entry(subject.to_string())
                .or_insert_with(Vec::new);
            subject_callbacks.push(Arc::new(callback));
        }

        /// Call all callbacks for a subject
        pub fn call_callbacks(&self, subject: &str, payload: Vec<u8>) -> Vec<Result<(), String>> {
            let callbacks = self.callbacks.lock().unwrap();
            if let Some(subject_callbacks) = callbacks.get(subject) {
                subject_callbacks
                    .iter()
                    .map(|callback| callback(payload.clone()))
                    .collect()
            } else {
                Vec::new()
            }
        }

        /// Remove all callbacks for a subject
        pub fn remove_callbacks(&self, subject: &str) {
            let mut callbacks = self.callbacks.lock().unwrap();
            callbacks.remove(subject);
        }

        /// Clear all callbacks
        pub fn clear(&self) {
            let mut callbacks = self.callbacks.lock().unwrap();
            callbacks.clear();
        }
    }

    /// A mock connection state
    #[derive(Debug, Default)]
    pub struct ConnectionState {
        /// Whether the connection is established
        pub connected: Mutex<bool>,
    }

    impl ConnectionState {
        /// Create a new connection state
        pub fn new() -> Self {
            Self {
                connected: Mutex::new(false),
            }
        }

        /// Set the connection state
        pub fn set_connected(&self, connected: bool) {
            let mut state = self.connected.lock().unwrap();
            *state = connected;
        }

        /// Get the connection state
        pub fn is_connected(&self) -> bool {
            let state = self.connected.lock().unwrap();
            *state
        }
    }
}
