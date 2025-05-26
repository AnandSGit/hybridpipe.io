"""
Core HybridPipe interface definition.

This module defines the abstract base class that all messaging protocol
implementations must inherit from, ensuring a consistent API across
all supported messaging systems.
"""

from abc import ABC, abstractmethod
from typing import Any, Optional, Dict, List
from contextlib import asynccontextmanager
import asyncio

from hybridpipe.core.types import (
    BrokerType,
    ConnectionState,
    MessageCallback,
    MessageMetadata,
    ProtocolCapabilities,
)
from hybridpipe.core.errors import HybridPipeError

__all__ = ["HybridPipe", "AsyncHybridPipeManager"]


class HybridPipe(ABC):
    """
    Abstract base class for all HybridPipe messaging implementations.

    This interface defines the contract that all messaging protocol
    implementations must follow, ensuring consistency across different
    messaging systems.

    All methods are async to support non-blocking I/O operations.
    Implementations should handle connection pooling, reconnection,
    and error recovery internally.
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize the HybridPipe implementation.

        Args:
            config: Protocol-specific configuration dictionary
        """
        self._config = config or {}
        self._connection_state = ConnectionState.DISCONNECTED
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._middleware_stack: Optional[Any] = None

    @property
    @abstractmethod
    def broker_type(self) -> BrokerType:
        """Get the broker type for this implementation."""
        pass

    @property
    @abstractmethod
    def capabilities(self) -> ProtocolCapabilities:
        """Get the capabilities supported by this protocol."""
        pass

    @property
    def connection_state(self) -> ConnectionState:
        """Get the current connection state."""
        return self._connection_state

    @property
    def is_connected(self) -> bool:
        """Check if currently connected to the messaging system."""
        return self._connection_state == ConnectionState.CONNECTED

    @property
    def config(self) -> Dict[str, Any]:
        """Get the current configuration."""
        return self._config.copy()

    @abstractmethod
    async def connect(self) -> None:
        """
        Establish connection to the messaging system.

        Raises:
            ConnectionError: If connection fails
            ConfigurationError: If configuration is invalid
            TimeoutError: If connection times out
        """
        pass

    @abstractmethod
    async def disconnect(self) -> None:
        """
        Close connection to the messaging system.

        Should gracefully close all subscriptions and clean up resources.

        Raises:
            ConnectionError: If disconnection fails
        """
        pass

    @abstractmethod
    async def dispatch(
        self,
        pipe: str,
        data: Any,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """
        Send a message to the specified pipe.

        Args:
            pipe: The pipe/channel/topic to send to
            data: The message data to send
            headers: Optional message headers
            metadata: Optional message metadata

        Raises:
            ConnectionError: If not connected
            SerializationError: If data cannot be serialized
            ProtocolError: If protocol-specific error occurs
            MessageTooLargeError: If message exceeds size limits
        """
        pass

    @abstractmethod
    async def dispatch_with_timeout(
        self,
        pipe: str,
        data: Any,
        timeout_seconds: float,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """
        Send a message with a timeout.

        Args:
            pipe: The pipe/channel/topic to send to
            data: The message data to send
            timeout_seconds: Maximum time to wait for send completion
            headers: Optional message headers
            metadata: Optional message metadata

        Raises:
            TimeoutError: If send operation times out
            ConnectionError: If not connected
            SerializationError: If data cannot be serialized
            ProtocolError: If protocol-specific error occurs
        """
        pass

    @abstractmethod
    async def subscribe(
        self,
        pipe: str,
        callback: MessageCallback,
        headers: Optional[Dict[str, Any]] = None,
    ) -> None:
        """
        Subscribe to messages from the specified pipe.

        Args:
            pipe: The pipe/channel/topic to subscribe to
            callback: Function to call when messages arrive
            headers: Optional subscription headers/options

        Raises:
            ConnectionError: If not connected
            ProtocolError: If subscription fails
            PipeNotFoundError: If pipe doesn't exist (for some protocols)
        """
        pass

    @abstractmethod
    async def unsubscribe(self, pipe: str) -> None:
        """
        Unsubscribe from the specified pipe.

        Args:
            pipe: The pipe/channel/topic to unsubscribe from

        Raises:
            ConnectionError: If not connected
            PipeNotFoundError: If not subscribed to pipe
        """
        pass

    async def unsubscribe_all(self) -> None:
        """
        Unsubscribe from all pipes.

        Default implementation calls unsubscribe for each subscribed pipe.
        Implementations can override for more efficient bulk operations.
        """
        pipes = list(self._subscriptions.keys())
        for pipe in pipes:
            try:
                await self.unsubscribe(pipe)
            except Exception as e:
                # Log error but continue unsubscribing from other pipes
                print(f"Failed to unsubscribe from pipe {pipe}: {e}")

    # Backward compatibility aliases
    async def accept(
        self,
        pipe: str,
        callback: MessageCallback,
        headers: Optional[Dict[str, Any]] = None,
    ) -> None:
        """Alias for subscribe() for backward compatibility."""
        await self.subscribe(pipe, callback, headers)

    async def remove(self, pipe: str) -> None:
        """Alias for unsubscribe() for backward compatibility."""
        await self.unsubscribe(pipe)

    async def close(self) -> None:
        """Alias for disconnect() for backward compatibility."""
        await self.disconnect()

    # Context manager support
    async def __aenter__(self) -> "HybridPipe":
        """Async context manager entry."""
        await self.connect()
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        """Async context manager exit."""
        await self.disconnect()

    # Health check and diagnostics
    async def health_check(self) -> Dict[str, Any]:
        """
        Perform a health check on the connection.

        Returns:
            Dictionary containing health status and metrics
        """
        return {
            "broker_type": self.broker_type.name,
            "connection_state": self.connection_state.value,
            "is_connected": self.is_connected,
            "subscriptions": list(self._subscriptions.keys()),
            "subscription_count": len(self._subscriptions),
        }

    def set_middleware_stack(self, middleware_stack: Any) -> None:
        """Set the middleware stack for this instance."""
        self._middleware_stack = middleware_stack

    def get_middleware_stack(self) -> Optional[Any]:
        """Get the current middleware stack."""
        return self._middleware_stack


class AsyncHybridPipeManager:
    """
    Async context manager for HybridPipe lifecycle management.

    Ensures proper connection and cleanup of HybridPipe instances.

    Example:
        async with AsyncHybridPipeManager(router) as pipe:
            await pipe.dispatch("test", {"message": "hello"})
    """

    def __init__(self, router: HybridPipe) -> None:
        self.router = router

    @asynccontextmanager
    async def managed_router(self):
        """Context manager that ensures proper cleanup."""
        try:
            if not self.router.is_connected:
                await self.router.connect()
            yield self.router
        finally:
            if self.router.is_connected:
                await self.router.disconnect()

    async def __aenter__(self) -> HybridPipe:
        """Async context manager entry."""
        if not self.router.is_connected:
            await self.router.connect()
        return self.router

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        """Async context manager exit."""
        if self.router.is_connected:
            await self.router.disconnect()
