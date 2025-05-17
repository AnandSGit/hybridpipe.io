// NetChan protocol implementation for HybridPipe
//
// This module provides an implementation of the HybridPipe interface for NetChan.
// NetChan allows Go-like channel operations over a network.

use async_trait::async_trait;
use log::{debug, error, info, warn};
use std::any::Any;
use std::collections::HashMap;
use std::net::{SocketAddr, TcpListener, TcpStream};
use std::sync::{Arc, Mutex, RwLock};
use std::time::Duration;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::{TcpListener as TokioTcpListener, TcpStream as TokioTcpStream};
use tokio::sync::mpsc;
use tokio::task;
use tokio::time;

use crate::core::registry::register_router_factory;
use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};
use crate::serialization::{deserialize, serialize};

/// Configuration for NetChan.
#[derive(Debug, Clone)]
pub struct NetChanConfig {
    /// Network address to listen on or connect to (e.g., "127.0.0.1:9000")
    pub address: String,
    /// Channel buffer size
    pub buffer_size: usize,
    /// Timeout for network operations in milliseconds
    pub timeout: u64,
    /// Reconnect interval in milliseconds
    pub reconnect_interval: u64,
    /// Maximum reconnect attempts (0 = infinite)
    pub max_reconnect_attempts: u32,
}

impl Default for NetChanConfig {
    fn default() -> Self {
        Self {
            address: "127.0.0.1:9000".to_string(),
            buffer_size: 10,
            timeout: 5000,
            reconnect_interval: 1000,
            max_reconnect_attempts: 5,
        }
    }
}

/// Message type for NetChan.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
enum MessageType {
    /// Send a value to a channel
    Send,
    /// Request a value from a channel
    Receive,
    /// Close a channel
    Close,
}

/// Message for NetChan.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
struct Message {
    /// Message type
    message_type: MessageType,
    /// Channel name
    channel: String,
    /// Message data (only used for Send)
    data: Option<Vec<u8>>,
}

/// Packet implements the HybridPipe interface for NetChan.
pub struct Packet {
    /// Configuration
    config: NetChanConfig,
    /// Server listener (if in server mode)
    listener: Mutex<Option<TcpListener>>,
    /// Client connection (if in client mode)
    connection: Mutex<Option<TcpStream>>,
    /// Mode (server or client)
    mode: Mutex<Option<String>>,
    /// Send channels
    send_channels: RwLock<HashMap<String, mpsc::Sender<Vec<u8>>>>,
    /// Receive channels
    receive_channels: RwLock<HashMap<String, mpsc::Receiver<Vec<u8>>>>,
    /// Callbacks for each pipe
    callbacks: RwLock<HashMap<String, Vec<Process>>>,
    /// Connected flag
    connected: Mutex<bool>,
    /// Shutdown channel
    shutdown_tx: Mutex<Option<mpsc::Sender<()>>>,
}

impl Packet {
    /// Create a new NetChan packet with the specified configuration.
    pub fn new(config: NetChanConfig) -> Self {
        Self {
            config,
            listener: Mutex::new(None),
            connection: Mutex::new(None),
            mode: Mutex::new(None),
            send_channels: RwLock::new(HashMap::new()),
            receive_channels: RwLock::new(HashMap::new()),
            callbacks: RwLock::new(HashMap::new()),
            connected: Mutex::new(false),
            shutdown_tx: Mutex::new(None),
        }
    }

    /// Create a new NetChan packet with default configuration.
    pub fn new_default() -> Self {
        Self::new(NetChanConfig::default())
    }

    /// Start the server.
    async fn start_server(&self) -> Result<()> {
        let address = self.config.address.clone();
        let listener = TcpListener::bind(&address)?;
        listener.set_nonblocking(true)?;

        // Store the listener
        let mut listener_guard = self.listener.lock().unwrap();
        *listener_guard = Some(listener);
        drop(listener_guard);

        // Set the mode
        let mut mode_guard = self.mode.lock().unwrap();
        *mode_guard = Some("server".to_string());
        drop(mode_guard);

        // Start the acceptor task
        let (shutdown_tx, mut shutdown_rx) = mpsc::channel::<()>(1);
        let config = self.config.clone();
        let callbacks = self.callbacks.clone();
        let send_channels = self.send_channels.clone();
        let receive_channels = self.receive_channels.clone();

        // Get the listener for the acceptor task
        let listener_guard = self.listener.lock().unwrap();
        let listener = listener_guard.as_ref().unwrap().try_clone()?;
        drop(listener_guard);

        // Convert to tokio listener
        let tokio_listener = TokioTcpListener::from_std(listener)?;

        task::spawn(async move {
            loop {
                // Check for shutdown signal
                if shutdown_rx.try_recv().is_ok() {
                    debug!("Acceptor shutting down");
                    break;
                }

                // Accept a connection
                match tokio_listener.accept().await {
                    Ok((stream, addr)) => {
                        debug!("Accepted connection from {}", addr);
                        let callbacks = callbacks.clone();
                        let send_channels = send_channels.clone();
                        let receive_channels = receive_channels.clone();
                        let config = config.clone();

                        // Handle the connection
                        task::spawn(async move {
                            if let Err(e) = handle_connection(
                                stream,
                                addr,
                                callbacks,
                                send_channels,
                                receive_channels,
                                config,
                            )
                            .await
                            {
                                error!("Error handling connection: {}", e);
                            }
                        });
                    }
                    Err(e) => {
                        if e.kind() == std::io::ErrorKind::WouldBlock {
                            // No connections to accept, sleep a bit
                            time::sleep(Duration::from_millis(100)).await;
                        } else {
                            error!("Error accepting connection: {}", e);
                            time::sleep(Duration::from_millis(1000)).await;
                        }
                    }
                }
            }
        });

        // Store the shutdown channel
        let mut shutdown_tx_guard = self.shutdown_tx.lock().unwrap();
        *shutdown_tx_guard = Some(shutdown_tx);
        drop(shutdown_tx_guard);

        info!("NetChan server started on {}", address);
        Ok(())
    }

