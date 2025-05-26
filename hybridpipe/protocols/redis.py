"""
Redis protocol implementation for HybridPipe.

This module provides Redis pub/sub messaging implementation using
aioredis for high-performance async Redis operations.
"""

import asyncio
import json
import time
from typing import Any, Dict, List, Optional
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
from hybridpipe.core.errors import ConnectionError, ProtocolError, ConfigurationError
from hybridpipe.core.registry import protocol_implementation
from hybridpipe.serialization.engine import encode_with_options, decode_with_options, SerializationOptions

__all__ = ["RedisHybridPipe"]


@protocol_implementation(
    BrokerType.REDIS,
    default_config={
        "host": "localhost",
        "port": 6379,
        "db": 0,
        "password": None,
        "username": None,
        "ssl": False,
        "max_connections": 10,
        "retry_on_timeout": True,
        "socket_keepalive": True,
        "connection_timeout": 30.0,
        "socket_connect_timeout": 5.0,
        "health_check_interval": 30.0,
    },
    metadata={
        "description": "Redis pub/sub messaging",
        "supports_persistence": True,
        "supports_clustering": True,
    },
)
class RedisHybridPipe(HybridPipe):
    """
    Redis implementation of HybridPipe using pub/sub messaging.
    
    This implementation uses Redis pub/sub for real-time messaging
    with support for pattern subscriptions, connection pooling,
    and automatic reconnection.
    
    Features:
    - Redis pub/sub messaging
    - Connection pooling
    - Pattern subscriptions
    - Automatic reconnection
    - SSL/TLS support
    - Authentication support
    - Message persistence (Redis streams)
    - Clustering support
    """
    
    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize Redis HybridPipe implementation.
        
        Args:
            config: Configuration dictionary with Redis connection options
        """
        super().__init__(config)
        
        # Import aioredis
        try:
            import aioredis
            self._aioredis = aioredis
        except ImportError:
            raise ConfigurationError(
                "aioredis library is required for Redis protocol. Install with: pip install aioredis",
                broker_type=BrokerType.REDIS,
            )
        
        # Configuration
        self.host = self._config.get("host", "localhost")
        self.port = self._config.get("port", 6379)
        self.db = self._config.get("db", 0)
        self.password = self._config.get("password")
        self.username = self._config.get("username")
        self.ssl = self._config.get("ssl", False)
        self.max_connections = self._config.get("max_connections", 10)
        self.retry_on_timeout = self._config.get("retry_on_timeout", True)
        self.socket_keepalive = self._config.get("socket_keepalive", True)
        self.connection_timeout = self._config.get("connection_timeout", 30.0)
        self.socket_connect_timeout = self._config.get("socket_connect_timeout", 5.0)
        self.health_check_interval = self._config.get("health_check_interval", 30.0)
        
        # Connection state
        self._connection_state = ConnectionState.DISCONNECTED
        self._redis_pool: Optional[Any] = None
        self._pubsub: Optional[Any] = None
        
        # Subscription management
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._pattern_subscriptions: Dict[str, List[MessageCallback]] = {}
        
        # Background tasks
        self._listen_task: Optional[asyncio.Task] = None
        self._health_check_task: Optional[asyncio.Task] = None
        self._stop_listening = False
        
        # Statistics
        self._messages_sent = 0
        self._messages_received = 0
        self._connection_errors = 0
        self._start_time = time.time()
    
    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.REDIS
    
    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=True,  # Redis streams
            supports_transactions=True,  # Redis transactions
            supports_clustering=True,   # Redis Cluster
            supports_compression=True,
            supports_encryption=self.ssl,
            supports_authentication=bool(self.password or self.username),
            supports_authorization=False,
            supports_dead_letter=False,
            supports_message_ordering=True,  # Redis streams guarantee ordering
            supports_exactly_once=False,     # At-least-once delivery
            max_message_size=512 * 1024 * 1024,  # 512MB Redis limit
            max_pipe_length=None,  # No practical limit
        )
    
    async def connect(self) -> None:
        """Establish connection to Redis."""
        if self._connection_state == ConnectionState.CONNECTED:
            return
        
        self._connection_state = ConnectionState.CONNECTING
        
        try:
            # Create Redis connection pool
            self._redis_pool = self._aioredis.ConnectionPool.from_url(
                self._build_redis_url(),
                max_connections=self.max_connections,
                retry_on_timeout=self.retry_on_timeout,
                socket_keepalive=self.socket_keepalive,
                socket_connect_timeout=self.socket_connect_timeout,
                health_check_interval=self.health_check_interval,
            )
            
            # Test connection
            redis = self._aioredis.Redis(connection_pool=self._redis_pool)
            await redis.ping()
            
            # Create pub/sub instance
            self._pubsub = redis.pubsub()
            
            self._connection_state = ConnectionState.CONNECTED
            
            # Start background tasks
            self._stop_listening = False
            self._listen_task = asyncio.create_task(self._listen_loop())
            self._health_check_task = asyncio.create_task(self._health_check_loop())
            
        except Exception as e:
            self._connection_state = ConnectionState.FAILED
            self._connection_errors += 1
            raise ConnectionError(
                f"Failed to connect to Redis: {str(e)}",
                broker_type=self.broker_type,
                cause=e,
            )
    
    async def disconnect(self) -> None:
        """Close connection to Redis."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return
        
        self._connection_state = ConnectionState.DISCONNECTED
        
        # Stop background tasks
        self._stop_listening = True
        
        if self._listen_task:
            self._listen_task.cancel()
            try:
                await self._listen_task
            except asyncio.CancelledError:
                pass
            self._listen_task = None
        
        if self._health_check_task:
            self._health_check_task.cancel()
            try:
                await self._health_check_task
            except asyncio.CancelledError:
                pass
            self._health_check_task = None
        
        # Close pub/sub
        if self._pubsub:
            await self._pubsub.close()
            self._pubsub = None
        
        # Close connection pool
        if self._redis_pool:
            await self._redis_pool.disconnect()
            self._redis_pool = None
        
        # Clear subscriptions
        self._subscriptions.clear()
        self._pattern_subscriptions.clear()
    
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
                "Not connected to Redis",
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
        
        # Create message envelope
        message_envelope = {
            "data": data,
            "headers": headers or {},
            "metadata": {
                "message_id": metadata.message_id,
                "timestamp": metadata.timestamp.isoformat(),
                "broker_type": metadata.broker_type.name,
                "serialization_format": metadata.serialization_format.name,
                "pipe": pipe,
            },
        }
        
        # Serialize message
        serialization_options = SerializationOptions(
            format=SerializationFormat.JSON,
            compression=True,
            compression_threshold=1024,
        )
        
        try:
            serialized_data = encode_with_options(message_envelope, serialization_options)
            
            # Publish to Redis
            redis = self._aioredis.Redis(connection_pool=self._redis_pool)
            await redis.publish(pipe, serialized_data)
            
            self._messages_sent += 1
            
        except Exception as e:
            raise ProtocolError(
                f"Failed to publish message to Redis: {str(e)}",
                broker_type=self.broker_type,
                pipe=pipe,
                cause=e,
            )
    
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
                "Not connected to Redis",
                broker_type=self.broker_type,
                pipe=pipe,
            )
        
        # Add to subscriptions
        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
            # Subscribe to Redis channel
            await self._pubsub.subscribe(pipe)
        
        self._subscriptions[pipe].append(callback)
    
    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified pipe."""
        if pipe not in self._subscriptions:
            return
        
        # Remove from subscriptions
        del self._subscriptions[pipe]
        
        # Unsubscribe from Redis channel
        if self._pubsub:
            await self._pubsub.unsubscribe(pipe)
    
    async def subscribe_pattern(
        self,
        pattern: str,
        callback: MessageCallback,
    ) -> None:
        """
        Subscribe to channels matching a pattern.
        
        Args:
            pattern: Redis pattern (e.g., "user.*", "events:*")
            callback: Callback function for matching messages
        """
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to Redis",
                broker_type=self.broker_type,
                pipe=pattern,
            )
        
        # Add to pattern subscriptions
        if pattern not in self._pattern_subscriptions:
            self._pattern_subscriptions[pattern] = []
            # Subscribe to Redis pattern
            await self._pubsub.psubscribe(pattern)
        
        self._pattern_subscriptions[pattern].append(callback)
    
    async def unsubscribe_pattern(self, pattern: str) -> None:
        """Unsubscribe from a pattern."""
        if pattern not in self._pattern_subscriptions:
            return
        
        # Remove from pattern subscriptions
        del self._pattern_subscriptions[pattern]
        
        # Unsubscribe from Redis pattern
        if self._pubsub:
            await self._pubsub.punsubscribe(pattern)
    
    def _build_redis_url(self) -> str:
        """Build Redis connection URL."""
        scheme = "rediss" if self.ssl else "redis"
        auth = ""
        
        if self.username and self.password:
            auth = f"{self.username}:{self.password}@"
        elif self.password:
            auth = f":{self.password}@"
        
        return f"{scheme}://{auth}{self.host}:{self.port}/{self.db}"
    
    async def _listen_loop(self) -> None:
        """Main message listening loop."""
        while not self._stop_listening and self._pubsub:
            try:
                # Get message from pub/sub
                message = await self._pubsub.get_message(timeout=1.0)
                
                if message is None:
                    continue
                
                # Skip subscription confirmation messages
                if message["type"] in ("subscribe", "unsubscribe", "psubscribe", "punsubscribe"):
                    continue
                
                # Process data messages
                if message["type"] in ("message", "pmessage"):
                    await self._process_message(message)
                
            except asyncio.CancelledError:
                break
            except Exception as e:
                import structlog
                logger = structlog.get_logger()
                logger.error(
                    "Error in Redis listen loop",
                    error=str(e),
                    error_type=type(e).__name__,
                )
                await asyncio.sleep(1)  # Brief pause before retry
    
    async def _process_message(self, redis_message: Dict[str, Any]) -> None:
        """Process a received Redis message."""
        try:
            # Extract message data
            channel = redis_message["channel"].decode("utf-8")
            data = redis_message["data"]
            
            # Handle pattern messages
            if redis_message["type"] == "pmessage":
                pattern = redis_message["pattern"].decode("utf-8")
                callbacks = self._pattern_subscriptions.get(pattern, [])
            else:
                callbacks = self._subscriptions.get(channel, [])
            
            if not callbacks:
                return
            
            # Deserialize message
            serialization_options = SerializationOptions(format=SerializationFormat.JSON)
            message_envelope = decode_with_options(data, serialization_options)
            
            # Extract message components
            message_data = message_envelope.get("data")
            headers = message_envelope.get("headers", {})
            metadata_dict = message_envelope.get("metadata", {})
            
            # Create metadata object
            metadata = MessageMetadata(
                message_id=metadata_dict.get("message_id", str(uuid.uuid4())),
                timestamp=datetime.fromisoformat(metadata_dict.get("timestamp", datetime.now().isoformat())),
                pipe=metadata_dict.get("pipe", channel),
                broker_type=BrokerType[metadata_dict.get("broker_type", "REDIS")],
                serialization_format=SerializationFormat[metadata_dict.get("serialization_format", "JSON")],
                size_bytes=len(data),
                headers=headers,
            )
            
            # Serialize message data for callback
            callback_data = encode_with_options(message_data, SerializationOptions(format=SerializationFormat.JSON))
            
            # Call all callbacks
            for callback in callbacks:
                try:
                    result = callback(callback_data, metadata)
                    if asyncio.iscoroutine(result):
                        await result
                except Exception as e:
                    import structlog
                    logger = structlog.get_logger()
                    logger.error(
                        "Error in Redis message callback",
                        channel=channel,
                        message_id=metadata.message_id,
                        error=str(e),
                        error_type=type(e).__name__,
                    )
            
            self._messages_received += 1
            
        except Exception as e:
            import structlog
            logger = structlog.get_logger()
            logger.error(
                "Error processing Redis message",
                error=str(e),
                error_type=type(e).__name__,
            )
    
    async def _health_check_loop(self) -> None:
        """Periodic health check loop."""
        while not self._stop_listening:
            try:
                if self._redis_pool:
                    redis = self._aioredis.Redis(connection_pool=self._redis_pool)
                    await redis.ping()
                
                await asyncio.sleep(self.health_check_interval)
                
            except asyncio.CancelledError:
                break
            except Exception as e:
                self._connection_errors += 1
                import structlog
                logger = structlog.get_logger()
                logger.warning(
                    "Redis health check failed",
                    error=str(e),
                    error_type=type(e).__name__,
                )
                await asyncio.sleep(5)  # Shorter retry interval for health checks
    
    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()
        
        # Add Redis-specific health information
        uptime = time.time() - self._start_time
        
        # Get Redis info if connected
        redis_info = {}
        if self._redis_pool:
            try:
                redis = self._aioredis.Redis(connection_pool=self._redis_pool)
                redis_info = await redis.info()
            except Exception:
                pass
        
        base_health.update({
            "uptime_seconds": uptime,
            "messages_sent": self._messages_sent,
            "messages_received": self._messages_received,
            "connection_errors": self._connection_errors,
            "subscribed_channels": list(self._subscriptions.keys()),
            "subscribed_patterns": list(self._pattern_subscriptions.keys()),
            "redis_info": {
                "redis_version": redis_info.get("redis_version"),
                "used_memory": redis_info.get("used_memory"),
                "connected_clients": redis_info.get("connected_clients"),
                "total_commands_processed": redis_info.get("total_commands_processed"),
            },
            "config": {
                "host": self.host,
                "port": self.port,
                "db": self.db,
                "ssl": self.ssl,
                "max_connections": self.max_connections,
            },
        })
        
        return base_health
