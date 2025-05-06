#!/usr/bin/env python3
"""
HybridPipe Python Client

This module provides a Python client for the HybridPipe messaging system.
It supports multiple serialization formats for interoperability with the Go implementation.
"""

import json
import struct
import gzip
import time
import uuid
from enum import IntEnum
from typing import Any, Dict, Optional, Union, Callable

try:
    import msgpack
    MSGPACK_AVAILABLE = True
except ImportError:
    MSGPACK_AVAILABLE = False

try:
    import google.protobuf.message
    PROTOBUF_AVAILABLE = True
except ImportError:
    PROTOBUF_AVAILABLE = False

# Constants for message format
MESSAGE_MAGIC = 0x4850  # "HP" in ASCII
CURRENT_VERSION = 1
HEADER_SIZE = 9

# Serialization formats
class SerializationFormat(IntEnum):
    GOB = 0  # Go's native format (not supported in Python)
    JSON = 1  # Standard JSON
    PROTOBUF = 2  # Protocol Buffers
    MSGPACK = 3  # MessagePack

# Message flags
class MessageFlags(IntEnum):
    NONE = 0
    COMPRESSED = 1

class HybridPipeError(Exception):
    """Base exception for HybridPipe errors."""
    pass

class SerializationError(HybridPipeError):
    """Exception raised for serialization errors."""
    pass

class DeserializationError(HybridPipeError):
    """Exception raised for deserialization errors."""
    pass

class UnsupportedFormatError(HybridPipeError):
    """Exception raised when an unsupported serialization format is used."""
    pass

class SerializationOptions:
    """Options for serialization."""
    
    def __init__(
        self,
        format: SerializationFormat = SerializationFormat.JSON,
        compression: bool = False,
        compression_level: int = 6,
        protobuf_message_type: str = None
    ):
        self.format = format
        self.compression = compression
        self.compression_level = compression_level
        self.protobuf_message_type = protobuf_message_type