    /// Connect to the server.
    async fn connect_to_server(&self) -> Result<()> {
        let address = self.config.address.clone();
        let stream = TcpStream::connect(&address)?;
        stream.set_nonblocking(true)?;

        // Store the connection
        let mut connection_guard = self.connection.lock().unwrap();
        *connection_guard = Some(stream);
        drop(connection_guard);

        // Set the mode
        let mut mode_guard = self.mode.lock().unwrap();
        *mode_guard = Some("client".to_string());
        drop(mode_guard);

        // Start the reader task
        let (shutdown_tx, mut shutdown_rx) = mpsc::channel::<()>(1);
        let config = self.config.clone();
        let callbacks = self.callbacks.clone();
        let send_channels = self.send_channels.clone();
        let receive_channels = self.receive_channels.clone();

        // Get the connection for the reader task
        let connection_guard = self.connection.lock().unwrap();
        let connection = connection_guard.as_ref().unwrap().try_clone()?;
        drop(connection_guard);

        // Convert to tokio stream
        let tokio_stream = TokioTcpStream::from_std(connection)?;
        let addr = tokio_stream.peer_addr()?;

        task::spawn(async move {
            if let Err(e) = handle_connection(
                tokio_stream,
                addr,
                callbacks,
                send_channels,
                receive_channels,
                config,
            )
            .await
            {
                error!("Error handling connection: {}", e);
            }
        });

        // Store the shutdown channel
        let mut shutdown_tx_guard = self.shutdown_tx.lock().unwrap();
        *shutdown_tx_guard = Some(shutdown_tx);
        drop(shutdown_tx_guard);

        info!("Connected to NetChan server at {}", address);
        Ok(())
    }
}

