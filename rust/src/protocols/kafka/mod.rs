// Kafka protocol implementation for HybridPipe
//
// This module provides an implementation of the HybridPipe interface for Apache Kafka.

use std::any::Any;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::Duration;
use async_trait::async_trait;
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::producer::{FutureProducer, FutureRecord};
use rdkafka::message::OwnedHeaders;
use rdkafka::util::Timeout;
use tokio::sync::RwLock;
use tokio::task::JoinHandle;
use uuid::Uuid;
use log::{debug, error, info, warn};

use crate::core::{BrokerType, HybridPipe, Process, Error, Result, BoxedHybridPipe, context};
use crate::core::config::{get_config, KafkaConfig};
use crate::core::registry::register_router_factory;
use crate::serialization::{encode, decode};

/// Packet implements the HybridPipe interface for Kafka messaging system.
pub struct Packet {
    /// Readers maps pipe names to their Kafka consumers
    readers: RwLock<HashMap<String, Arc<StreamConsumer>>>,
    /// Writer is the Kafka producer
    writer: Mutex<Option<FutureProducer>>,
    /// Server is the Kafka server address with port
    server: String,
    /// Reader tasks maps pipe names to their consumer task handles
    reader_tasks: RwLock<HashMap<String, JoinHandle<()>>>,
    /// Config holds the Kafka configuration
    config: KafkaConfig,
    /// Connected flag indicates if the packet is connected
    connected: Mutex<bool>,
}

impl Packet {
    /// Create a new Kafka packet.
    pub fn new() -> Self {
        // Get the Kafka configuration
        let config = match get_config() {
            Ok(config) => config.kafka.clone(),
            Err(_) => KafkaConfig {
                k_servers: "localhost".to_string(),
                k_lport: 9093,
                k_timeout: 10,
                kafka_cert_file: String::new(),
                kafka_key_file: String::new(),
                kafka_ca_file: String::new(),
            },
        };

        // Create the server address
        let server = format!("{}:{}", config.k_servers, config.k_lport);

        Self {
            readers: RwLock::new(HashMap::new()),
            writer: Mutex::new(None),
            server,
            reader_tasks: RwLock::new(HashMap::new()),
            config,
            connected: Mutex::new(false),
        }
    }

    /// Create a Kafka producer configuration.
    fn create_producer_config(&self) -> ClientConfig {
        let mut config = ClientConfig::new();
        config.set("bootstrap.servers", &self.server);
        config.set("message.timeout.ms", &self.config.k_timeout.to_string());
        
        // Add TLS configuration if certificate files are provided
        if !self.config.kafka_cert_file.is_empty() && !self.config.kafka_key_file.is_empty() && !self.config.kafka_ca_file.is_empty() {
            config.set("security.protocol", "ssl");
            config.set("ssl.certificate.location", &self.config.kafka_cert_file);
            config.set("ssl.key.location", &self.config.kafka_key_file);
            config.set("ssl.ca.location", &self.config.kafka_ca_file);
        }
        
        config
    }

    /// Create a Kafka consumer configuration.
    fn create_consumer_config(&self, group_id: &str) -> ClientConfig {
        let mut config = ClientConfig::new();
        config.set("bootstrap.servers", &self.server);
        config.set("group.id", group_id);
        config.set("enable.auto.commit", "true");
        config.set("auto.offset.reset", "earliest");
        config.set("session.timeout.ms", &(self.config.k_timeout * 1000).to_string());
        
        // Add TLS configuration if certificate files are provided
        if !self.config.kafka_cert_file.is_empty() && !self.config.kafka_key_file.is_empty() && !self.config.kafka_ca_file.is_empty() {
            config.set("security.protocol", "ssl");
            config.set("ssl.certificate.location", &self.config.kafka_cert_file);
            config.set("ssl.key.location", &self.config.kafka_key_file);
            config.set("ssl.ca.location", &self.config.kafka_ca_file);
        }
        
        config
    }
}

#[async_trait]
impl HybridPipe for Packet {
    async fn connect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if *connected {
            return Err(Error::AlreadyConnected);
        }

        // Create the Kafka producer
        let producer_config = self.create_producer_config();
        let producer: FutureProducer = producer_config.create()
            .map_err(|e| Error::ConnectionError(format!("Failed to create Kafka producer: {}", e)))?;
        
        // Store the producer
        let mut writer = self.writer.lock().unwrap();
        *writer = Some(producer);
        
        // Set connected flag
        *connected = true;
        
