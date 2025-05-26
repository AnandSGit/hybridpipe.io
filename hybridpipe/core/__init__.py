"""
Core module for HybridPipe messaging middleware.

This module contains the fundamental interfaces, types, and utilities
that form the foundation of the HybridPipe messaging system.
"""

from hybridpipe.core.interface import HybridPipe
from hybridpipe.core.types import (
    BrokerType,
    MessageCallback,
    SerializationFormat,
    ConnectionState,
    MessageMetadata,
)
from hybridpipe.core.errors import (
    HybridPipeError,
    ConnectionError,
    SerializationError,
    ProtocolError,
    ConfigurationError,
    TimeoutError,
)
from hybridpipe.core.registry import HybridPipeRegistry, deploy_router, register_factory
from hybridpipe.core.config import HybridPipeConfig, ProtocolConfig

__all__ = [
    "HybridPipe",
    "BrokerType",
    "MessageCallback",
    "SerializationFormat",
    "ConnectionState",
    "MessageMetadata",
    "HybridPipeError",
    "ConnectionError",
    "SerializationError",
    "ProtocolError",
    "ConfigurationError",
    "TimeoutError",
    "HybridPipeRegistry",
    "deploy_router",
    "register_factory",
    "HybridPipeConfig",
    "ProtocolConfig",
]
