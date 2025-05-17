// ZeroMQ protocol implementation for HybridPipe
//
// This module provides an implementation of the HybridPipe interface for ZeroMQ.

use async_trait::async_trait;
use log::{debug, error, info, warn};
use std::any::Any;
use std::collections::HashMap;
use std::sync::{Arc, Mutex, RwLock};
use std::time::Duration;
use tokio::sync::mpsc;
use tokio::task;
use tokio::time;
use zmq::{Context, Socket, SocketType};

use crate::core::registry::register_router_factory;
use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};
use crate::serialization::{deserialize, serialize};

/// Configuration for ZeroMQ.
#[derive(Debug, Clone)]
pub struct ZeroMQConfig {
    /// Publisher endpoint (e.g., "tcp://127.0.0.1:5555")
    pub publisher_endpoint: String,
    /// Subscriber endpoint (e.g., "tcp://127.0.0.1:5556")
    pub subscriber_endpoint: String,
    /// Publisher bind flag (true = bind, false = connect)
    pub publisher_bind: bool,
    /// Subscriber bind flag (true = bind, false = connect)
    pub subscriber_bind: bool,
    /// Publisher high water mark
    pub publisher_hwm: i32,
    /// Subscriber high water mark
    pub subscriber_hwm: i32,
    /// Publisher linger period in milliseconds
    pub publisher_linger: i32,
    /// Subscriber linger period in milliseconds
    pub subscriber_linger: i32,
    /// Reconnect interval in milliseconds
    pub reconnect_interval: u64,
    /// Maximum reconnect attempts (0 = infinite)
    pub max_reconnect_attempts: u32,
}

impl Default for ZeroMQConfig {
    fn default() -> Self {
        Self {
            publisher_endpoint: "tcp://127.0.0.1:5555".to_string(),
            subscriber_endpoint: "tcp://127.0.0.1:5556".to_string(),
            publisher_bind: false,
            subscriber_bind: false,
            publisher_hwm: 1000,
            subscriber_hwm: 1000,
            publisher_linger: 0,
            subscriber_linger: 0,
            reconnect_interval: 1000,
            max_reconnect_attempts: 10,
        }
    }
}

/// Packet implements the HybridPipe interface for ZeroMQ.
pub struct Packet {
    /// ZeroMQ context
    context: Arc<Context>,
    /// Publisher socket
    publisher: Mutex<Option<Socket>>,
    /// Subscriber socket
    subscriber: Mutex<Option<Socket>>,
    /// Callbacks for each pipe
    callbacks: RwLock<HashMap<String, Vec<Process>>>,
    /// Configuration
    config: ZeroMQConfig,
    /// Connected flag
    connected: Mutex<bool>,
    /// Shutdown channel
    shutdown_tx: Mutex<Option<mpsc::Sender<()>>>,
}

impl Packet {
    /// Create a new ZeroMQ packet with the specified configuration.
    pub fn new(config: ZeroMQConfig) -> Self {
        Self {
            context: Arc::new(Context::new()),
            publisher: Mutex::new(None),
            subscriber: Mutex::new(None),
            callbacks: RwLock::new(HashMap::new()),
            config,
            connected: Mutex::new(false),
            shutdown_tx: Mutex::new(None),
        }
    }

    /// Create a new ZeroMQ packet with default configuration.
    pub fn new_default() -> Self {
        Self::new(ZeroMQConfig::default())
    }

    /// Create a publisher socket.
    fn create_publisher(&self) -> Result<Socket> {
        let publisher = self.context.socket(SocketType::PUB)?;
        publisher.set_sndhwm(self.config.publisher_hwm)?;
        publisher.set_linger(self.config.publisher_linger)?;

        if self.config.publisher_bind {
            debug!("Binding publisher to {}", self.config.publisher_endpoint);
            publisher.bind(&self.config.publisher_endpoint)?;
        } else {
            debug!("Connecting publisher to {}", self.config.publisher_endpoint);
            publisher.connect(&self.config.publisher_endpoint)?;
        }

        Ok(publisher)
    }

    /// Create a subscriber socket.
    fn create_subscriber(&self) -> Result<Socket> {
        let subscriber = self.context.socket(SocketType::SUB)?;
        subscriber.set_rcvhwm(self.config.subscriber_hwm)?;
        subscriber.set_linger(self.config.subscriber_linger)?;

        if self.config.subscriber_bind {
            debug!("Binding subscriber to {}", self.config.subscriber_endpoint);
            subscriber.bind(&self.config.subscriber_endpoint)?;
        } else {
            debug!("Connecting subscriber to {}", self.config.subscriber_endpoint);
            subscriber.connect(&self.config.subscriber_endpoint)?;
        }

        Ok(subscriber)
    }

