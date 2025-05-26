"""
Core serialization engine for HybridPipe.

This module provides the main serialization interface that maintains
wire format compatibility with the original Go/Rust implementations.
It supports multiple serialization formats and includes compression
and header management.
"""

import struct
import gzip
import json
from typing import Any, Dict, Optional, Type, Union
from dataclasses import dataclass
from abc import ABC, abstractmethod

from hybridpipe.core.types import SerializationFormat
from hybridpipe.core.errors import SerializationError

__all__ = [
    "SerializationOptions",
    "BaseSerializer",
    "encode",
    "decode",
    "encode_with_options",
    "decode_with_options",
    "register_type",
    "get_serializer",
]

# Constants matching Go implementation
HEADER_SIZE = 9
MAGIC_BYTES = b'HP'
VERSION = 1
COMPRESSION_FLAG = 0x80


@dataclass
class SerializationOptions:
    """
    Options for serialization operations.
    
    Attributes:
        format: Serialization format to use
        compression: Whether to enable compression
        compression_level: Compression level (0-9)
        compression_threshold: Minimum size to trigger compression
        cross_language_compatible: Use cross-language compatible formats only
        type_registry: Custom type registry for serialization
    """
    format: SerializationFormat = SerializationFormat.JSON
    compression: bool = False
    compression_level: int = 6
    compression_threshold: int = 64
    cross_language_compatible: bool = True
    type_registry: Optional[Dict[str, Type]] = None


class BaseSerializer(ABC):
    """Abstract base class for serializers."""
    
    @abstractmethod
    def serialize(self, data: Any) -> bytes:
        """Serialize data to bytes."""
        pass
    
    @abstractmethod
    def deserialize(self, data: bytes, target_type: Optional[Type] = None) -> Any:
        """Deserialize bytes to data."""
        pass
    
    @property
    @abstractmethod
    def format_id(self) -> SerializationFormat:
        """Get the format ID for this serializer."""
        pass


class JSONSerializer(BaseSerializer):
    """JSON serializer implementation."""
    
    def serialize(self, data: Any) -> bytes:
        """Serialize data to JSON bytes."""
        try:
            return json.dumps(data, separators=(',', ':'), ensure_ascii=False).encode('utf-8')
        except (TypeError, ValueError) as e:
            raise SerializationError(
                f"Failed to serialize data to JSON: {str(e)}",
                serialization_format="JSON",
                data_type=type(data).__name__,
                cause=e,
            )
    
    def deserialize(self, data: bytes, target_type: Optional[Type] = None) -> Any:
        """Deserialize JSON bytes to data."""
        try:
            return json.loads(data.decode('utf-8'))
        except (json.JSONDecodeError, UnicodeDecodeError) as e:
            raise SerializationError(
                f"Failed to deserialize JSON data: {str(e)}",
                serialization_format="JSON",
                cause=e,
            )
    
    @property
    def format_id(self) -> SerializationFormat:
        return SerializationFormat.JSON


class MessagePackSerializer(BaseSerializer):
    """MessagePack serializer implementation."""
    
    def __init__(self):
        try:
            import msgpack
            self._msgpack = msgpack
        except ImportError:
            raise SerializationError(
                "MessagePack library not available. Install with: pip install msgpack",
                serialization_format="MessagePack",
            )
    
    def serialize(self, data: Any) -> bytes:
        """Serialize data to MessagePack bytes."""
        try:
            return self._msgpack.packb(data, use_bin_type=True)
        except (TypeError, ValueError) as e:
            raise SerializationError(
                f"Failed to serialize data to MessagePack: {str(e)}",
                serialization_format="MessagePack",
                data_type=type(data).__name__,
                cause=e,
            )
    
    def deserialize(self, data: bytes, target_type: Optional[Type] = None) -> Any:
        """Deserialize MessagePack bytes to data."""
        try:
            return self._msgpack.unpackb(data, raw=False, strict_map_key=False)
        except (self._msgpack.exceptions.ExtraData, 
                self._msgpack.exceptions.UnpackException,
                ValueError) as e:
            raise SerializationError(
                f"Failed to deserialize MessagePack data: {str(e)}",
                serialization_format="MessagePack",
                cause=e,
            )
    
    @property
    def format_id(self) -> SerializationFormat:
        return SerializationFormat.MSGPACK


