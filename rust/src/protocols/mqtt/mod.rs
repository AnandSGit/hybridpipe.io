// MQTT protocol implementation for HybridPipe
//
// This module provides an implementation of the HybridPipe interface for MQTT messaging system.

use async_trait::async_trait;
use log::{debug, error, info, warn};
use rumqttc::{
    AsyncClient, ConnectionError, Event, MqttOptions, Packet as MqttPacket, QoS, TlsConfiguration,
    Transport,
};
use std::any::Any;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::Duration;
use tokio::sync::RwLock;
use tokio::task::JoinHandle;

use crate::core::config::{get_config, MQTTConfig};
use crate::core::registry::register_router_factory;
use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};

/// Packet implements the HybridPipe interface for MQTT messaging system.
pub struct Packet {
    /// Client is the MQTT client
    client: Mutex<Option<AsyncClient>>,
    /// Event loop handle
    event_loop_handle: Mutex<Option<JoinHandle<()>>>,
    /// Subscribers maps pipe names to their callback functions
    subscribers: RwLock<HashMap<String, Process>>,
    /// Server is the MQTT broker address
    server: String,
    /// Config holds the MQTT configuration
    config: MQTTConfig,
    /// Connected flag indicates if the packet is connected
    connected: Mutex<bool>,
}

impl Packet {
    /// Create a new MQTT packet.
    pub fn new() -> Self {
        // Get the MQTT configuration
        let config = match get_config() {
            Ok(config) => config.mqtt.clone(),
            Err(_) => MQTTConfig {
                mqtt_broker: "tcp://localhost:1883".to_string(),
            },
        };

        // Extract the server address from the broker URL
        let server = config.mqtt_broker.clone();

        Self {
            client: Mutex::new(None),
            event_loop_handle: Mutex::new(None),
            subscribers: RwLock::new(HashMap::new()),
            server,
            config,
            connected: Mutex::new(false),
        }
    }

    /// Parse the MQTT broker URL to extract host, port, and protocol.
    fn parse_broker_url(&self) -> Result<(String, u16, bool)> {
        let url = &self.config.mqtt_broker;

        // Check if the URL starts with tcp:// or ssl://
        let (protocol, rest) = if url.starts_with("tcp://") {
            ("tcp", &url[6..])
        } else if url.starts_with("ssl://") {
            ("ssl", &url[6..])
        } else {
            return Err(Error::InvalidConfiguration(format!(
                "Invalid MQTT broker URL: {}. Must start with tcp:// or ssl://",
                url
            )));
        };

        // Split the rest by : to get host and port
        let parts: Vec<&str> = rest.split(':').collect();
        if parts.len() != 2 {
            return Err(Error::InvalidConfiguration(format!(
                "Invalid MQTT broker URL: {}. Expected format: tcp://host:port or ssl://host:port",
                url
            )));
        }

        let host = parts[0].to_string();
        let port = parts[1]
            .parse::<u16>()
            .map_err(|e| Error::InvalidConfiguration(format!("Invalid MQTT port: {}", e)))?;

        let use_tls = protocol == "ssl";

        Ok((host, port, use_tls))
    }
}

#[async_trait]
impl HybridPipe for Packet {
    async fn connect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if *connected {
            return Err(Error::AlreadyConnected);
        }

        // Parse the broker URL
        let (host, port, use_tls) = self.parse_broker_url()?;

        // Create MQTT options
        let mut mqtt_options = MqttOptions::new("hybridpipe-client", host, port);

        // Set connection options
        mqtt_options.set_keep_alive(Duration::from_secs(30));
        mqtt_options.set_clean_session(true);

        // Set TLS if needed
        if use_tls {
            let tls_config = TlsConfiguration::Simple {
                ca: Vec::new(),
                alpn: None,
                client_auth: None,
            };
            mqtt_options.set_transport(Transport::tls_with_config(tls_config));
        }