    /// Start the message receiver.
    async fn start_receiver(&self, subscriber: Arc<Socket>) -> Result<mpsc::Sender<()>> {
        let (shutdown_tx, mut shutdown_rx) = mpsc::channel::<()>(1);
        let callbacks = self.callbacks.clone();
        let config = self.config.clone();

        // Spawn a task to receive messages
        task::spawn(async move {
            let mut items = [subscriber.as_poll_item(zmq::POLLIN)];
            let mut reconnect_attempts = 0;

            'outer: loop {
                // Check for shutdown signal
                if shutdown_rx.try_recv().is_ok() {
                    debug!("Receiver shutting down");
                    break;
                }

                // Poll for messages
                match subscriber.poll(&mut items, 100) {
                    Ok(0) => {
                        // No messages, continue
                        continue;
                    }
                    Ok(_) => {
                        // Reset reconnect attempts on successful poll
                        reconnect_attempts = 0;

                        // Process all available messages
                        while items[0].is_readable() {
                            // Receive the message
                            let result = subscriber.recv_multipart(zmq::DONTWAIT);
                            match result {
                                Ok(parts) => {
                                    if parts.len() < 2 {
                                        warn!("Received invalid message: too few parts");
                                        continue;
                                    }

                                    // Extract the pipe and message data
                                    let pipe = match String::from_utf8(parts[0].clone()) {
                                        Ok(s) => s,
                                        Err(e) => {
                                            warn!("Failed to decode pipe name: {}", e);
                                            continue;
                                        }
                                    };
                                    let data = parts[1].clone();

                                    // Process the message
                                    let callbacks_guard = callbacks.read().unwrap();
                                    if let Some(callbacks) = callbacks_guard.get(&pipe) {
                                        for callback in callbacks {
                                            let callback = callback.clone();
                                            let data = data.clone();
                                            task::spawn(async move {
                                                if let Err(e) = callback(data).await {
                                                    error!("Error processing message: {}", e);
                                                }
                                            });
                                        }
                                    }

                                    // Poll again to check if there are more messages
                                    match subscriber.poll(&mut items, 0) {
                                        Ok(_) => continue,
                                        Err(e) => {
                                            error!("Error polling for messages: {}", e);
                                            break;
                                        }
                                    }
                                }
                                Err(zmq::Error::EAGAIN) => {
                                    // No more messages
                                    break;
                                }
                                Err(e) => {
                                    error!("Error receiving message: {}", e);
                                    // Try to reconnect
                                    reconnect_attempts += 1;
                                    if config.max_reconnect_attempts > 0
                                        && reconnect_attempts >= config.max_reconnect_attempts
                                    {
                                        error!("Max reconnect attempts reached, giving up");
                                        break 'outer;
                                    }
                                    warn!(
                                        "Reconnect attempt {}/{}",
                                        reconnect_attempts, config.max_reconnect_attempts
                                    );
                                    time::sleep(Duration::from_millis(config.reconnect_interval))
                                        .await;
                                    break;
                                }
                            }
                        }
                    }
                    Err(e) => {
                        error!("Error polling for messages: {}", e);
                        // Try to reconnect
                        reconnect_attempts += 1;
                        if config.max_reconnect_attempts > 0
                            && reconnect_attempts >= config.max_reconnect_attempts
                        {
                            error!("Max reconnect attempts reached, giving up");
                            break;
                        }
                        warn!(
                            "Reconnect attempt {}/{}",
                            reconnect_attempts, config.max_reconnect_attempts
                        );
                        time::sleep(Duration::from_millis(config.reconnect_interval)).await;
                    }
                }
            }

            debug!("Receiver task exited");
        });

        Ok(shutdown_tx)
    }
}

#[async_trait]
impl HybridPipe for Packet {
    async fn connect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if *connected {
            return Err(Error::AlreadyConnected);
        }

        // Create publisher socket
        let publisher = self.create_publisher()?;
        let mut publisher_guard = self.publisher.lock().unwrap();
        *publisher_guard = Some(publisher);
        drop(publisher_guard);

        // Create subscriber socket
        let subscriber = self.create_subscriber()?;
        let mut subscriber_guard = self.subscriber.lock().unwrap();
        *subscriber_guard = Some(subscriber);
        drop(subscriber_guard);

        // Get the subscriber socket for the receiver task
        let subscriber_guard = self.subscriber.lock().unwrap();
        let subscriber = subscriber_guard.as_ref().unwrap().clone();
        drop(subscriber_guard);

        // Start the receiver task
        let shutdown_tx = self.start_receiver(Arc::new(subscriber)).await?;
        let mut shutdown_tx_guard = self.shutdown_tx.lock().unwrap();
        *shutdown_tx_guard = Some(shutdown_tx);
        drop(shutdown_tx_guard);

