// Serialization module for HybridPipe
//
// This module provides functionality for encoding and decoding data in various formats
// for transmission through the HybridPipe messaging system.

pub mod protobuf;

#[cfg(test)]
mod tests;

use bytes::{Bytes, BytesMut};
use flate2::read::GzDecoder;
use flate2::write::GzEncoder;
use flate2::Compression;
use serde::{Deserialize, Serialize};
use std::io::{Read, Write};

use crate::core::{Error, Result};

/// SerializationFormat represents the format used for serialization.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SerializationFormat {
    /// Bincode is Rust's binary serialization format (similar to Go's gob)
    Bincode,
    /// JSON is a text-based, human-readable format
    Json,
    /// Protocol Buffers is a binary format with schema
    Protobuf,
    /// MessagePack is a binary JSON alternative
    MessagePack,
}

/// SerializationOptions configures the serialization behavior.
#[derive(Debug, Clone)]
pub struct SerializationOptions {
    /// Format specifies the serialization format to use
    pub format: SerializationFormat,
    /// Compression enables compression for large messages
    pub compression: bool,
    /// CompressionLevel specifies the compression level (0-9, where 0 is no compression and 9 is maximum compression)
    pub compression_level: u32,
    /// CrossLanguageCompatible enables cross-language compatible serialization
    pub cross_language_compatible: bool,
    /// ProtobufMessageType specifies the Protocol Buffer message type name
    pub protobuf_message_type: Option<String>,
}

impl Default for SerializationOptions {
    fn default() -> Self {
        Self {
            format: SerializationFormat::Bincode,
            compression: false,
            compression_level: 6,
            cross_language_compatible: false,
            protobuf_message_type: None,
        }
    }
}

/// Header for serialized messages
#[derive(Debug, Clone, Copy)]
struct MessageHeader {
    /// Magic number to identify HybridPipe messages
    magic: [u8; 2],
    /// Version of the message format
    version: u8,
    /// Format used for serialization
    format: u8,
    /// Flags for additional options (e.g., compression)
    flags: u8,
    /// Reserved for future use
    reserved: [u8; 3],
    /// Length of the payload
    payload_length: u32,
    /// Checksum of the payload
    checksum: u8,
}

const HEADER_SIZE: usize = 12;
const MAGIC: [u8; 2] = [b'H', b'P'];
const VERSION: u8 = 1;
const COMPRESSION_FLAG: u8 = 0x01;

/// Encode data using the default serialization options.
pub fn encode<T: Serialize + 'static>(data: &T) -> Result<Bytes> {
    encode_with_options(data, &SerializationOptions::default())
}

/// Encode data using the specified serialization options.
pub fn encode_with_options<T: Serialize + 'static>(
    data: &T,
    options: &SerializationOptions,
) -> Result<Bytes> {
    // Serialize the data based on the format
    let mut payload = match options.format {
        SerializationFormat::Bincode => bincode::serialize(data)
            .map_err(|e| Error::SerializationError(format!("bincode error: {}", e)))?,
        SerializationFormat::Json => serde_json::to_vec(data)
            .map_err(|e| Error::SerializationError(format!("json error: {}", e)))?,
        SerializationFormat::MessagePack => rmp_serde::to_vec(data)
            .map_err(|e| Error::SerializationError(format!("messagepack error: {}", e)))?,
        SerializationFormat::Protobuf => {
            if let Some(type_name) = &options.protobuf_message_type {
                // Convert the result to Vec<u8> to match other branches
                let bytes = protobuf::encode_protobuf_dynamic(data, type_name)
                    .map_err(|e| Error::SerializationError(format!("protobuf error: {}", e)))?;
                bytes.to_vec()
            } else {
                return Err(Error::SerializationError(
                    "protobuf serialization requires a message type".to_string(),
                ));
            }
        }
    };

    // Compress the payload if requested
    if options.compression {
        let mut encoder = GzEncoder::new(Vec::new(), Compression::new(options.compression_level));
        encoder
            .write_all(&payload)
            .map_err(|e| Error::SerializationError(format!("compression error: {}", e)))?;
        payload = encoder
            .finish()
            .map_err(|e| Error::SerializationError(format!("compression finish error: {}", e)))?;
    }

    // Create the header
    let mut header = [0u8; HEADER_SIZE];

    // Magic number
    header[0] = MAGIC[0];
    header[1] = MAGIC[1];

    // Version
    header[2] = VERSION;

    // Format
    header[3] = match options.format {
        SerializationFormat::Bincode => 0,
        SerializationFormat::Json => 1,
        SerializationFormat::Protobuf => 2,
        SerializationFormat::MessagePack => 3,
    };

    // Flags
    let mut flags = 0u8;
    if options.compression {
        flags |= COMPRESSION_FLAG;
    }
    header[4] = flags;

    // Reserved
    // header[5..8] remains zeros

    // Payload length
    let payload_len = payload.len() as u32;
    header[8] = (payload_len >> 24) as u8;
    header[9] = (payload_len >> 16) as u8;
    header[10] = (payload_len >> 8) as u8;
    header[11] = payload_len as u8;

    // Combine header and payload
    let mut result = BytesMut::with_capacity(HEADER_SIZE + payload.len());
    result.extend_from_slice(&header);
    result.extend_from_slice(&payload);

    Ok(result.freeze())
}

