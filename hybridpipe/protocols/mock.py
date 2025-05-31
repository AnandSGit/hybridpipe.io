"""
Mock protocol implementation for HybridPipe.

This module provides an in-memory mock implementation for testing
and development purposes. It simulates message passing without
requiring external message brokers.
"""

import asyncio
import time
from typing import Any, Dict, List, Optional
from collections import defaultdict, deque
from datetime import datetime
import uuid

from hybridpipe.core.interface import HybridPipe
from hybridpipe.core.types import (
    BrokerType,
    ConnectionState,
    MessageCallback,
    MessageMetadata,
    ProtocolCapabilities,
    SerializationFormat,
)
from hybridpipe.core.errors import ConnectionError, ProtocolError
from hybridpipe.core.registry import protocol_implementation
from hybridpipe.serialization.engine import encode, decode


__all__ = ["MockHybridPipe"]


@protocol_implementation(
    BrokerType.MOCK,
    default_config={
        "max_queue_size": 1000,
        "delivery_delay_ms": 0,
        "simulate_failures": False,
        "failure_rate": 0.0,
    },
    metadata={
        "description": "In-memory mock implementation for testing",
        "supports_persistence": False,
        "supports_clustering": False,
    },
)
class MockHybridPipe(HybridPipe):
    """
    Mock implementation of HybridPipe for testing and development.
    
    This implementation provides an in-memory message passing system
    that simulates the behavior of real messaging brokers without
    requiring external dependencies.
    
    Features:
    - In-memory message queues
    - Configurable delivery delays
    - Failure simulation for testing
    - Message persistence (in memory only)
    - Multiple subscribers per pipe
    """
    
    # Class-level shared state for simulating a message broker
    _global_queues: Dict[str, deque] = defaultdict(lambda: deque(maxlen=1000))
    _global_subscribers: Dict[str, List[MessageCallback]] = defaultdict(list)
    _global_lock = asyncio.Lock()
    
    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize mock HybridPipe implementation.
        
        Args:
            config: Configuration dictionary with options:
                - max_queue_size: Maximum messages per queue (default: 1000)
                - delivery_delay_ms: Simulated delivery delay (default: 0)
                - simulate_failures: Whether to simulate failures (default: False)
                - failure_rate: Rate of simulated failures 0.0-1.0 (default: 0.0)
        """
        super().__init__(config)
        
        # Configuration
        self.max_queue_size = self._config.get("max_queue_size", 1000)
        self.delivery_delay_ms = self._config.get("delivery_delay_ms", 0)
        self.simulate_failures = self._config.get("simulate_failures", False)
        self.failure_rate = self._config.get("failure_rate", 0.0)
        
        # Instance state
        self._connection_state = ConnectionState.DISCONNECTED
        self._local_subscriptions: Dict[str, List[MessageCallback]] = defaultdict(list)
        self._message_count = 0
        self._error_count = 0
        self._start_time = time.time()
        
        # Background task for message delivery
        self._delivery_task: Optional[asyncio.Task] = None
        self._stop_delivery = False
    
    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.MOCK
    
    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=False,  # In-memory only
            supports_transactions=False,
            supports_clustering=False,
            supports_compression=True,
            supports_encryption=False,
            supports_authentication=False,
            supports_authorization=False,
            supports_dead_letter=False,
            supports_message_ordering=True,
            supports_exactly_once=False,
            max_message_size=None,  # No limit
            max_pipe_length=255,
        )
    
    async def connect(self) -> None:
        """Establish connection to the mock broker."""
        if self._connection_state == ConnectionState.CONNECTED:
            return
        
        self._connection_state = ConnectionState.CONNECTING
        
        # Simulate connection delay
        if self.delivery_delay_ms > 0:
            await asyncio.sleep(self.delivery_delay_ms / 1000.0)
        
        # Simulate connection failure
        if self.simulate_failures and self._should_fail():
            self._connection_state = ConnectionState.FAILED
            raise ConnectionError(
                "Simulated connection failure",
                broker_type=self.broker_type,
            )
        
        self._connection_state = ConnectionState.CONNECTED
        
        # Start message delivery task
        self._stop_delivery = False
        self._delivery_task = asyncio.create_task(self._delivery_loop())
    
    async def disconnect(self) -> None:
        """Close connection to the mock broker."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return
        
        self._connection_state = ConnectionState.DISCONNECTED
        
        # Stop delivery task
        self._stop_delivery = True
        if self._delivery_task:
            self._delivery_task.cancel()
            try:
                await self._delivery_task
            except asyncio.CancelledError:
                pass
            self._delivery_task = None
        
        # Remove all local subscriptions from global state
        async with self._global_lock:
            for pipe, callbacks in self._local_subscriptions.items():
                for callback in callbacks:
                    if callback in self._global_subscribers[pipe]:
                        self._global_subscribers[pipe].remove(callback)
        
        self._local_subscriptions.clear()
    
    async def dispatch(
        self,
        pipe: str,
        data: Any,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """Send a message to the specified pipe."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to mock broker",
                broker_type=self.broker_type,
                pipe=pipe,
            )
        
        # Simulate send failure
        if self.simulate_failures and self._should_fail():
            self._error_count += 1
            raise ProtocolError(
                "Simulated send failure",
                broker_type=self.broker_type,
                pipe=pipe,
            )
        
        # Create message metadata
        if metadata is None:
            metadata = MessageMetadata(
                message_id=str(uuid.uuid4()),
                timestamp=datetime.now(),
                pipe=pipe,
                broker_type=self.broker_type,
                serialization_format=SerializationFormat.JSON,
                size_bytes=0,
                headers=headers or {},
            )
        
        # Serialize the message
        serialized_data = encode(data, SerializationFormat.JSON)
        metadata.size_bytes = len(serialized_data)
        
        # Create message envelope
        message = {
            "data": serialized_data,
            "metadata": metadata,
            "headers": headers or {},
            "timestamp": time.time(),
        }
        
        # Add to queue
        async with self._global_lock:
            queue = self._global_queues[pipe]
            if len(queue) >= self.max_queue_size:
                # Remove oldest message if queue is full
                queue.popleft()
            queue.append(message)
        
        self._message_count += 1
    
    async def dispatch_with_timeout(
        self,
        pipe: str,
        data: Any,
        timeout_seconds: float,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """Send a message with timeout."""
        try:
            await asyncio.wait_for(
                self.dispatch(pipe, data, headers, metadata),
                timeout=timeout_seconds
            )
        except asyncio.TimeoutError:
            from hybridpipe.core.errors import TimeoutError
            raise TimeoutError(
                f"Message dispatch timed out after {timeout_seconds} seconds",
                timeout_seconds=timeout_seconds,
                operation="dispatch",
                broker_type=self.broker_type,
                pipe=pipe,
            )
    
    async def subscribe(
        self,
        pipe: str,
        callback: MessageCallback,
        headers: Optional[Dict[str, Any]] = None,
    ) -> None:
        """Subscribe to messages from the specified pipe."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to mock broker",
                broker_type=self.broker_type,
                pipe=pipe,
            )
        
        # Add to local subscriptions
        self._local_subscriptions[pipe].append(callback)
        
        # Add to global subscriptions
        async with self._global_lock:
            self._global_subscribers[pipe].append(callback)
    
    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified pipe."""
        if pipe not in self._local_subscriptions:
            return
        
        # Remove from global subscriptions
        async with self._global_lock:
            callbacks = self._local_subscriptions[pipe]
            for callback in callbacks:
                if callback in self._global_subscribers[pipe]:
                    self._global_subscribers[pipe].remove(callback)
        
        # Remove from local subscriptions
        del self._local_subscriptions[pipe]
    
    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()
        
        # Add mock-specific health information
        uptime = time.time() - self._start_time
        
        async with self._global_lock:
            total_queued_messages = sum(len(queue) for queue in self._global_queues.values())
            total_subscribers = sum(len(subs) for subs in self._global_subscribers.values())
        
        base_health.update({
            "uptime_seconds": uptime,
            "messages_sent": self._message_count,
            "errors": self._error_count,
            "total_queued_messages": total_queued_messages,
            "total_subscribers": total_subscribers,
            "local_subscriptions": {
                pipe: len(callbacks)
                for pipe, callbacks in self._local_subscriptions.items()
            },
            "config": {
                "max_queue_size": self.max_queue_size,
                "delivery_delay_ms": self.delivery_delay_ms,
                "simulate_failures": self.simulate_failures,
                "failure_rate": self.failure_rate,
            },
        })
        
        return base_health
    
    def _should_fail(self) -> bool:
        """Determine if operation should fail based on failure rate."""
        import random
        return random.random() < self.failure_rate
    
    async def _delivery_loop(self) -> None:
        """Background task for delivering messages to subscribers."""
        while not self._stop_delivery:
            try:
                await self._process_queued_messages()
                
                # Small delay to prevent busy waiting
                await asyncio.sleep(0.01)
                
            except asyncio.CancelledError:
                break
            except Exception as e:
                # Log error but continue processing
                import structlog # type: ignore
                logger = structlog.get_logger()
                logger.error(
                    "Error in mock delivery loop",
                    error=str(e),
                    error_type=type(e).__name__,
                )
                await asyncio.sleep(0.1)
    
    async def _process_queued_messages(self) -> None:
        """Process messages from all queues."""
        async with self._global_lock:
            # Get a snapshot of queues and subscribers
            queue_snapshot = {
                pipe: list(queue) for pipe, queue in self._global_queues.items()
            }
            subscriber_snapshot = {
                pipe: list(subs) for pipe, subs in self._global_subscribers.items()
            }
            
            # Clear processed messages from queues
            for pipe in queue_snapshot:
                self._global_queues[pipe].clear()
        
        # Process messages outside the lock
        for pipe, messages in queue_snapshot.items():
            subscribers = subscriber_snapshot.get(pipe, [])
            
            for message in messages:
                # Apply delivery delay if configured
                if self.delivery_delay_ms > 0:
                    await asyncio.sleep(self.delivery_delay_ms / 1000.0)
                
                # Deliver to all subscribers
                for callback in subscribers:
                    try:
                        # Deserialize message data
                        data = message["data"]
                        metadata = message["metadata"]
                        
                        # Call subscriber callback
                        result = callback(data, metadata)
                        if asyncio.iscoroutine(result):
                            await result
                            
                    except Exception as e:
                        # Log delivery error but continue
                        import structlog # type: ignore
                        logger = structlog.get_logger()
                        logger.error(
                            "Error delivering message to subscriber",
                            pipe=pipe,
                            message_id=message["metadata"].message_id,
                            error=str(e),
                            error_type=type(e).__name__,
                        )
    
    # Utility methods for testing
    def get_queue_size(self, pipe: str) -> int:
        """Get the current queue size for a pipe."""
        return len(self._global_queues[pipe])
    
    def get_subscriber_count(self, pipe: str) -> int:
        """Get the number of subscribers for a pipe."""
        return len(self._global_subscribers[pipe])
    
    def clear_queue(self, pipe: str) -> None:
        """Clear all messages from a pipe's queue."""
        self._global_queues[pipe].clear()
    
    @classmethod
    def clear_all_queues(cls) -> None:
        """Clear all messages from all queues (class method for testing)."""
        cls._global_queues.clear()
        cls._global_subscribers.clear()