class ProtobufSerializer(BaseSerializer):
    """Protocol Buffers serializer implementation."""
    
    def __init__(self):
        try:
            from google.protobuf import message
            self._protobuf_message = message
        except ImportError:
            raise SerializationError(
                "Protobuf library not available. Install with: pip install protobuf",
                serialization_format="Protobuf",
            )
    
    def serialize(self, data: Any) -> bytes:
        """Serialize protobuf message to bytes."""
        if not isinstance(data, self._protobuf_message.Message):
            raise SerializationError(
                "Data must be a protobuf Message instance",
                serialization_format="Protobuf",
                data_type=type(data).__name__,
            )
        
        try:
            return data.SerializeToString()
        except Exception as e:
            raise SerializationError(
                f"Failed to serialize protobuf message: {str(e)}",
                serialization_format="Protobuf",
                data_type=type(data).__name__,
                cause=e,
            )
    
    def deserialize(self, data: bytes, target_type: Optional[Type] = None) -> Any:
        """Deserialize bytes to protobuf message."""
        if target_type is None:
            raise SerializationError(
                "target_type is required for protobuf deserialization",
                serialization_format="Protobuf",
            )
        
        if not issubclass(target_type, self._protobuf_message.Message):
            raise SerializationError(
                "target_type must be a protobuf Message class",
                serialization_format="Protobuf",
                data_type=target_type.__name__,
            )
        
        try:
            message = target_type()
            message.ParseFromString(data)
            return message
        except Exception as e:
            raise SerializationError(
                f"Failed to deserialize protobuf message: {str(e)}",
                serialization_format="Protobuf",
                data_type=target_type.__name__,
                cause=e,
            )
    
    @property
    def format_id(self) -> SerializationFormat:
        return SerializationFormat.PROTOBUF


class PickleSerializer(BaseSerializer):
    """Pickle serializer implementation (Python equivalent of Go's gob)."""
    
    def __init__(self):
        import pickle
        self._pickle = pickle
    
    def serialize(self, data: Any) -> bytes:
        """Serialize data to pickle bytes."""
        try:
            return self._pickle.dumps(data, protocol=self._pickle.HIGHEST_PROTOCOL)
        except (TypeError, self._pickle.PicklingError) as e:
            raise SerializationError(
                f"Failed to serialize data to pickle: {str(e)}",
                serialization_format="Pickle",
                data_type=type(data).__name__,
                cause=e,
            )
    
    def deserialize(self, data: bytes, target_type: Optional[Type] = None) -> Any:
        """Deserialize pickle bytes to data."""
        try:
            return self._pickle.loads(data)
        except (self._pickle.UnpicklingError, EOFError, ValueError) as e:
            raise SerializationError(
                f"Failed to deserialize pickle data: {str(e)}",
                serialization_format="Pickle",
                cause=e,
            )
    
    @property
    def format_id(self) -> SerializationFormat:
        return SerializationFormat.GOB  # Map to GOB for compatibility


# Global serializer registry
_serializers: Dict[SerializationFormat, BaseSerializer] = {}
_type_registry: Dict[str, Type] = {}


def _initialize_serializers() -> None:
    """Initialize the default serializers."""
    global _serializers
    if not _serializers:
        _serializers[SerializationFormat.JSON] = JSONSerializer()
        
        try:
            _serializers[SerializationFormat.MSGPACK] = MessagePackSerializer()
        except SerializationError:
            pass  # MessagePack not available
        
        try:
            _serializers[SerializationFormat.PROTOBUF] = ProtobufSerializer()
        except SerializationError:
            pass  # Protobuf not available
        
        _serializers[SerializationFormat.GOB] = PickleSerializer()


def get_serializer(format: SerializationFormat) -> BaseSerializer:
    """
    Get a serializer for the specified format.
    
    Args:
        format: The serialization format
    
    Returns:
        Serializer instance
    
    Raises:
        SerializationError: If format is not supported
    """
    _initialize_serializers()
    
    if format not in _serializers:
        raise SerializationError(
            f"Serialization format {format.name} is not supported",
            serialization_format=format.name,
        )
    
    return _serializers[format]