        // Create the client and event loop
        let (client, mut event_loop) = AsyncClient::new(mqtt_options, 10);

        // Store the client
        let mut client_guard = self.client.lock().unwrap();
        *client_guard = Some(client);
        drop(client_guard);

        // Start the event loop
        let event_loop_handle = tokio::spawn(async move {
            loop {
                match event_loop.poll().await {
                    Ok(Event::Incoming(MqttPacket::Publish(publish))) => {
                        debug!("Received MQTT message on topic: {}", publish.topic);
                        // The subscribers will be handled separately
                    }
                    Ok(_) => {
                        // Ignore other events
                    }
                    Err(ConnectionError::Io(e)) => {
                        error!("MQTT connection error: {}", e);
                        break;
                    }
                    Err(e) => {
                        error!("MQTT error: {}", e);
                    }
                }
            }
        });

        // Store the event loop handle
        let mut handle_guard = self.event_loop_handle.lock().unwrap();
        *handle_guard = Some(event_loop_handle);

        // Set connected flag
        *connected = true;

        info!("Connected to MQTT broker at {}", self.server);
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

        // Cancel the event loop
        {
            let mut handle_guard = self.event_loop_handle.lock().unwrap();
            if let Some(handle) = handle_guard.take() {
                handle.abort();
            }
        }

        // Get the client to disconnect
        let client_to_disconnect = {
            let mut client_guard = self.client.lock().unwrap();
            client_guard.take()
        };

        // Disconnect the client if it exists
        if let Some(client) = client_to_disconnect {
            if let Err(e) = client.disconnect().await {
                warn!("Error disconnecting from MQTT broker: {}", e);
            }
        }

        // Clear the subscribers
        let mut subscribers = self.subscribers.write().await;
        subscribers.clear();

        info!("Disconnected from MQTT broker at {}", self.server);
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

        // Get the client
        let client = {
            let client_guard = self.client.lock().unwrap();
            match client_guard.as_ref() {
                Some(client) => client.clone(),
                None => return Err(Error::NotConnected),
            }
        };

        // Convert the data to a JSON string
        let json_string = serde_json::to_string(&format!("{:?}", data))
            .map_err(|e| Error::SerializationError(format!("Failed to encode message: {}", e)))?;

        // Convert the JSON string to bytes
        let bytes = json_string.into_bytes();

        // Publish the message
        client
            .publish(pipe, QoS::AtLeastOnce, false, bytes.to_vec())
            .await
            .map_err(|e| Error::DispatchError(format!("Failed to publish message: {}", e)))?;

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

        // Get the client
        let client = {
            let client_guard = self.client.lock().unwrap();
            match client_guard.as_ref() {
                Some(client) => client.clone(),
                None => return Err(Error::NotConnected),
            }
        };

        // Subscribe to the topic
        client
            .subscribe(pipe, QoS::AtLeastOnce)
            .await
            .map_err(|e| {
                Error::SubscriptionError(format!("Failed to subscribe to topic: {}", e))
            })?;

        // Store the callback
        let mut subscribers = self.subscribers.write().await;
        subscribers.insert(pipe.to_string(), callback);

        info!("Subscribed to MQTT topic: {}", pipe);
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

        // Get the client
        let client = {
            let client_guard = self.client.lock().unwrap();
            match client_guard.as_ref() {
                Some(client) => client.clone(),
                None => return Err(Error::NotConnected),
            }
        };

        // Unsubscribe from the topic
        client.unsubscribe(pipe).await.map_err(|e| {
            Error::UnsubscriptionError(format!("Failed to unsubscribe from topic: {}", e))
        })?;

        // Remove the callback
        let mut subscribers = self.subscribers.write().await;
        subscribers.remove(pipe);

        info!("Unsubscribed from MQTT topic: {}", pipe);
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

/// Register the MQTT protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::MQTT, Box::new(|| Arc::new(Packet::new())));
}

#[cfg(test)]
mod tests;