class HybridPipeClient:
    """
    Client for the HybridPipe messaging system.
    
    This client can connect to various messaging systems supported by HybridPipe
    and provides serialization/deserialization compatible with the Go implementation.
    """
    
    def __init__(self, broker_type: str, options: Optional[Dict[str, Any]] = None):
        """
        Initialize the HybridPipe client.
        
        Args:
            broker_type: The type of messaging system to use (e.g., "kafka", "nats").
            options: Additional options for the client.
        """
        self.broker_type = broker_type.lower()
        self.options = options or {}
        self.serialization_options = SerializationOptions()
        
        # Initialize the appropriate client based on broker_type
        if self.broker_type == "kafka":
            self._init_kafka_client()
        elif self.broker_type == "nats":
            self._init_nats_client()
        elif self.broker_type == "rabbitmq":
            self._init_rabbitmq_client()
        elif self.broker_type == "redis":
            self._init_redis_client()
        elif self.broker_type == "mqtt":
            self._init_mqtt_client()
        else:
            raise ValueError(f"Unsupported broker type: {broker_type}")
    
    def _init_kafka_client(self):
        """Initialize a Kafka client."""
        try:
            from kafka import KafkaProducer, KafkaConsumer
            
            # Get Kafka configuration
            bootstrap_servers = self.options.get("bootstrap_servers", "localhost:9092")
            
            # Create producer
            self.producer = KafkaProducer(
                bootstrap_servers=bootstrap_servers,
                value_serializer=lambda v: v  # We handle serialization ourselves
            )
            
            # Consumer will be created when subscribing to a topic
            self.consumer = None
        except ImportError:
            raise ImportError("Kafka support requires kafka-python package. Install with: pip install kafka-python")
    
    def _init_nats_client(self):
        """Initialize a NATS client."""
        try:
            import nats
            
            # Implementation details for NATS
            pass
        except ImportError:
            raise ImportError("NATS support requires nats-py package. Install with: pip install nats-py")
    
    def _init_rabbitmq_client(self):
        """Initialize a RabbitMQ client."""
        try:
            import pika
            
            # Implementation details for RabbitMQ
            pass
        except ImportError:
            raise ImportError("RabbitMQ support requires pika package. Install with: pip install pika")
    
    def _init_redis_client(self):
        """Initialize a Redis client."""
        try:
            import redis
            
            # Implementation details for Redis
            pass
        except ImportError:
            raise ImportError("Redis support requires redis package. Install with: pip install redis")
    
    def _init_mqtt_client(self):
        """Initialize an MQTT client."""
        try:
            import paho.mqtt.client as mqtt
            
            # Implementation details for MQTT
            pass
        except ImportError:
            raise ImportError("MQTT support requires paho-mqtt package. Install with: pip install paho-mqtt")
    
    def set_serialization_format(self, format: SerializationFormat):
        """
        Set the serialization format to use.
        
        Args:
            format: The serialization format.
        
        Raises:
            UnsupportedFormatError: If the format is not supported.
        """
        if format == SerializationFormat.GOB:
            raise UnsupportedFormatError("GOB format is not supported in Python")
        
        if format == SerializationFormat.PROTOBUF and not PROTOBUF_AVAILABLE:
            raise UnsupportedFormatError("Protocol Buffers support requires protobuf package")
        
        if format == SerializationFormat.MSGPACK and not MSGPACK_AVAILABLE:
            raise UnsupportedFormatError("MessagePack support requires msgpack package")
        
        self.serialization_options.format = format
    
    def enable_compression(self, enabled: bool = True, level: int = 6):
        """
        Enable or disable compression for messages.
        
        Args:
            enabled: Whether compression is enabled.
            level: Compression level (1-9, higher = better compression but slower).
        """
        self.serialization_options.compression = enabled
        self.serialization_options.compression_level = level
    
    def serialize(self, data: Any) -> bytes:
        """
        Serialize data using the configured format.
        
        Args:
            data: The data to serialize.
        
        Returns:
            The serialized data as bytes.
        
        Raises:
            SerializationError: If serialization fails.
        """
        try:
            # Serialize based on the format
            if self.serialization_options.format == SerializationFormat.JSON:
                payload = json.dumps(data).encode('utf-8')
            elif self.serialization_options.format == SerializationFormat.PROTOBUF:
                if not isinstance(data, google.protobuf.message.Message):
                    raise SerializationError("Data must be a Protocol Buffer message")
                payload = data.SerializeToString()
            elif self.serialization_options.format == SerializationFormat.MSGPACK:
                payload = msgpack.packb(data)
            else:
                raise UnsupportedFormatError(f"Unsupported serialization format: {self.serialization_options.format}")
            
            # Apply compression if enabled and beneficial
            flags = MessageFlags.NONE
            if self.serialization_options.compression and len(payload) > 64:
                compressed = gzip.compress(payload, self.serialization_options.compression_level)
                if len(compressed) < len(payload):
                    payload = compressed
                    flags = MessageFlags.COMPRESSED
            
            # Create the message header
            header = struct.pack(
                ">HBBBI",
                MESSAGE_MAGIC,
                CURRENT_VERSION,
                self.serialization_options.format,
                flags,
                len(payload)
            )
            
            # Combine header and payload
            return header + payload
        
        except Exception as e:
            raise SerializationError(f"Failed to serialize data: {e}")
    
    def deserialize(self, data: bytes) -> Any:
        """
        Deserialize data.
        
        Args:
            data: The serialized data.
        
        Returns:
            The deserialized data.
        
        Raises:
            DeserializationError: If deserialization fails.
        """
        try:
            # Check if the data has our header format
            if len(data) < HEADER_SIZE:
                raise DeserializationError("Data too short to contain header")
            
            # Read the header
            magic, version, format_byte, flags, payload_length = struct.unpack(">HBBBI", data[:HEADER_SIZE])
            
            # Validate the header
            if magic != MESSAGE_MAGIC:
                raise DeserializationError("Invalid message magic")
            
            # Check version compatibility
            if version > CURRENT_VERSION:
                raise DeserializationError(f"Unsupported protocol version: {version}")
            
            # Extract the payload
            if HEADER_SIZE + payload_length > len(data):
                raise DeserializationError("Invalid payload length")
            
            payload = data[HEADER_SIZE:HEADER_SIZE + payload_length]
            
            # Decompress if needed
            if flags & MessageFlags.COMPRESSED:
                payload = gzip.decompress(payload)
            
            # Deserialize based on the format
            format_enum = SerializationFormat(format_byte)
            
            if format_enum == SerializationFormat.JSON:
                return json.loads(payload.decode('utf-8'))
            elif format_enum == SerializationFormat.PROTOBUF:
                if not PROTOBUF_AVAILABLE:
                    raise UnsupportedFormatError("Protocol Buffers support requires protobuf package")
                # Note: The caller needs to parse this with the correct message type
                return payload
            elif format_enum == SerializationFormat.MSGPACK:
                if not MSGPACK_AVAILABLE:
                    raise UnsupportedFormatError("MessagePack support requires msgpack package")
                return msgpack.unpackb(payload)
            elif format_enum == SerializationFormat.GOB:
                raise UnsupportedFormatError("GOB format is not supported in Python")
            else:
                raise UnsupportedFormatError(f"Unsupported serialization format: {format_enum}")
        
        except Exception as e:
            raise DeserializationError(f"Failed to deserialize data: {e}")
    
    def dispatch(self, pipe: str, data: Any) -> None:
        """
        Send a message to the specified pipe.
        
        Args:
            pipe: The pipe (topic, channel, etc.) to send the message to.
            data: The data to send.
        
        Raises:
            Exception: If sending fails.
        """
        # Serialize the data
        serialized = self.serialize(data)
        
        # Send based on broker type
        if self.broker_type == "kafka":
            future = self.producer.send(pipe, value=serialized)
            # Wait for the message to be sent
            future.get(timeout=10)
        elif self.broker_type == "nats":
            # NATS implementation
            pass
        elif self.broker_type == "rabbitmq":
            # RabbitMQ implementation
            pass
        elif self.broker_type == "redis":
            # Redis implementation
            pass
        elif self.broker_type == "mqtt":
            # MQTT implementation
            pass
    
    def accept(self, pipe: str, handler: Callable[[Any], None]) -> None:
        """
        Subscribe to messages from the specified pipe.
        
        Args:
            pipe: The pipe (topic, channel, etc.) to subscribe to.
            handler: The function to call when a message is received.
        
        Raises:
            Exception: If subscription fails.
        """
        # Subscribe based on broker type
        if self.broker_type == "kafka":
            from kafka import KafkaConsumer
            
            # Create consumer if not already created
            if self.consumer is None:
                bootstrap_servers = self.options.get("bootstrap_servers", "localhost:9092")
                group_id = self.options.get("group_id", f"hybridpipe-{uuid.uuid4()}")
                
                self.consumer = KafkaConsumer(
                    pipe,
                    bootstrap_servers=bootstrap_servers,
                    group_id=group_id,
                    auto_offset_reset="earliest",
                    enable_auto_commit=True
                )
            
            # Start consuming in a separate thread
            import threading
            
            def consume():
                for message in self.consumer:
                    try:
                        # Deserialize the message
                        data = self.deserialize(message.value)
                        # Call the handler
                        handler(data)
                    except Exception as e:
                        print(f"Error processing message: {e}")
            
            thread = threading.Thread(target=consume, daemon=True)
            thread.start()
        
        elif self.broker_type == "nats":
            # NATS implementation
            pass
        elif self.broker_type == "rabbitmq":
            # RabbitMQ implementation
            pass
        elif self.broker_type == "redis":
            # Redis implementation
            pass
        elif self.broker_type == "mqtt":
            # MQTT implementation
            pass
    
    def close(self) -> None:
        """Close the client and release resources."""
        if self.broker_type == "kafka":
            if self.producer is not None:
                self.producer.close()
            if self.consumer is not None:
                self.consumer.close()
        elif self.broker_type == "nats":
            # NATS cleanup
            pass
        elif self.broker_type == "rabbitmq":
            # RabbitMQ cleanup
            pass
        elif self.broker_type == "redis":
            # Redis cleanup
            pass
        elif self.broker_type == "mqtt":
            # MQTT cleanup
            pass


# Example usage
if __name__ == "__main__":
    # Create a client
    client = HybridPipeClient("kafka", {
        "bootstrap_servers": "localhost:9092"
    })
    
    # Set serialization format to JSON (most compatible)
    client.set_serialization_format(SerializationFormat.JSON)
    
    # Enable compression for large messages
    client.enable_compression(True)
    
    # Define a message handler
    def handle_message(data):
        print(f"Received message: {data}")
    
    # Subscribe to a topic
    client.accept("example-topic", handle_message)
    
    # Send a message
    client.dispatch("example-topic", {
        "message": "Hello from Python!",
        "timestamp": time.time()
    })
    
    # Keep the program running to receive messages
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("Shutting down...")
    finally:
        client.close()
