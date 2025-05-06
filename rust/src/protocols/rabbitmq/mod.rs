// RabbitMQ protocol implementation for HybridPipe
//
// This module provides an implementation of the HybridPipe interface for RabbitMQ messaging system.

use async_trait::async_trait;
use futures::StreamExt;
use lapin::{
    options::{
        BasicAckOptions, BasicConsumeOptions, BasicPublishOptions, ExchangeDeclareOptions,
        QueueBindOptions, QueueDeclareOptions,
    },
    types::FieldTable,
    BasicProperties, Channel, Connection, ConnectionProperties, Consumer, ExchangeKind,
};
use log::{debug, error, info, warn};
use std::any::Any;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tokio::sync::RwLock;
use tokio::task::JoinHandle;

use crate::core::config::{get_config, RabbitMQConfig};
use crate::core::registry::register_router_factory;
use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};

/// Packet implements the HybridPipe interface for RabbitMQ messaging system.
pub struct Packet {
    /// Connection is the RabbitMQ connection
    connection: Mutex<Option<Connection>>,
    /// Channel is the RabbitMQ channel
    channel: Mutex<Option<Channel>>,
    /// Consumers maps pipe names to their consumers
    consumers: RwLock<HashMap<String, Consumer>>,
    /// Consumer tasks maps pipe names to their consumer task handles
    consumer_tasks: RwLock<HashMap<String, JoinHandle<()>>>,
    /// Server is the RabbitMQ server address
    server: String,
    /// Config holds the RabbitMQ configuration
    config: RabbitMQConfig,
    /// Connected flag indicates if the packet is connected
    connected: Mutex<bool>,
    /// Exchange name
    exchange: String,
}

