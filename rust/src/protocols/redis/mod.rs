// Redis protocol implementation for HybridPipe
//
// This module provides an implementation of the HybridPipe interface for Redis pub/sub.

use async_trait::async_trait;
use futures::StreamExt;
use log::{debug, error, info};
use redis::{
    aio::MultiplexedConnection, AsyncCommands, Client,
};
use std::any::Any;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tokio::sync::RwLock;
use tokio::task::JoinHandle;

use crate::core::config::{get_config, RedisConfig};
use crate::core::registry::register_router_factory;
use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};

/// Packet implements the HybridPipe interface for Redis pub/sub.
pub struct Packet {
    /// Client is the Redis client
    client: Mutex<Option<Client>>,
    /// Connection is the Redis connection for publishing
    connection: Mutex<Option<MultiplexedConnection>>,
    /// Subscriber tasks maps pipe names to their subscriber task handles
    subscriber_tasks: RwLock<HashMap<String, JoinHandle<()>>>,
    /// Server is the Redis server address with port
    server: String,
    /// Config holds the Redis configuration
    config: RedisConfig,
    /// Connected flag indicates if the packet is connected
    connected: Mutex<bool>,
}

impl Packet {
    /// Create a new Redis packet.
    pub fn new() -> Self {
        // Get the Redis configuration
        let config = match get_config() {
            Ok(config) => config.redis.clone(),
            Err(_) => RedisConfig {
                redis_address: "localhost".to_string(),
            },
        };

        // Create the server address
        let server = format!("redis://{}:{}", config.redis_address, 6379);

        Self {
            client: Mutex::new(None),
            connection: Mutex::new(None),
            subscriber_tasks: RwLock::new(HashMap::new()),
            server,
            config,
            connected: Mutex::new(false),
        }
    }
}

#[async_trait]
impl HybridPipe for Packet {
    async fn connect(&self) -> Result<()> {
        // Check if already connected
        {
            let connected = self.connected.lock().unwrap();
            if *connected {
                return Err(Error::AlreadyConnected);
            }
        }

        // Create Redis client
        let client = Client::open(self.server.clone())
            .map_err(|e| Error::ConnectionError(format!("Failed to create Redis client: {}", e)))?;

        // Create a multiplexed connection for publishing
        let connection = client
            .get_multiplexed_async_connection()
            .await
            .map_err(|e| Error::ConnectionError(format!("Failed to connect to Redis: {}", e)))?;

        // Store the client and connection
        {
            let mut client_guard = self.client.lock().unwrap();
            *client_guard = Some(client);
        }

        {
            let mut connection_guard = self.connection.lock().unwrap();
            *connection_guard = Some(connection);
        }

        // Set connected flag
        {
            let mut connected = self.connected.lock().unwrap();
            *connected = true;
        }

        info!("Connected to Redis server at {}", self.server);
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

        // Cancel all subscriber tasks
        let mut subscriber_tasks = self.subscriber_tasks.write().await;
        for (pipe, task) in subscriber_tasks.drain() {
            task.abort();
            debug!("Cancelled subscriber task for pipe: {}", pipe);
        }

        // Clear the connection
        {
            let mut connection_guard = self.connection.lock().unwrap();
            *connection_guard = None;
        }

        // Clear the client
        {
            let mut client_guard = self.client.lock().unwrap();
            *client_guard = None;
        }

        info!("Disconnected from Redis server at {}", self.server);
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

        // Get the connection
        let connection = {
            let connection_guard = self.connection.lock().unwrap();
            match connection_guard.as_ref() {
                Some(conn) => conn.clone(),
                None => return Err(Error::NotConnected),
            }
        };

        // Convert the data to a JSON string
        let json_string = serde_json::to_string(&format!("{:?}", data))
            .map_err(|e| Error::SerializationError(format!("Failed to encode message: {}", e)))?;

        // Convert the JSON string to bytes
        let bytes = json_string.into_bytes();

        // Publish the message
        // We need to use a mutable connection
        let mut conn = connection;
        conn.publish(pipe, bytes)
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
        let subscriber_tasks = self.subscriber_tasks.read().await;
        if subscriber_tasks.contains_key(pipe) {
            return Err(Error::AlreadySubscribed(pipe.to_string()));
        }
        drop(subscriber_tasks);

        // Get the client
        let client = {
            let client_guard = self.client.lock().unwrap();
            let client = client_guard.as_ref().ok_or(Error::NotConnected)?;
            client.clone()
        };

        // Create a new connection for subscribing
        let pubsub_conn = client.get_async_connection().await.map_err(|e| {
            Error::SubscriptionError(format!("Failed to create Redis connection: {}", e))
        })?;

        // Create a task to process messages
        let pipe_owned = pipe.to_string();
        let callback_owned = callback;
        let task = tokio::spawn(async move {
            info!("Started Redis subscriber for pipe: {}", pipe_owned);

            // Subscribe to the channel
            let mut pubsub = pubsub_conn.into_pubsub();
            if let Err(e) = pubsub.subscribe(&pipe_owned).await {
                error!("Failed to subscribe to Redis channel {}: {}", pipe_owned, e);
                return;
            }

            // Get a stream of messages
            let mut stream = pubsub.on_message();

            // Process messages
            while let Some(msg) = stream.next().await {
                let payload: Vec<u8> = match msg.get_payload() {
                    Ok(payload) => payload,
                    Err(e) => {
                        error!("Failed to get payload from Redis message: {}", e);
                        continue;
                    }
                };

                if let Err(e) = callback_owned(payload) {
                    error!("Error processing message from pipe {}: {}", pipe_owned, e);
                }
            }
        });

        // Store the task
        let mut subscriber_tasks = self.subscriber_tasks.write().await;
        subscriber_tasks.insert(pipe.to_string(), task);

        info!("Subscribed to Redis channel: {}", pipe);
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
        let subscriber_tasks = self.subscriber_tasks.read().await;
        if !subscriber_tasks.contains_key(pipe) {
            return Err(Error::NotSubscribed(pipe.to_string()));
        }
        drop(subscriber_tasks);

        // Cancel the subscriber task
        let mut subscriber_tasks = self.subscriber_tasks.write().await;
        if let Some(task) = subscriber_tasks.remove(pipe) {
            task.abort();
        }

        info!("Unsubscribed from Redis channel: {}", pipe);
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

/// Register the Redis protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::REDIS, Box::new(|| Arc::new(Packet::new())));
}
