// TCP protocol implementation for HybridPipe
//
// This module provides an implementation of the HybridPipe interface for direct TCP communication.

use async_trait::async_trait;
use futures::sink::SinkExt;
use futures::stream::StreamExt;
use log::{debug, error, info};
use std::any::Any;
use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::{Arc, Mutex};
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::{TcpListener, TcpStream};
use tokio::sync::{mpsc, RwLock};
use tokio::task::JoinHandle;
use tokio_util::codec::{Decoder, Framed, LengthDelimitedCodec};

use crate::core::config::{get_config, TCPConfig};
use crate::core::registry::register_router_factory;
use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};

/// Packet implements the HybridPipe interface for direct TCP communication.
pub struct Packet {
    /// Client connections maps pipe names to their TCP streams
    client_connections: RwLock<HashMap<String, Framed<TcpStream, LengthDelimitedCodec>>>,
    /// Server listener is the TCP listener for server mode
    server_listener: Mutex<Option<TcpListener>>,
    /// Server task is the task handling incoming connections in server mode
    server_task: Mutex<Option<JoinHandle<()>>>,
    /// Subscribers maps pipe names to their callback functions
    subscribers: RwLock<HashMap<String, Process>>,
    /// Subscriber tasks maps pipe names to their subscriber task handles
    subscriber_tasks: RwLock<HashMap<String, JoinHandle<()>>>,
    /// Config holds the TCP configuration
    config: TCPConfig,
    /// Connected flag indicates if the packet is connected
    connected: Mutex<bool>,
    /// Server mode flag indicates if the packet is in server mode
    server_mode: bool,
}

/// TcpPacket is a version of Packet that uses Arc for all fields.
/// This is used for spawning tasks that need to access the packet's fields.
struct TcpPacket {
    /// Client connections maps pipe names to their TCP streams
    client_connections: Arc<RwLock<HashMap<String, Framed<TcpStream, LengthDelimitedCodec>>>>,
    /// Server listener is the TCP listener for server mode
    server_listener: Arc<Mutex<Option<TcpListener>>>,
    /// Server task is the task handling incoming connections in server mode
    server_task: Arc<Mutex<Option<JoinHandle<()>>>>,
    /// Subscribers maps pipe names to their callback functions
    subscribers: Arc<RwLock<HashMap<String, Process>>>,
    /// Subscriber tasks maps pipe names to their subscriber task handles
    subscriber_tasks: Arc<RwLock<HashMap<String, JoinHandle<()>>>>,
    /// Connected flag indicates if the packet is connected
    connected: Arc<Mutex<bool>>,
    /// Server mode flag indicates if the packet is in server mode
    server_mode: bool,
}