impl Packet {
    /// Create a new RabbitMQ packet.
    pub fn new() -> Self {
        // Get the RabbitMQ configuration
        let config = match get_config() {
            Ok(config) => config.rabbitmq.clone(),
            Err(_) => RabbitMQConfig {
                r_server_port: "amqp://guest:guest@localhost:5672/".to_string(),
            },
        };

        Self {
            connection: Mutex::new(None),
            channel: Mutex::new(None),
            consumers: RwLock::new(HashMap::new()),
            consumer_tasks: RwLock::new(HashMap::new()),
            server: config.r_server_port.clone(),
            config,
            connected: Mutex::new(false),
            exchange: "hybridpipe".to_string(),
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

        // Connect to RabbitMQ
        let connection = Connection::connect(
            &self.server,
            ConnectionProperties::default().with_connection_name("hybridpipe".into()),
        )
        .await
        .map_err(|e| Error::ConnectionError(format!("Failed to connect to RabbitMQ: {}", e)))?;

        // Create a channel
        let channel = connection.create_channel().await.map_err(|e| {
            Error::ConnectionError(format!("Failed to create RabbitMQ channel: {}", e))
        })?;

        // Declare the exchange
        channel
            .exchange_declare(
                &self.exchange,
                ExchangeKind::Topic,
                ExchangeDeclareOptions {
                    durable: true,
                    auto_delete: false,
                    ..Default::default()
                },
                FieldTable::default(),
            )
            .await
            .map_err(|e| {
                Error::ConnectionError(format!("Failed to declare RabbitMQ exchange: {}", e))
            })?;

        // Store the connection and channel
        let mut connection_guard = self.connection.lock().unwrap();
        *connection_guard = Some(connection);
        drop(connection_guard);

        let mut channel_guard = self.channel.lock().unwrap();
        *channel_guard = Some(channel);
        drop(channel_guard);

        // Set connected flag
        {
            let mut connected = self.connected.lock().unwrap();
            *connected = true;
        }

        info!("Connected to RabbitMQ server at {}", self.server);
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

        // Cancel all consumer tasks
        let mut consumer_tasks = self.consumer_tasks.write().await;
        for (pipe, task) in consumer_tasks.drain() {
            task.abort();
            debug!("Cancelled consumer task for pipe: {}", pipe);
        }

        // Clear the consumers
        let mut consumers = self.consumers.write().await;
        consumers.clear();

        // Get the channel to close
        let channel_to_close = {
            let mut channel_guard = self.channel.lock().unwrap();
            channel_guard.take()
        };

        // Close the channel if it exists
        if let Some(channel) = channel_to_close {
            if let Err(e) = channel.close(0, "Normal shutdown").await {
                warn!("Error closing RabbitMQ channel: {}", e);
            }
        }

        // Get the connection to close
        let connection_to_close = {
            let mut connection_guard = self.connection.lock().unwrap();
            connection_guard.take()
        };

        // Close the connection if it exists
        if let Some(connection) = connection_to_close {
            if let Err(e) = connection.close(0, "Normal shutdown").await {
                warn!("Error closing RabbitMQ connection: {}", e);
            }
        }

        info!("Disconnected from RabbitMQ server at {}", self.server);
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

        // Get the channel
        let channel = {
            let channel_guard = self.channel.lock().unwrap();
            match channel_guard.as_ref() {
                Some(channel) => channel.clone(),
                None => return Err(Error::NotConnected),
            }
        };

        // Convert the data to a JSON string
        let json_string = serde_json::to_string(&format!("{:?}", data))
            .map_err(|e| Error::SerializationError(format!("Failed to encode message: {}", e)))?;

        // Convert the JSON string to bytes
        let bytes = json_string.into_bytes();

        // Publish the message
        channel
            .basic_publish(
                &self.exchange,
                pipe,
                BasicPublishOptions::default(),
                &bytes.to_vec(),
                BasicProperties::default(),
            )
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
        let consumers = self.consumers.read().await;
        if consumers.contains_key(pipe) {
            return Err(Error::AlreadySubscribed(pipe.to_string()));
        }
        drop(consumers);

        // Get the channel
        let channel = {
            let channel_guard = self.channel.lock().unwrap();
            let channel = channel_guard.as_ref().ok_or(Error::NotConnected)?;
            channel.clone()
        };

        // Declare a queue
        let queue_name = format!("hybridpipe.{}", pipe);
        let queue = channel
            .queue_declare(
                &queue_name,
                QueueDeclareOptions {
                    durable: true,
                    auto_delete: false,
                    ..Default::default()
                },
                FieldTable::default(),
            )
            .await
            .map_err(|e| Error::SubscriptionError(format!("Failed to declare queue: {}", e)))?;

        // Bind the queue to the exchange
        channel
            .queue_bind(
                &queue_name,
                &self.exchange,
                pipe,
                QueueBindOptions::default(),
                FieldTable::default(),
            )
            .await
            .map_err(|e| Error::SubscriptionError(format!("Failed to bind queue: {}", e)))?;

        // Create a consumer
        let mut consumer = channel
            .basic_consume(
                &queue_name,
                &format!("hybridpipe-consumer-{}", pipe),
                BasicConsumeOptions::default(),
                FieldTable::default(),
            )
            .await
            .map_err(|e| Error::SubscriptionError(format!("Failed to create consumer: {}", e)))?;

        // Store the consumer
        let mut consumers = self.consumers.write().await;
        consumers.insert(pipe.to_string(), consumer.clone());
        drop(consumers);

        // Create a task to process messages
        let pipe_owned = pipe.to_string();
        let callback_owned = callback;
        let task = tokio::spawn(async move {
            info!("Started RabbitMQ consumer for pipe: {}", pipe_owned);

            while let Some(delivery) = consumer.next().await {
                match delivery {
                    Ok(delivery) => {
                        if let Err(e) = callback_owned(delivery.data.clone()) {
                            error!("Error processing message from pipe {}: {}", pipe_owned, e);
                        }

                        // Acknowledge the message
                        if let Err(e) = delivery.ack(BasicAckOptions::default()).await {
                            error!("Failed to acknowledge message: {}", e);
                        }
                    }
                    Err(e) => {
                        error!("Error receiving message from RabbitMQ: {}", e);
                    }
                }
            }
        });

        // Store the task
        let mut consumer_tasks = self.consumer_tasks.write().await;
        consumer_tasks.insert(pipe.to_string(), task);

        info!("Subscribed to RabbitMQ routing key: {}", pipe);
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
        let consumers = self.consumers.read().await;
        if !consumers.contains_key(pipe) {
            return Err(Error::NotSubscribed(pipe.to_string()));
        }
        drop(consumers);

        // Cancel the consumer task
        let mut consumer_tasks = self.consumer_tasks.write().await;
        if let Some(task) = consumer_tasks.remove(pipe) {
            task.abort();
        }

        // Remove the consumer
        let mut consumers = self.consumers.write().await;
        consumers.remove(pipe);

        info!("Unsubscribed from RabbitMQ routing key: {}", pipe);
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

/// Register the RabbitMQ protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::RABBITMQ, Box::new(|| Arc::new(Packet::new())));
}
