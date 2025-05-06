// Protocol Buffers serialization support for HybridPipe
//
// This module provides support for Protocol Buffers serialization in HybridPipe.

use bytes::{Bytes, BytesMut};
use prost::Message;
use std::any::Any;
use std::collections::HashMap;
use std::sync::RwLock;

use crate::core::{Error, Result};

// Global registry for Protocol Buffers message types
lazy_static::lazy_static! {
    static ref PROTOBUF_REGISTRY: RwLock<HashMap<String, Box<dyn Fn() -> Box<dyn Any + Send + Sync> + Send + Sync>>> =
        RwLock::new(HashMap::new());
}

/// Register a Protocol Buffers message type with the registry.
pub fn register_protobuf_type<T>(type_name: &str)
where
    T: Message + Default + Clone + Send + Sync + 'static,
{
    let mut registry = PROTOBUF_REGISTRY.write().unwrap();
    registry.insert(
        type_name.to_string(),
        Box::new(move || Box::new(T::default())),
    );
}

/// Create a new instance of a Protocol Buffers message by type name.
pub fn create_message(type_name: &str) -> Result<Box<dyn Any + Send + Sync>> {
    let registry = PROTOBUF_REGISTRY.read().unwrap();
    if let Some(factory) = registry.get(type_name) {
        Ok(factory())
    } else {
        Err(Error::SerializationError(format!(
            "Protocol Buffers message type not registered: {}",
            type_name
        )))
    }
}

/// Encode a Protocol Buffers message.
pub fn encode_protobuf<T: Message + Send + Sync>(message: &T) -> Result<Bytes> {
    let mut buf = BytesMut::with_capacity(message.encoded_len());
    message.encode(&mut buf).map_err(|e| {
        Error::SerializationError(format!("Protocol Buffers encoding error: {}", e))
    })?;
    Ok(buf.freeze())
}

/// Decode a Protocol Buffers message.
pub fn decode_protobuf<T: Message + Default>(data: &[u8]) -> Result<T> {
    T::decode(data)
        .map_err(|e| Error::DeserializationError(format!("Protocol Buffers decoding error: {}", e)))
}

/// Encode a Protocol Buffers message with dynamic type.
pub fn encode_protobuf_dynamic(_message: &dyn Any, _type_name: &str) -> Result<Bytes> {
    // For now, we'll just return an error since we can't directly call encode on a trait object
    // This is a simplified implementation
    Err(Error::SerializationError(
        "Dynamic protobuf encoding not implemented".to_string(),
    ))
}

/// Decode a Protocol Buffers message with dynamic type.
pub fn decode_protobuf_dynamic(
    _data: &[u8],
    _type_name: &str,
) -> Result<Box<dyn Any + Send + Sync>> {
    // For now, we'll just return an error since we can't directly call merge on a trait object
    // This is a simplified implementation
    Err(Error::DeserializationError(
        "Dynamic protobuf decoding not implemented".to_string(),
    ))
}