impl Packet {
    /// Create a new TCP packet.
    pub fn new() -> Self {
        // Get the TCP configuration
        let config = match get_config() {
            Ok(config) => config.tcp.clone(),
            Err(_) => TCPConfig {
                tcp_host: "localhost".to_string(),
            },
        };

        // Determine if we're in server mode
        let server_mode = config.tcp_host == "0.0.0.0" || config.tcp_host == "::";

        Self {
            client_connections: RwLock::new(HashMap::new()),
            server_listener: Mutex::new(None),
            server_task: Mutex::new(None),
            subscribers: RwLock::new(HashMap::new()),
            subscriber_tasks: RwLock::new(HashMap::new()),
            config,
            connected: Mutex::new(false),
            server_mode,
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

        if self.server_mode {
            // Server mode: bind to a port and listen for connections
            let addr = format!("{}:{}", self.config.tcp_host, 9000)
                .parse::<SocketAddr>()
                .map_err(|e| Error::ConnectionError(format!("Invalid TCP address: {}", e)))?;

            let listener = TcpListener::bind(addr).await.map_err(|e| {
                Error::ConnectionError(format!("Failed to bind TCP listener: {}", e))
            })?;

            info!("TCP server listening on {}", addr);

            // Create a channel for new connections
            let (tx, mut rx) =
                mpsc::channel::<(String, Framed<TcpStream, LengthDelimitedCodec>)>(100);

            // Create a task to handle incoming connections
            // We can't clone RwLock directly, but we can use a reference to it
            let subscribers = &self.subscribers;

            // Create a separate task for the listener
            let listener_clone = listener;
            let server_task = tokio::spawn(async move {
                loop {
                    match listener_clone.accept().await {
                        Ok((stream, addr)) => {
                            info!("New TCP connection from {}", addr);

                            // Set up framing
                            let framed = LengthDelimitedCodec::new().framed(stream);

                            // Create a unique pipe name for this connection
                            let pipe = format!("tcp.{}", addr);

                            // Send the connection to the main task
                            if let Err(e) = tx.send((pipe.clone(), framed)).await {
                                error!("Failed to send new connection: {}", e);
                            }
                        }
                        Err(e) => {
                            error!("Failed to accept TCP connection: {}", e);
                        }
                    }
                }
            });

            // Store the server task
            let mut task_guard = self.server_task.lock().unwrap();
            *task_guard = Some(server_task);
            drop(task_guard);

            // Create a task to process incoming connections
            // We need to create a new TcpPacket to avoid capturing self
            let tcp_packet = Arc::new(TcpPacket {
                server_mode: self.server_mode,
                connected: Arc::new(Mutex::new(*self.connected.lock().unwrap())),
                subscribers: Arc::new(RwLock::new(HashMap::new())),
                client_connections: Arc::new(RwLock::new(HashMap::new())),
                server_listener: Arc::new(Mutex::new(None)),
                server_task: Arc::new(Mutex::new(None)),
                subscriber_tasks: Arc::new(RwLock::new(HashMap::new())),
            });

            tokio::spawn(async move {
                while let Some((pipe, framed)) = rx.recv().await {
                    // Store the connection
                    let mut connections = tcp_packet.client_connections.write().await;
                    connections.insert(pipe.clone(), framed);
                    drop(connections);

                    // Clone what we need for the new task
                    let pipe_clone = pipe.clone();
                    let tcp_packet_clone = Arc::clone(&tcp_packet);

                    // Spawn a new task to handle this connection
                    tokio::spawn(async move {
                        // Process messages from this connection
                        let addr = match pipe_clone.parse::<SocketAddr>() {
                            Ok(addr) => addr,
                            Err(e) => {
                                error!("Invalid address in pipe {}: {}", pipe_clone, e);
                                return;
                            }
                        };

                        // Create a new connection to the address
                        let addr = match pipe_clone.parse::<SocketAddr>() {
                            Ok(addr) => addr,
                            Err(e) => {
                                error!("Invalid address in pipe {}: {}", pipe_clone, e);
                                return;
                            }
                        };

                        let stream = match TcpStream::connect(addr).await {
                            Ok(stream) => stream,
                            Err(e) => {
                                error!("Failed to connect to {}: {}", addr, e);
                                return;
                            }
                        };

                        let mut framed = LengthDelimitedCodec::new().framed(stream);

                        // Process messages
                        while let Some(result) = framed.next().await {
                            match result {
                                Ok(bytes) => {
                                    // Process the message
                                    let subscribers = tcp_packet_clone.subscribers.read().await;
                                    if let Some(callback) = subscribers.get(&pipe_clone) {
                                        if let Err(e) = callback(bytes.to_vec()) {
                                            error!(
                                                "Error processing message from pipe {}: {}",
                                                pipe_clone, e
                                            );
                                        }
                                    }
                                }
                                Err(e) => {
                                    error!("Error reading from TCP connection: {}", e);
                                    break;
                                }
                            }
                        }

                        // Connection closed, remove it
                        let mut connections = tcp_packet_clone.client_connections.write().await;
                        connections.remove(&pipe_clone);
                        info!("TCP connection closed: {}", pipe_clone);
                    });
                }
            });
        }

        // Set connected flag
        {
            let mut connected = self.connected.lock().unwrap();
            *connected = true;
        }

        info!(
            "TCP protocol initialized in {} mode",
            if self.server_mode { "server" } else { "client" }
        );
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

        // Cancel the server task if in server mode
        if self.server_mode {
            let mut task_guard = self.server_task.lock().unwrap();
            if let Some(task) = task_guard.take() {
                task.abort();
            }

            // Clear the server listener
            let mut listener_guard = self.server_listener.lock().unwrap();
            *listener_guard = None;
        }

        // Clear the client connections
        let mut connections = self.client_connections.write().await;
        connections.clear();

        // Clear the subscribers
        let mut subscribers = self.subscribers.write().await;
        subscribers.clear();

        info!("TCP protocol disconnected");
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

        // Convert the data to a JSON string
        let json_string = serde_json::to_string(&format!("{:?}", data))
            .map_err(|e| Error::SerializationError(format!("Failed to encode message: {}", e)))?;

        // Convert the JSON string to bytes
        let bytes = json_string.into_bytes();

        // Get the connection for this pipe
        let mut connections = self.client_connections.write().await;

        if self.server_mode {
            // In server mode, we need to find the connection for this pipe
            if let Some(framed) = connections.get_mut(pipe) {
                framed
                    .send(bytes.into())
                    .await
                    .map_err(|e| Error::DispatchError(format!("Failed to send message: {}", e)))?;
            } else {
                return Err(Error::PipeNotFound(pipe.to_string()));
            }
        } else {
            // In client mode, we need to create a connection if it doesn't exist
            if !connections.contains_key(pipe) {
                // Parse the address from the pipe name
                let addr = pipe.parse::<SocketAddr>().map_err(|_| {
                    Error::InvalidConfiguration(format!("Invalid TCP address in pipe: {}", pipe))
                })?;

                // Connect to the server
                let stream = TcpStream::connect(addr).await.map_err(|e| {
                    Error::ConnectionError(format!("Failed to connect to TCP server: {}", e))
                })?;

                // Set up framing
                let framed = LengthDelimitedCodec::new().framed(stream);

                // Store the connection
                connections.insert(pipe.to_string(), framed);
            }

            // Send the message
            if let Some(framed) = connections.get_mut(pipe) {
                framed
                    .send(bytes.into())
                    .await
                    .map_err(|e| Error::DispatchError(format!("Failed to send message: {}", e)))?;
            }
        }

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

        // Store the callback
        let mut subscribers = self.subscribers.write().await;
        subscribers.insert(pipe.to_string(), callback);

        info!("Subscribed to TCP pipe: {}", pipe);
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

        // Remove the subscriber
        let mut subscribers = self.subscribers.write().await;
        subscribers.remove(pipe);

        info!("Unsubscribed from TCP pipe: {}", pipe);
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

/// Register the TCP protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::TCP, Box::new(|| Arc::new(Packet::new())));
}