        *connected = true;
        info!("Connected to ZeroMQ");
        Ok(())
    }

    async fn disconnect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }

        // Stop the receiver task
        let mut shutdown_tx_guard = self.shutdown_tx.lock().unwrap();
        if let Some(shutdown_tx) = shutdown_tx_guard.take() {
            if let Err(e) = shutdown_tx.send(()).await {
                warn!("Failed to send shutdown signal: {}", e);
            }
        }
        drop(shutdown_tx_guard);

        // Close the publisher socket
        let mut publisher_guard = self.publisher.lock().unwrap();
        *publisher_guard = None;
        drop(publisher_guard);

        // Close the subscriber socket
        let mut subscriber_guard = self.subscriber.lock().unwrap();
        *subscriber_guard = None;
        drop(subscriber_guard);

        *connected = false;
        info!("Disconnected from ZeroMQ");
        Ok(())
    }

    async fn dispatch(&self, pipe: &str, data: Box<dyn Any + Send + Sync>) -> Result<()> {
        let connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }
        drop(connected);

        // Serialize the data
        let serialized = serialize(&*data)?;

        // Get the publisher socket
        let publisher_guard = self.publisher.lock().unwrap();
        let publisher = publisher_guard
            .as_ref()
            .ok_or_else(|| Error::NotConnected)?;

        // Send the message
        publisher.send_multipart(&[pipe.as_bytes(), &serialized], 0)?;

        Ok(())
    }

    async fn dispatch_with_context(
        &self,
        ctx: &context::Context,
        pipe: &str,
        data: Box<dyn Any + Send + Sync>,
    ) -> Result<()> {
        // Check if the context is cancelled
        if ctx.is_cancelled() {
            return Err(Error::ContextCancelled);
        }

        // Dispatch the message
        self.dispatch(pipe, data).await
    }

    async fn subscribe(&self, pipe: &str, callback: Process) -> Result<()> {
        let connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }
        drop(connected);

        // Get the subscriber socket
        let subscriber_guard = self.subscriber.lock().unwrap();
        let subscriber = subscriber_guard
            .as_ref()
            .ok_or_else(|| Error::NotConnected)?;

        // Subscribe to the pipe
        subscriber.set_subscribe(pipe.as_bytes())?;
        drop(subscriber_guard);

        // Add the callback to the pipe
        let mut callbacks = self.callbacks.write().unwrap();
        let pipe_callbacks = callbacks.entry(pipe.to_string()).or_insert_with(Vec::new);
        pipe_callbacks.push(callback);

        Ok(())
    }

    async fn subscribe_with_context(
        &self,
        ctx: &context::Context,
        pipe: &str,
        callback: Process,
    ) -> Result<()> {
        // Check if the context is cancelled
        if ctx.is_cancelled() {
            return Err(Error::ContextCancelled);
        }

        // Create a wrapper that checks the context before calling the callback
        let ctx_clone = ctx.clone();
        let wrapper: Process = Box::new(move |data| {
            let ctx = ctx_clone.clone();
            let callback = callback.clone();
            Box::pin(async move {
                if ctx.is_cancelled() {
                    return Err(Error::ContextCancelled);
                }
                callback(data).await
            })
        });

        // Subscribe with the wrapper
        self.subscribe(pipe, wrapper).await
    }

    async fn remove(&self, pipe: &str) -> Result<()> {
        self.unsubscribe(pipe).await
    }

    async fn unsubscribe(&self, pipe: &str) -> Result<()> {
        let connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }
        drop(connected);

        // Get the subscriber socket
        let subscriber_guard = self.subscriber.lock().unwrap();
        let subscriber = subscriber_guard
            .as_ref()
            .ok_or_else(|| Error::NotConnected)?;

        // Unsubscribe from the pipe
        subscriber.set_unsubscribe(pipe.as_bytes())?;
        drop(subscriber_guard);

        // Remove the callbacks for the pipe
        let mut callbacks = self.callbacks.write().unwrap();
        callbacks.remove(pipe);

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

/// Register the ZeroMQ protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::ZEROMQ, Box::new(|| {
        Arc::new(Packet::new_default())
    }));
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;

    #[tokio::test]
    async fn test_zeromq_connect_disconnect() {
        let packet = Packet::new_default();
        assert!(!packet.is_connected());

        // Connect
        let result = packet.connect().await;
        assert!(result.is_ok());
        assert!(packet.is_connected());

        // Disconnect
        let result = packet.disconnect().await;
        assert!(result.is_ok());
        assert!(!packet.is_connected());
    }
}
