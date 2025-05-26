"""
Serialization module for HybridPipe.

This module provides serialization and deserialization capabilities
for various data formats, maintaining wire format compatibility with
the original Go/Rust implementations.
"""

from hybridpipe.serialization.engine import (
    encode,
    decode,
    encode_with_options,
    decode_with_options,
    SerializationOptions,
    register_type,
    get_serializer,
)
from hybridpipe.serialization.formats import (
    JSONSerializer,
    MessagePackSerializer,
    ProtobufSerializer,
    PickleSerializer,
)

__all__ = [
    "encode",
    "decode",
    "encode_with_options", 
    "decode_with_options",
    "SerializationOptions",
    "register_type",
    "get_serializer",
    "JSONSerializer",
    "MessagePackSerializer",
    "ProtobufSerializer",
    "PickleSerializer",
]