def register_type(name: str, type_class: Type) -> None:
    """
    Register a type for serialization.
    
    Args:
        name: Name to register the type under
        type_class: The type class to register
    """
    _type_registry[name] = type_class


def _calculate_checksum(data: bytes) -> int:
    """Calculate a simple checksum for the data."""
    return sum(data) & 0xFF


def _compress_data(data: bytes, level: int) -> bytes:
    """Compress data using gzip."""
    return gzip.compress(data, compresslevel=level)


def _decompress_data(data: bytes) -> bytes:
    """Decompress gzip data."""
    return gzip.decompress(data)


def encode_with_options(data: Any, options: SerializationOptions) -> bytes:
    """
    Encode data with the specified options.
    
    Args:
        data: Data to encode
        options: Serialization options
    
    Returns:
        Encoded bytes with HybridPipe header
    
    Raises:
        SerializationError: If encoding fails
    """
    # Get the appropriate serializer
    serializer = get_serializer(options.format)
    
    # Serialize the data
    payload = serializer.serialize(data)
    
    # Apply compression if enabled and beneficial
    format_byte = options.format.value
    if (options.compression and 
        len(payload) > options.compression_threshold):
        
        compressed = _compress_data(payload, options.compression_level)
        if len(compressed) < len(payload):
            payload = compressed
            format_byte |= COMPRESSION_FLAG
    
    # Create header (matching Go implementation)
    header = bytearray(HEADER_SIZE)
    header[0:2] = MAGIC_BYTES
    header[2] = VERSION
    header[3] = format_byte
    struct.pack_into('>I', header, 4, len(payload))  # Big-endian uint32
    header[8] = _calculate_checksum(payload)
    
    return bytes(header) + payload


def decode_with_options(
    data: bytes, 
    options: SerializationOptions,
    target_type: Optional[Type] = None,
) -> Any:
    """
    Decode data with the specified options.
    
    Args:
        data: Encoded data with HybridPipe header
        options: Serialization options
        target_type: Target type for deserialization (required for protobuf)
    
    Returns:
        Decoded data
    
    Raises:
        SerializationError: If decoding fails
    """
    # Validate header
    if len(data) < HEADER_SIZE:
        raise SerializationError("Data too short to contain header")
    
    if data[0:2] != MAGIC_BYTES:
        raise SerializationError("Invalid magic bytes in header")
    
    if data[2] != VERSION:
        raise SerializationError(f"Unsupported version: {data[2]}")
    
    # Extract format and compression flag
    format_byte = data[3]
    compressed = (format_byte & COMPRESSION_FLAG) != 0
    format_id = SerializationFormat(format_byte & ~COMPRESSION_FLAG)
    
    # Extract payload length and validate
    payload_length = struct.unpack('>I', data[4:8])[0]
    if len(data) - HEADER_SIZE != payload_length:
        raise SerializationError("Payload length mismatch")
    
    # Validate checksum
    payload = data[HEADER_SIZE:]
    expected_checksum = _calculate_checksum(payload)
    if data[8] != expected_checksum:
        raise SerializationError("Checksum mismatch")
    
    # Decompress if needed
    if compressed:
        try:
            payload = _decompress_data(payload)
        except Exception as e:
            raise SerializationError(f"Decompression failed: {str(e)}", cause=e)
    
    # Deserialize the payload
    serializer = get_serializer(format_id)
    return serializer.deserialize(payload, target_type)


def encode(data: Any, format: SerializationFormat = SerializationFormat.JSON) -> bytes:
    """
    Encode data using the specified format with default options.
    
    Args:
        data: Data to encode
        format: Serialization format to use
    
    Returns:
        Encoded bytes
    """
    options = SerializationOptions(format=format)
    return encode_with_options(data, options)


def decode(
    data: bytes, 
    target_type: Optional[Type] = None,
    format: Optional[SerializationFormat] = None,
) -> Any:
    """
    Decode data, automatically detecting format from header.
    
    Args:
        data: Encoded data
        target_type: Target type for deserialization
        format: Expected format (for validation, optional)
    
    Returns:
        Decoded data
    """
    options = SerializationOptions(format=format or SerializationFormat.JSON)
    return decode_with_options(data, options, target_type)


# Initialize serializers on module import
_initialize_serializers()
