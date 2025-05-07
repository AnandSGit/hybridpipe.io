// Mock implementation of TCP client for testing
//
// This module provides a mock implementation of the TCP client for testing.

use std::collections::VecDeque;
use std::net::SocketAddr;
use std::sync::Arc;
use std::sync::Mutex;

use super::common::{ConnectionState, MessageStore};

/// Mock TCP listener
#[derive(Debug)]
pub struct MockTcpListener {
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
    /// Incoming connections
    pub incoming: Arc<Mutex<VecDeque<MockTcpStream>>>,
}

impl MockTcpListener {
    /// Create a new mock TCP listener
    pub fn new() -> Self {
        Self {
            connection_state: Arc::new(ConnectionState::new()),
            incoming: Arc::new(Mutex::new(VecDeque::new())),
        }
    }

    /// Accept a connection
    pub async fn accept(&self) -> Result<(MockTcpStream, SocketAddr), String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        let mut incoming = self.incoming.lock().unwrap();
        if let Some(stream) = incoming.pop_front() {
            Ok((stream, "127.0.0.1:12345".parse().unwrap()))
        } else {
            Err("No incoming connections".to_string())
        }
    }

    /// Add a mock connection
    pub fn add_connection(&self, stream: MockTcpStream) {
        let mut incoming = self.incoming.lock().unwrap();
        incoming.push_back(stream);
    }
}

/// Mock TCP stream
#[derive(Debug, Clone)]
pub struct MockTcpStream {
    /// Message store for sent messages
    pub message_store: Arc<MessageStore>,
    /// Connection state
    pub connection_state: Arc<ConnectionState>,
    /// Read buffer
    pub read_buffer: Arc<Mutex<VecDeque<u8>>>,
    /// Write buffer
    pub write_buffer: Arc<Mutex<VecDeque<u8>>>,
}

impl MockTcpStream {
    /// Create a new mock TCP stream
    pub fn new() -> Self {
        Self {
            message_store: Arc::new(MessageStore::new()),
            connection_state: Arc::new(ConnectionState::new()),
            read_buffer: Arc::new(Mutex::new(VecDeque::new())),
            write_buffer: Arc::new(Mutex::new(VecDeque::new())),
        }
    }

    /// Connect to a remote address
    pub async fn connect(_addr: &str) -> Result<Self, String> {
        let stream = Self::new();
        stream.connection_state.set_connected(true);
        Ok(stream)
    }

    /// Read data from the stream
    pub async fn read(&self, buf: &mut [u8]) -> Result<usize, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        let mut read_buffer = self.read_buffer.lock().unwrap();
        let mut bytes_read = 0;

        for i in 0..buf.len() {
            if let Some(byte) = read_buffer.pop_front() {
                buf[i] = byte;
                bytes_read += 1;
            } else {
                break;
            }
        }

        Ok(bytes_read)
    }

    /// Write data to the stream
    pub async fn write(&self, buf: &[u8]) -> Result<usize, String> {
        if !self.connection_state.is_connected() {
            return Err("Not connected".to_string());
        }

        let mut write_buffer = self.write_buffer.lock().unwrap();
        for &byte in buf {
            write_buffer.push_back(byte);
        }

        Ok(buf.len())
    }

    /// Close the stream
    pub async fn close(&self) -> Result<(), String> {
        self.connection_state.set_connected(false);
        Ok(())
    }

    /// Check if the stream is connected
    pub fn is_connected(&self) -> bool {
        self.connection_state.is_connected()
    }

    /// Add data to the read buffer
    pub fn add_read_data(&self, data: &[u8]) {
        let mut read_buffer = self.read_buffer.lock().unwrap();
        for &byte in data {
            read_buffer.push_back(byte);
        }
    }

    /// Get data from the write buffer
    pub fn get_write_data(&self) -> Vec<u8> {
        let mut write_buffer = self.write_buffer.lock().unwrap();
        let mut data = Vec::new();
        while let Some(byte) = write_buffer.pop_front() {
            data.push(byte);
        }
        data
    }
}

/// Bind a TCP listener to an address
pub async fn bind(_addr: &str) -> Result<MockTcpListener, String> {
    let listener = MockTcpListener::new();
    listener.connection_state.set_connected(true);
    Ok(listener)
}

/// Connect to a TCP server
pub async fn connect(addr: &str) -> Result<MockTcpStream, String> {
    MockTcpStream::connect(addr).await
}