/// Handle a connection.
async fn handle_connection(
    mut stream: TokioTcpStream,
    addr: SocketAddr,
    callbacks: Arc<RwLock<HashMap<String, Vec<Process>>>>,
    send_channels: Arc<RwLock<HashMap<String, mpsc::Sender<Vec<u8>>>>>,
    receive_channels: Arc<RwLock<HashMap<String, mpsc::Receiver<Vec<u8>>>>>,
    config: NetChanConfig,
) -> Result<()> {
    let mut buffer = vec![0u8; 4096];

    loop {
        // Read from the stream
        let n = match stream.read(&mut buffer).await {
            Ok(0) => {
                // Connection closed
                debug!("Connection closed by peer");
                return Ok(());
            }
            Ok(n) => n,
            Err(e) => {
                if e.kind() == std::io::ErrorKind::WouldBlock {
                    // No data to read, sleep a bit
                    time::sleep(Duration::from_millis(100)).await;
                    continue;
                } else {
                    return Err(Error::IoError(e.to_string()));
                }
            }
        };

        // Deserialize the message
        let message: Message = match serde_json::from_slice(&buffer[..n]) {
            Ok(m) => m,
            Err(e) => {
                error!("Failed to deserialize message: {}", e);
                continue;
            }
        };

        // Process the message
        match message.message_type {
            MessageType::Send => {
                // Get the data
                let data = match message.data {
                    Some(d) => d,
                    None => {
                        error!("Send message has no data");
                        continue;
                    }
                };

                // Process the message
                let callbacks_guard = callbacks.read().unwrap();
                if let Some(callbacks) = callbacks_guard.get(&message.channel) {
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
            }
            MessageType::Receive => {
                // Get the send channel
                let send_channels_guard = send_channels.read().unwrap();
                if let Some(sender) = send_channels_guard.get(&message.channel) {
                    // Try to get a value from the channel
                    let sender = sender.clone();
                    drop(send_channels_guard);

                    // Create a response message
                    let response = Message {
                        message_type: MessageType::Send,
                        channel: message.channel,
                        data: Some(vec![]), // Empty data for now
                    };

                    // Serialize the response
                    let response_data = match serde_json::to_vec(&response) {
                        Ok(d) => d,
                        Err(e) => {
                            error!("Failed to serialize response: {}", e);
                            continue;
                        }
                    };

                    // Send the response
                    if let Err(e) = stream.write_all(&response_data).await {
                        error!("Failed to send response: {}", e);
                    }
                }
            }
            MessageType::Close => {
                // Remove the callbacks for the channel
                let mut callbacks_guard = callbacks.write().unwrap();
                callbacks_guard.remove(&message.channel);
                drop(callbacks_guard);

                // Remove the send channel
                let mut send_channels_guard = send_channels.write().unwrap();
                send_channels_guard.remove(&message.channel);
                drop(send_channels_guard);

                // Remove the receive channel
                let mut receive_channels_guard = receive_channels.write().unwrap();
                receive_channels_guard.remove(&message.channel);
                drop(receive_channels_guard);
            }
        }
    }
}

#[async_trait]
impl HybridPipe for Packet {
    async fn connect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if *connected {
            return Err(Error::AlreadyConnected);
        }

        // Try to start as a server first
        match self.start_server().await {
            Ok(()) => {
                *connected = true;
                return Ok(());
            }
            Err(e) => {
                debug!("Failed to start as server: {}", e);
                debug!("Trying to connect as client...");
            }
        }

        // Try to connect as a client
        match self.connect_to_server().await {
            Ok(()) => {
                *connected = true;
                Ok(())
            }
            Err(e) => {
                error!("Failed to connect as client: {}", e);
                Err(e)
            }
        }
    }

    async fn disconnect(&self) -> Result<()> {
        let mut connected = self.connected.lock().unwrap();
        if !*connected {
            return Err(Error::NotConnected);
        }

        // Stop the acceptor/reader task
        let mut shutdown_tx_guard = self.shutdown_tx.lock().unwrap();
        if let Some(shutdown_tx) = shutdown_tx_guard.take() {
            if let Err(e) = shutdown_tx.send(()).await {
                warn!("Failed to send shutdown signal: {}", e);
            }
        }
        drop(shutdown_tx_guard);

        // Close the listener
        let mut listener_guard = self.listener.lock().unwrap();
        *listener_guard = None;
        drop(listener_guard);

        // Close the connection
        let mut connection_guard = self.connection.lock().unwrap();
        *connection_guard = None;
        drop(connection_guard);

        // Clear the mode
        let mut mode_guard = self.mode.lock().unwrap();
        *mode_guard = None;
        drop(mode_guard);

        // Clear the channels
        let mut send_channels_guard = self.send_channels.write().unwrap();
        send_channels_guard.clear();
        drop(send_channels_guard);

        let mut receive_channels_guard = self.receive_channels.write().unwrap();
        receive_channels_guard.clear();
        drop(receive_channels_guard);

        // Clear the callbacks
        let mut callbacks_guard = self.callbacks.write().unwrap();
        callbacks_guard.clear();
        drop(callbacks_guard);

        *connected = false;
        info!("Disconnected from NetChan");
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

        // Create a message
        let message = Message {
            message_type: MessageType::Send,
            channel: pipe.to_string(),
            data: Some(serialized),
        };

        // Serialize the message
        let message_data = serde_json::to_vec(&message)?;

        // Get the mode
        let mode_guard = self.mode.lock().unwrap();
        let mode = mode_guard.as_ref().ok_or_else(|| Error::NotConnected)?;
        drop(mode_guard);

        if mode == "client" {
            // Get the connection
            let connection_guard = self.connection.lock().unwrap();
            let connection = connection_guard
                .as_ref()
                .ok_or_else(|| Error::NotConnected)?
                .try_clone()?;
            drop(connection_guard);

            // Convert to tokio stream
            let mut stream = TokioTcpStream::from_std(connection)?;

            // Send the message
            stream.write_all(&message_data).await?;
        } else {
            // Server mode: not implemented yet
            return Err(Error::Other(
                "Server mode dispatch not implemented yet".to_string(),
            ));
        }

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

        // Remove the callbacks for the pipe
        let mut callbacks = self.callbacks.write().unwrap();
        callbacks.remove(pipe);

        // Create a close message
        let message = Message {
            message_type: MessageType::Close,
            channel: pipe.to_string(),
            data: None,
        };

        // Serialize the message
        let message_data = serde_json::to_vec(&message)?;

        // Get the mode
        let mode_guard = self.mode.lock().unwrap();
        let mode = mode_guard.as_ref().ok_or_else(|| Error::NotConnected)?;
        drop(mode_guard);

        if mode == "client" {
            // Get the connection
            let connection_guard = self.connection.lock().unwrap();
            let connection = connection_guard
                .as_ref()
                .ok_or_else(|| Error::NotConnected)?
                .try_clone()?;
            drop(connection_guard);

            // Convert to tokio stream
            let mut stream = TokioTcpStream::from_std(connection)?;

            // Send the message
            stream.write_all(&message_data).await?;
        } else {
            // Server mode: not implemented yet
            return Err(Error::Other(
                "Server mode unsubscribe not implemented yet".to_string(),
            ));
        }

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

/// Register the NetChan protocol with the HybridPipe system.
pub fn register() {
    register_router_factory(BrokerType::NETCHAN, Box::new(|| {
        Arc::new(Packet::new_default())
    }));
}

/// Channel is a network channel.
pub struct Channel {
    /// Channel name
    name: String,
    /// Packet
    packet: Arc<Packet>,
    /// Send channel
    send_channel: mpsc::Sender<Vec<u8>>,
    /// Receive channel
    receive_channel: mpsc::Receiver<Vec<u8>>,
}

impl Channel {
    /// Create a new channel.
    pub fn new(name: &str, packet: Arc<Packet>) -> Self {
        let (send_tx, send_rx) = mpsc::channel(packet.config.buffer_size);
        let (recv_tx, recv_rx) = mpsc::channel(packet.config.buffer_size);

        // Store the channels
        let mut send_channels = packet.send_channels.write().unwrap();
        send_channels.insert(name.to_string(), send_tx.clone());
        drop(send_channels);

        let mut receive_channels = packet.receive_channels.write().unwrap();
        receive_channels.insert(name.to_string(), recv_rx.clone());
        drop(receive_channels);

        Self {
            name: name.to_string(),
            packet,
            send_channel: send_tx,
            receive_channel: recv_rx,
        }
    }

    /// Send a value to the channel.
    pub async fn send(&self, value: Box<dyn Any + Send + Sync>) -> Result<()> {
        self.packet.dispatch(&self.name, value).await
    }

    /// Receive a value from the channel.
    pub async fn receive(&self) -> Result<Box<dyn Any + Send + Sync>> {
        // Create a receive message
        let message = Message {
            message_type: MessageType::Receive,
            channel: self.name.clone(),
            data: None,
        };

        // Serialize the message
        let message_data = serde_json::to_vec(&message)?;

        // Get the mode
        let mode_guard = self.packet.mode.lock().unwrap();
        let mode = mode_guard
            .as_ref()
            .ok_or_else(|| Error::NotConnected)?
            .clone();
        drop(mode_guard);

        if mode == "client" {
            // Get the connection
            let connection_guard = self.packet.connection.lock().unwrap();
            let connection = connection_guard
                .as_ref()
                .ok_or_else(|| Error::NotConnected)?
                .try_clone()?;
            drop(connection_guard);

            // Convert to tokio stream
            let mut stream = TokioTcpStream::from_std(connection)?;

            // Send the message
            stream.write_all(&message_data).await?;

            // Wait for a response
            let mut buffer = vec![0u8; 4096];
            let n = stream.read(&mut buffer).await?;

            // Deserialize the response
            let response: Message = serde_json::from_slice(&buffer[..n])?;

            // Check the response
            if response.message_type != MessageType::Send || response.channel != self.name {
                return Err(Error::Other("Invalid response".to_string()));
            }

            // Get the data
            let data = response
                .data
                .ok_or_else(|| Error::Other("Response has no data".to_string()))?;

            // Deserialize the data
            let value = deserialize(&data)?;
            Ok(value)
        } else {
            // Server mode: not implemented yet
            Err(Error::Other(
                "Server mode receive not implemented yet".to_string(),
            ))
        }
    }

    /// Close the channel.
    pub async fn close(&self) -> Result<()> {
        self.packet.unsubscribe(&self.name).await
    }
}

/// NetChan is a network channel manager.
pub struct NetChan {
    /// Packet
    packet: Arc<Packet>,
}

impl NetChan {
    /// Create a new NetChan with the specified configuration.
    pub async fn new(config: NetChanConfig) -> Result<Self> {
        let packet = Arc::new(Packet::new(config));
        packet.connect().await?;
        Ok(Self { packet })
    }

    /// Create a new NetChan with default configuration.
    pub async fn new_default() -> Result<Self> {
        Self::new(NetChanConfig::default()).await
    }

    /// Create a new channel.
    pub fn chan(&self, name: &str) -> Channel {
        Channel::new(name, self.packet.clone())
    }

    /// Close the NetChan.
    pub async fn close(&self) -> Result<()> {
        self.packet.close().await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_netchan_connect_disconnect() {
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