/// Decode data using automatic format detection.
pub fn decode<T: for<'de> Deserialize<'de> + 'static>(data: &[u8]) -> Result<T> {
    decode_with_options(data, &SerializationOptions::default())
}

/// Decode data using the specified serialization options.
pub fn decode_with_options<T: for<'de> Deserialize<'de> + 'static>(
    data: &[u8],
    options: &SerializationOptions,
) -> Result<T> {
    if data.len() < HEADER_SIZE {
        return Err(Error::DeserializationError(
            "data too short for header".to_string(),
        ));
    }

    // Parse the header
    let header = &data[0..HEADER_SIZE];

    // Check magic number
    if header[0] != MAGIC[0] || header[1] != MAGIC[1] {
        return Err(Error::DeserializationError(
            "invalid magic number".to_string(),
        ));
    }

    // Check version
    if header[2] != VERSION {
        return Err(Error::DeserializationError(format!(
            "unsupported version: {}",
            header[2]
        )));
    }

    // Get format
    let format = match header[3] {
        0 => SerializationFormat::Bincode,
        1 => SerializationFormat::Json,
        2 => SerializationFormat::Protobuf,
        3 => SerializationFormat::MessagePack,
        _ => {
            return Err(Error::DeserializationError(format!(
                "unsupported format: {}",
                header[3]
            )));
        }
    };

    // Get flags
    let flags = header[4];
    let compressed = (flags & COMPRESSION_FLAG) != 0;

    // Get payload length
    let payload_len = ((header[8] as u32) << 24)
        | ((header[9] as u32) << 16)
        | ((header[10] as u32) << 8)
        | (header[11] as u32);

    // Extract payload
    let payload = &data[HEADER_SIZE..];
    if payload.len() != payload_len as usize {
        return Err(Error::DeserializationError(format!(
            "payload length mismatch: expected {}, got {}",
            payload_len,
            payload.len()
        )));
    }

    // Decompress if necessary
    let payload = if compressed {
        let mut decoder = GzDecoder::new(payload);
        let mut decompressed = Vec::new();
        decoder
            .read_to_end(&mut decompressed)
            .map_err(|e| Error::DeserializationError(format!("decompression error: {}", e)))?;
        decompressed
    } else {
        payload.to_vec()
    };

    // Deserialize based on format
    match format {
        SerializationFormat::Bincode => bincode::deserialize(&payload)
            .map_err(|e| Error::DeserializationError(format!("bincode error: {}", e))),
        SerializationFormat::Json => serde_json::from_slice(&payload)
            .map_err(|e| Error::DeserializationError(format!("json error: {}", e))),
        SerializationFormat::MessagePack => rmp_serde::from_slice(&payload)
            .map_err(|e| Error::DeserializationError(format!("messagepack error: {}", e))),
        SerializationFormat::Protobuf => {
            if let Some(type_name) = &options.protobuf_message_type {
                let decoded = protobuf::decode_protobuf_dynamic(&payload, type_name)?;
                // We need to convert the Box<dyn Any> to T
                match decoded.downcast::<T>() {
                    Ok(t) => Ok(*t),
                    Err(_) => Err(Error::DeserializationError(format!(
                        "failed to downcast protobuf message to target type: {}",
                        std::any::type_name::<T>()
                    ))),
                }
            } else {
                Err(Error::DeserializationError(
                    "protobuf deserialization requires a message type".to_string(),
                ))
            }
        }
    }
}

/// Register a type for serialization.
pub fn register_type<T: 'static>() {
    // This is a no-op in Rust since type information is handled by the compiler
    // and serde. It's included for API compatibility with the Go version.
}
