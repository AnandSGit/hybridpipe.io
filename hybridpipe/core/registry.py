"""
Registry system for HybridPipe protocol implementations.

This module provides a thread-safe registry for managing protocol factories
and deploying router instances. It implements the singleton pattern to ensure
a single global registry across the application.
"""

from typing import Any, Callable, Dict, List, Optional
import threading

from hybridpipe.core.types import BrokerType
from hybridpipe.core.interface import HybridPipe
from hybridpipe.core.errors import (
    BrokerNotSupportedError,
    ConfigurationError,
    HybridPipeError,
)

__all__ = ["HybridPipeRegistry", "deploy_router", "register_factory", "get_registry"]

# Type alias for factory functions
RouterFactory = Callable[[Optional[Dict[str, Any]]], HybridPipe]


class HybridPipeRegistry:
    """
    Thread-safe singleton registry for HybridPipe protocol implementations.

    This registry manages factory functions for creating protocol-specific
    HybridPipe instances. It ensures thread safety and provides a consistent
    interface for protocol registration and deployment.
    """

    _instance: Optional["HybridPipeRegistry"] = None
    _lock = threading.Lock()

    def __new__(cls) -> "HybridPipeRegistry":
        """Singleton pattern implementation."""
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = super().__new__(cls)
                    cls._instance._initialize()
        return cls._instance

    def _initialize(self) -> None:
        """Initialize the registry instance."""
        self._factories: Dict[BrokerType, RouterFactory] = {}
        self._factory_lock = threading.RLock()
        self._default_configs: Dict[BrokerType, Dict[str, Any]] = {}
        self._protocol_metadata: Dict[BrokerType, Dict[str, Any]] = {}

    def register(
        self,
        broker_type: BrokerType,
        factory: RouterFactory,
        default_config: Optional[Dict[str, Any]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> None:
        """
        Register a factory function for a broker type.

        Args:
            broker_type: The type of broker this factory creates
            factory: Function that creates HybridPipe instances
            default_config: Default configuration for this broker type
            metadata: Additional metadata about this protocol

        Raises:
            ValueError: If broker_type is already registered
        """
        with self._factory_lock:
            if broker_type in self._factories:
                raise ValueError(f"Broker type {broker_type.name} is already registered")

            self._factories[broker_type] = factory
            self._default_configs[broker_type] = default_config or {}
            self._protocol_metadata[broker_type] = metadata or {}

    def unregister(self, broker_type: BrokerType) -> None:
        """
        Unregister a broker type.

        Args:
            broker_type: The broker type to unregister

        Raises:
            BrokerNotSupportedError: If broker type is not registered
        """
        with self._factory_lock:
            if broker_type not in self._factories:
                raise BrokerNotSupportedError(
                    f"Broker type {broker_type.name} is not registered",
                    requested_broker=broker_type,
                    available_brokers=list(self._factories.keys()),
                )

            del self._factories[broker_type]
            del self._default_configs[broker_type]
            del self._protocol_metadata[broker_type]

    async def deploy_router(
        self,
        broker_type: BrokerType,
        config: Optional[Dict[str, Any]] = None,
        auto_connect: bool = True,
    ) -> HybridPipe:
        """
        Deploy a router instance for the specified broker type.

        Args:
            broker_type: The type of broker to deploy
            config: Configuration for the router (merged with defaults)
            auto_connect: Whether to automatically connect after creation

        Returns:
            Connected HybridPipe instance

        Raises:
            BrokerNotSupportedError: If broker type is not registered
            ConfigurationError: If configuration is invalid
            ConnectionError: If auto_connect is True and connection fails
        """
        with self._factory_lock:
            if broker_type not in self._factories:
                raise BrokerNotSupportedError(
                    f"Broker type {broker_type.name} is not supported",
                    requested_broker=broker_type,
                    available_brokers=list(self._factories.keys()),
                )

            factory = self._factories[broker_type]
            default_config = self._default_configs[broker_type]

        # Merge configuration with defaults
        merged_config = default_config.copy()
        if config:
            merged_config.update(config)

        try:
            # Create the router instance
            router = factory(merged_config)

            # Auto-connect if requested
            if auto_connect:
                await router.connect()

            return router

        except Exception as e:
            if isinstance(e, HybridPipeError):
                raise
            raise ConfigurationError(
                f"Failed to create router for {broker_type.name}: {str(e)}",
                broker_type=broker_type,
                cause=e,
            )

    def get_registered_protocols(self) -> List[BrokerType]:
        """
        Get a list of all registered broker types.

        Returns:
            List of registered BrokerType values
        """
        with self._factory_lock:
            return list(self._factories.keys())

    def is_registered(self, broker_type: BrokerType) -> bool:
        """
        Check if a broker type is registered.

        Args:
            broker_type: The broker type to check

        Returns:
            True if registered, False otherwise
        """
        with self._factory_lock:
            return broker_type in self._factories

    def get_default_config(self, broker_type: BrokerType) -> Dict[str, Any]:
        """
        Get the default configuration for a broker type.

        Args:
            broker_type: The broker type

        Returns:
            Default configuration dictionary

        Raises:
            BrokerNotSupportedError: If broker type is not registered
        """
        with self._factory_lock:
            if broker_type not in self._default_configs:
                raise BrokerNotSupportedError(
                    f"Broker type {broker_type.name} is not registered",
                    requested_broker=broker_type,
                    available_brokers=list(self._factories.keys()),
                )
            return self._default_configs[broker_type].copy()

    def get_protocol_metadata(self, broker_type: BrokerType) -> Dict[str, Any]:
        """
        Get metadata for a protocol.

        Args:
            broker_type: The broker type

        Returns:
            Protocol metadata dictionary

        Raises:
            BrokerNotSupportedError: If broker type is not registered
        """
        with self._factory_lock:
            if broker_type not in self._protocol_metadata:
                raise BrokerNotSupportedError(
                    f"Broker type {broker_type.name} is not registered",
                    requested_broker=broker_type,
                    available_brokers=list(self._factories.keys()),
                )
            return self._protocol_metadata[broker_type].copy()

    def get_registry_info(self) -> Dict[str, Any]:
        """
        Get comprehensive information about the registry.

        Returns:
            Dictionary containing registry information
        """
        with self._factory_lock:
            return {
                "registered_protocols": [bt.name for bt in self._factories.keys()],
                "protocol_count": len(self._factories),
                "protocols": {
                    bt.name: {
                        "default_config": self._default_configs.get(bt, {}),
                        "metadata": self._protocol_metadata.get(bt, {}),
                    }
                    for bt in self._factories.keys()
                },
            }


# Global registry instance
_global_registry: Optional[HybridPipeRegistry] = None


def get_registry() -> HybridPipeRegistry:
    """
    Get the global HybridPipe registry instance.

    Returns:
        The singleton registry instance
    """
    global _global_registry
    if _global_registry is None:
        _global_registry = HybridPipeRegistry()
    return _global_registry


def register_factory(
    broker_type: BrokerType,
    factory: RouterFactory,
    default_config: Optional[Dict[str, Any]] = None,
    metadata: Optional[Dict[str, Any]] = None,
) -> None:
    """
    Register a factory function for a broker type.

    This is a convenience function that uses the global registry.

    Args:
        broker_type: The type of broker this factory creates
        factory: Function that creates HybridPipe instances
        default_config: Default configuration for this broker type
        metadata: Additional metadata about this protocol
    """
    registry = get_registry()
    registry.register(broker_type, factory, default_config, metadata)


async def deploy_router(
    broker_type: BrokerType,
    config: Optional[Dict[str, Any]] = None,
    auto_connect: bool = True,
) -> HybridPipe:
    """
    Deploy a router instance for the specified broker type.

    This is a convenience function that uses the global registry.

    Args:
        broker_type: The type of broker to deploy
        config: Configuration for the router
        auto_connect: Whether to automatically connect after creation

    Returns:
        HybridPipe instance
    """
    registry = get_registry()
    return await registry.deploy_router(broker_type, config, auto_connect)


# Note: protocol_implementation decorator is defined in hybridpipe.core.decorators
# to avoid circular imports
