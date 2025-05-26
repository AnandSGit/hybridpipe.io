"""
Decorators for HybridPipe protocol implementations.

This module provides decorators for registering protocol implementations
and adding metadata to protocol classes.
"""

from typing import Any, Dict, Optional, Callable, Type
from functools import wraps

from hybridpipe.core.types import BrokerType


def protocol_implementation(
    broker_type: BrokerType,
    default_config: Optional[Dict[str, Any]] = None,
    metadata: Optional[Dict[str, Any]] = None,
) -> Callable[[Type], Type]:
    """
    Decorator for registering protocol implementations.

    This decorator automatically registers a protocol implementation
    with the HybridPipe registry when the class is defined.

    Args:
        broker_type: The broker type this implementation handles
        default_config: Default configuration for this protocol
        metadata: Additional metadata about the protocol

    Returns:
        Decorated class with registration side effect

    Example:
        @protocol_implementation(
            BrokerType.KAFKA,
            default_config={"bootstrap_servers": ["localhost:9092"]},
            metadata={"supports_transactions": True}
        )
        class KafkaHybridPipe(HybridPipe):
            pass
    """
    def decorator(cls: Type) -> Type:
        # Add metadata to the class
        cls._broker_type = broker_type
        cls._default_config = default_config or {}
        cls._metadata = metadata or {}

        # Create factory function
        def factory(config: Optional[Dict[str, Any]] = None) -> Any:
            # Merge default config with provided config
            merged_config = cls._default_config.copy()
            if config:
                merged_config.update(config)
            return cls(merged_config)

        # Register the factory with the registry
        try:
            # Import here to avoid circular dependency
            from hybridpipe.core.registry import HybridPipeRegistry
            registry = HybridPipeRegistry()
            registry.register(broker_type, factory)
        except (ValueError, ImportError):
            # Already registered or registry not available yet
            pass

        return cls

    return decorator


# Additional decorators can be added here as needed
