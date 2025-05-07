// Mock implementation of MQTT client for testing
//
// This module provides a mock implementation of the MQTT client for testing.

// No need for Stream, Pin, Context, Poll in this mock
use std::collections::VecDeque;
use std::sync::Arc;
use std::sync::Mutex;
use std::time::Duration;

use super::common::{CallbackRegistry, ConnectionState, MessageStore};

/// Mock MQTT client
#[derive(Debug)]
pub struct MockMqttClient {
    /// Message store for sent messages
    pub message_store: Arc<MessageStore>,
    /// Callback registry for subscribers
    pub callback_registry: Arc<CallbackRegistry>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
}

impl MockMqttClient {
    /// Create a new mock MQTT client
    pub fn new() -> Self {
        Self {
            message_store: Arc::new(MessageStore::new()),
            callback_registry: Arc::new(CallbackRegistry::new()),
            connection_state: Arc::new(ConnectionState::new()),
        }
    }

    /// Connect to the mock MQTT broker
    pub async fn connect(&self) -> Result<(), String> {
        self.connection_state.set_connected(true);
        Ok(())
    }

    /// Disconnect from the mock MQTT broker
    pub async fn disconnect(&self) -> Result<(), String> {
        self.connection_state.set_connected(false);
        self.callback_registry.clear();
        self.message_store.clear();
        Ok(())
    }

    /// Publish a message to a topic
    pub async fn publish(
        &self,
        topic: &str,
        _qos: QoS,
        _retain: bool,
        payload: Vec<u8>,
    ) -> Result<(), String> {
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
    pub async fn subscribe(&self, _topic: &str, _qos: QoS) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        Ok(())
    }

    /// Unsubscribe from a topic
    pub async fn unsubscribe(&self, topic: &str) -> Result<(), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        self.callback_registry.remove_callbacks(topic);

        Ok(())
    }

    /// Check if the client is connected
    pub fn is_connected(&self) -> bool {
        self.connection_state.is_connected()
    }
}

impl Clone for MockMqttClient {
    fn clone(&self) -> Self {
        Self {
            message_store: self.message_store.clone(),
            callback_registry: self.callback_registry.clone(),
            connection_state: self.connection_state.clone(),
        }
    }
}

/// Mock MQTT event loop
pub struct MockEventLoop {
    /// Message store for retrieving messages
    pub message_store: Arc<MessageStore>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
    /// Queue of events to return
    pub events: Mutex<VecDeque<Event>>,
}

impl MockEventLoop {
    /// Create a new mock event loop
    pub fn new(message_store: Arc<MessageStore>, connection_state: Arc<ConnectionState>) -> Self {
        Self {
            message_store,
            connection_state,
            events: Mutex::new(VecDeque::new()),
        }
    }

    /// Add an event to the queue
    pub fn add_event(&self, event: Event) {
        let mut events = self.events.lock().unwrap();
        events.push_back(event);
    }

    /// Add a publish event for a topic
    pub fn add_publish_event(&self, topic: &str, payload: Vec<u8>) {
        let packet = Packet::Publish(Publish {
            topic: topic.to_string(),
            payload,
            qos: QoS::AtLeastOnce,
            retain: false,
        });

        self.add_event(Event::Incoming(packet));
    }

    /// Poll for the next event
    pub async fn poll(&mut self) -> Result<Event, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        let mut events = self.events.lock().unwrap();
        if let Some(event) = events.pop_front() {
            Ok(event)
        } else {
            // In a real implementation, this would wait for events
            // For testing, we'll just return an error
            Err("No events available".to_string())
        }
    }
}

/// Mock MQTT QoS
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum QoS {
    /// At most once delivery
    AtMostOnce,
    /// At least once delivery
    AtLeastOnce,
    /// Exactly once delivery
    ExactlyOnce,
}

/// Mock MQTT event
#[derive(Debug, Clone)]
pub enum Event {
    /// Incoming packet
    Incoming(Packet),
    /// Outgoing packet
    Outgoing(Packet),
}

/// Mock MQTT packet
#[derive(Debug, Clone)]
pub enum Packet {
    /// Publish packet
    Publish(Publish),
    /// Connect packet
    Connect,
    /// Disconnect packet
    Disconnect,
    /// Subscribe packet
    Subscribe,
    /// Unsubscribe packet
    Unsubscribe,
}

/// Mock MQTT publish packet
#[derive(Debug, Clone)]
pub struct Publish {
    /// The topic this message was sent to
    pub topic: String,
    /// The message payload
    pub payload: Vec<u8>,
    /// The QoS level
    pub qos: QoS,
    /// Whether the message should be retained
    pub retain: bool,
}

/// Mock MQTT options
#[derive(Debug, Clone)]
pub struct MqttOptions {
    /// Client ID
    pub client_id: String,
    /// Broker host
    pub host: String,
    /// Broker port
    pub port: u16,
    /// Keep alive interval
    pub keep_alive: Duration,
    /// Clean session flag
    pub clean_session: bool,
    /// Transport configuration
    pub transport: Transport,
}

impl MqttOptions {
    /// Create new MQTT options
    pub fn new<S: Into<String>>(client_id: S, host: S, port: u16) -> Self {
        Self {
            client_id: client_id.into(),
            host: host.into(),
            port,
            keep_alive: Duration::from_secs(60),
            clean_session: true,
            transport: Transport::Tcp,
        }
    }

    /// Set keep alive interval
    pub fn set_keep_alive(&mut self, keep_alive: Duration) {
        self.keep_alive = keep_alive;
    }

    /// Set clean session flag
    pub fn set_clean_session(&mut self, clean_session: bool) {
        self.clean_session = clean_session;
    }

    /// Set transport
    pub fn set_transport(&mut self, transport: Transport) {
        self.transport = transport;
    }
}

/// Mock MQTT transport
#[derive(Debug, Clone)]
pub enum Transport {
    /// TCP transport
    Tcp,
    /// TLS transport
    Tls(TlsConfiguration),
}

/// Mock MQTT TLS configuration
#[derive(Debug, Clone)]
pub enum TlsConfiguration {
    /// Simple TLS configuration
    Simple {
        /// CA certificates
        ca: Vec<u8>,
        /// ALPN protocols
        alpn: Option<Vec<String>>,
        /// Client authentication
        client_auth: Option<(Vec<u8>, Vec<u8>)>,
    },
}

/// Create a new mock MQTT client and event loop
pub fn new(_options: MqttOptions, _cap: usize) -> (MockMqttClient, MockEventLoop) {
    let client = MockMqttClient::new();
    let event_loop = MockEventLoop::new(
        client.message_store.clone(),
        client.connection_state.clone(),
    );

    (client, event_loop)
}

/// Create a TLS transport with the given configuration
pub fn tls_with_config(config: TlsConfiguration) -> Transport {
    Transport::Tls(config)
}