        info!("Connected to Kafka server at {}", self.server);
        Ok(())
    }

    async fn disconnect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }

        // Cancel all reader tasks
        let mut reader_tasks = self.reader_tasks.write().await;
        for (pipe, task) in reader_tasks.drain() {
            task.abort();
            debug!("Cancelled reader task for pipe: {}", pipe);
        }

        // Clear the readers
        let mut readers = self.readers.write().await;
        readers.clear();

        // Clear the writer
        let mut writer = self.writer.lock().unwrap();
        *writer = None;

        // Set connected flag
        *connected = false;

        info!("Disconnected from Kafka server at {}", self.server);
        Ok(())
    }

    async fn dispatch(&self, pipe: &str, data: Box<dyn Any + Send + Sync>) -> Result<()> {
        let connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }

        // Get the writer
        let writer = self.writer.lock().unwrap();
        let producer = writer.as_ref().ok_or(Error::NotConnected)?;

        // Downcast the data to a serializable type
        let serializable_data = match data.downcast_ref::<Box<dyn Any + Send + Sync>>() {
            Some(data) => data,
            None => {
                return Err(Error::SerializationError(
                    "Failed to downcast data to serializable type".to_string(),
                ));
            }
        };

        // Encode the data
        let bytes = encode(serializable_data)
            .map_err(|e| Error::SerializationError(format!("Failed to encode message: {}", e)))?;

        // Create a Kafka record
        let record = FutureRecord::to(pipe)
            .payload(&bytes)
            .key(pipe)
            .headers(OwnedHeaders::new());

        // Send the record
        let timeout = Timeout::After(Duration::from_secs(self.config.k_timeout as u64));
        match producer.send(record, timeout).await {
            Ok(_) => Ok(()),
            Err((e, _)) => Err(Error::DispatchError(format!("Failed to send message: {}", e))),
        }
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

    async fn accept<F>(&self, pipe: &str, callback: F) -> Result<()>
    where
        F: Fn(Box<dyn Any + Send + Sync>) + Send + Sync + 'static,
    {
        // Create a Process wrapper for the callback
        let process: Process = Box::new(move |data| {
            let decoded: Box<dyn Any + Send + Sync> = match decode(&data) {
                Ok(decoded) => Box::new(decoded),
                Err(e) => return Err(e),
            };
            callback(decoded);
            Ok(())
        });

        self.subscribe(pipe, process).await
    }

    async fn subscribe(&self, pipe: &str, callback: Process) -> Result<()> {
        let connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }

        // Check if already subscribed
        let readers = self.readers.read().await;
        if readers.contains_key(pipe) {
            return Err(Error::AlreadySubscribed(pipe.to_string()));
        }
        drop(readers);

        // Create a unique group ID for this subscription
        let group_id = format!("hybridpipe-{}", Uuid::new_v4());

        // Create the consumer
        let consumer_config = self.create_consumer_config(&group_id);
        let consumer: StreamConsumer = consumer_config.create()
            .map_err(|e| Error::SubscriptionError(format!("Failed to create Kafka consumer: {}", e)))?;

        // Subscribe to the topic
        consumer.subscribe(&[pipe])
            .map_err(|e| Error::SubscriptionError(format!("Failed to subscribe to topic: {}", e)))?;

        // Store the consumer
        let consumer = Arc::new(consumer);
        let mut readers = self.readers.write().await;
        readers.insert(pipe.to_string(), consumer.clone());
        drop(readers);

        // Create a task to process messages
        let pipe_owned = pipe.to_string();
        let callback_owned = callback;
        let task = tokio::spawn(async move {
            info!("Started Kafka consumer for pipe: {}", pipe_owned);
            
            loop {
                match consumer.recv().await {
                    Ok(message) => {
                        if let Some(payload) = message.payload() {
                            if let Err(e) = callback_owned(payload.to_vec()) {
                                error!("Error processing message from pipe {}: {}", pipe_owned, e);
                            }
                        }
                    }
                    Err(e) => {
                        error!("Error receiving message from Kafka: {}", e);
                    }
                }
            }
        });

        // Store the task
        let mut reader_tasks = self.reader_tasks.write().await;
        reader_tasks.insert(pipe.to_string(), task);

        info!("Subscribed to Kafka topic: {}", pipe);
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
        let connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }

        // Check if subscribed
        let readers = self.readers.read().await;
        if !readers.contains_key(pipe) {
            return Err(Error::NotSubscribed(pipe.to_string()));
        }
        drop(readers);

        // Cancel the reader task
        let mut reader_tasks = self.reader_tasks.write().await;
        if let Some(task) = reader_tasks.remove(pipe) {
            task.abort();
        }

        // Remove the reader
        let mut readers = self.readers.write().await;
        readers.remove(pipe);

        info!("Unsubscribed from Kafka topic: {}", pipe);
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

/// Register the Kafka protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::KAFKA, Box::new(|| {
        Arc::new(Packet::new())
    }));
}
