"""
Core types and enumerations for HybridPipe.

This module defines the fundamental types used throughout the HybridPipe
messaging system, including broker types, serialization formats, and
callback function signatures.
"""

from enum import IntEnum, Enum
from typing import Any, Callable, Dict, Optional, Union
from dataclasses import dataclass
from datetime import datetime
import asyncio

__all__ = [
    "BrokerType",
    "SerializationFormat",
    "ConnectionState",
    "MessageCallback",
    "MessageMetadata",
    "ProtocolCapabilities",
]


class BrokerType(IntEnum):
    """Enumeration of supported broker types."""

    # Phase 1: Core protocols
    MOCK = 1
    TCP = 2
    REDIS = 3

    # Phase 2: Production messaging protocols
    KAFKA = 4
    RABBITMQ = 5
    MQTT = 6

    # Phase 3: Advanced protocols
    NATS = 7
    ZEROMQ = 8
    AMQP1 = 9

    # Phase 4: Specialized protocols (planned)
    NSQ = 10
    NETCHAN = 11


class SerializationFormat(IntEnum):
    """
    Enumeration of supported serialization formats.

    Values match the original Go implementation for wire compatibility.
    """
    GOB = 0      # Go's gob format (Python uses pickle as equivalent)
    JSON = 1     # Standard JSON encoding
    PROTOBUF = 2 # Protocol Buffers encoding
    MSGPACK = 3  # MessagePack encoding


class ConnectionState(Enum):
    """Connection state enumeration."""
    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    RECONNECTING = "reconnecting"
    FAILED = "failed"


@dataclass
class MessageMetadata:
    """
    Metadata associated with a message.

    Attributes:
        message_id: Unique identifier for the message
        timestamp: When the message was created/received
        pipe: The pipe/channel/topic the message was sent to
        broker_type: The type of broker that handled the message
        serialization_format: Format used to serialize the message
        size_bytes: Size of the serialized message in bytes
        headers: Additional headers/metadata
        retry_count: Number of times this message has been retried
        correlation_id: ID for correlating related messages
        reply_to: Pipe to send replies to (for request-response patterns)
    """
    message_id: str
    timestamp: datetime
    pipe: str
    broker_type: BrokerType
    serialization_format: SerializationFormat
    size_bytes: int
    headers: Dict[str, Any]
    retry_count: int = 0
    correlation_id: Optional[str] = None
    reply_to: Optional[str] = None


@dataclass(frozen=True)
class ProtocolCapabilities:
    """
    Capabilities supported by a protocol implementation.

    Attributes:
        supports_persistence: Whether messages can be persisted
        supports_transactions: Whether transactions are supported
        supports_clustering: Whether clustering/high availability is supported
        supports_compression: Whether message compression is supported
        supports_encryption: Whether encryption is supported
        supports_authentication: Whether authentication is supported
        supports_authorization: Whether authorization is supported
        supports_dead_letter: Whether dead letter queues are supported
        supports_message_ordering: Whether message ordering is guaranteed
        supports_exactly_once: Whether exactly-once delivery is supported
        max_message_size: Maximum message size in bytes (None = unlimited)
        max_pipe_length: Maximum pipe/topic name length (None = unlimited)
    """
    supports_persistence: bool = False
    supports_transactions: bool = False
    supports_clustering: bool = False
    supports_compression: bool = False
    supports_encryption: bool = False
    supports_authentication: bool = False
    supports_authorization: bool = False
    supports_dead_letter: bool = False
    supports_message_ordering: bool = False
    supports_exactly_once: bool = False
    max_message_size: Optional[int] = None
    max_pipe_length: Optional[int] = None


# Type aliases for better readability
MessageCallback = Callable[[bytes, MessageMetadata], Union[None, asyncio.Future[None]]]
"""
Type alias for message callback functions.

The callback receives:
- data: The raw message data as bytes
- metadata: Message metadata including headers, timestamp, etc.

The callback can be either sync or async (returning a Future).
"""

MiddlewareCallback = Callable[[str, Any, MessageMetadata], Union[Any, asyncio.Future[Any]]]
"""
Type alias for middleware callback functions.

The callback receives:
- pipe: The pipe/channel/topic name
- data: The message data (before/after serialization)
- metadata: Message metadata

Returns the potentially modified data.
"""

ConfigDict = Dict[str, Any]
"""Type alias for configuration dictionaries."""

HeaderDict = Dict[str, Union[str, int, float, bool]]
"""Type alias for message headers."""


# Protocol-specific type aliases
KafkaHeaders = Dict[str, Union[str, bytes]]
RabbitMQProperties = Dict[str, Any]
MQTTProperties = Dict[str, Any]
NATSHeaders = Dict[str, str]
RedisHeaders = Dict[str, str]
