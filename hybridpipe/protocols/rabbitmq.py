"""
RabbitMQ protocol implementation for HybridPipe.

This module provides AMQP 0.9.1 messaging support using aio-pika
with full exchange/queue management and connection recovery.
"""

import asyncio
import json
import time
import uuid
from typing import Any, Dict, List, Optional, Set
from datetime import datetime

try:
    import aio_pika
    from aio_pika import (
        connect_robust, Message, DeliveryMode, ExchangeType,
        IncomingMessage, RobustConnection, RobustChannel, RobustExchange, RobustQueue
    )
    from aio_pika.exceptions import AMQPException, ConnectionClosed
    RABBITMQ_AVAILABLE = True
except ImportError:
    RABBITMQ_AVAILABLE = False

from hybridpipe.core.interface import HybridPipe
from hybridpipe.core.types import (
    BrokerType,
    MessageCallback,
    MessageMetadata,
    ProtocolCapabilities,
    ConnectionState,
    SerializationFormat,
)
from hybridpipe.core.errors import (
    ConnectionError,
    ProtocolError,
    TimeoutError,
    SerializationError,
)
from hybridpipe.core.decorators import protocol_implementation
from hybridpipe.serialization.engine import encode, decode


@protocol_implementation(
    BrokerType.RABBITMQ,
    default_config={
        "host": "localhost",
        "port": 5672,
        "virtual_host": "/",
        "username": "guest",
        "password": "guest",
        "ssl": False,
        "ssl_verify": True,
        "heartbeat": 60,
        "connection_timeout": 30,
        "channel_max": 2047,
        "frame_max": 131072,
        "exchange_name": "hybridpipe",
        "exchange_type": "topic",
        "exchange_durable": True,
        "queue_durable": True,
        "queue_auto_delete": False,
        "message_ttl": None,
        "max_priority": None,
        "prefetch_count": 10,
        "auto_ack": False,
        "delivery_mode": "persistent",
    },
    metadata={
        "description": "RabbitMQ AMQP 0.9.1 messaging with exchanges and queues",
        "supports_persistence": True,
        "supports_clustering": True,
        "supports_transactions": True,
        "supports_routing": True,
    },
)
class RabbitMQHybridPipe(HybridPipe):
    """
    RabbitMQ implementation of HybridPipe using AMQP 0.9.1.

    This implementation provides full RabbitMQ functionality including
    exchanges, queues, routing keys, and message acknowledgments.

    Features:
    - Exchange and queue management
    - Message routing with patterns
    - Durable queues and persistent messages
    - Connection recovery and heartbeat
    - Message acknowledgments
    - Virtual host support
    - SSL/TLS encryption
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize RabbitMQ HybridPipe implementation.

        Args:
            config: Configuration dictionary with RabbitMQ-specific options
        """
        if not RABBITMQ_AVAILABLE:
            raise ImportError(
                "aio-pika is required for RabbitMQ support. "
                "Install with: pip install aio-pika"
            )

        super().__init__(config)

        # RabbitMQ configuration
        self.host = self._config.get("host", "localhost")
        self.port = self._config.get("port", 5672)
        self.virtual_host = self._config.get("virtual_host", "/")
        self.username = self._config.get("username", "guest")
        self.password = self._config.get("password", "guest")
        self.ssl = self._config.get("ssl", False)
        self.ssl_verify = self._config.get("ssl_verify", True)
        self.heartbeat = self._config.get("heartbeat", 60)
        self.connection_timeout = self._config.get("connection_timeout", 30)

        # Exchange and queue configuration
        self.exchange_name = self._config.get("exchange_name", "hybridpipe")
        self.exchange_type = self._config.get("exchange_type", "topic")
        self.exchange_durable = self._config.get("exchange_durable", True)
        self.queue_durable = self._config.get("queue_durable", True)
        self.queue_auto_delete = self._config.get("queue_auto_delete", False)
        self.prefetch_count = self._config.get("prefetch_count", 10)
        self.auto_ack = self._config.get("auto_ack", False)

        # Message configuration
        delivery_mode = self._config.get("delivery_mode", "persistent")
        self.delivery_mode = (
            DeliveryMode.PERSISTENT if delivery_mode == "persistent"
            else DeliveryMode.NOT_PERSISTENT
        )

        # Connection state
        self._connection_state = ConnectionState.DISCONNECTED
        self._connection: Optional[RobustConnection] = None
        self._channel: Optional[RobustChannel] = None
        self._exchange: Optional[RobustExchange] = None

        # Subscription management
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._queues: Dict[str, RobustQueue] = {}
        self._consumers: Dict[str, Any] = {}

        # Metrics
        self._messages_published = 0
        self._messages_consumed = 0
        self._publish_errors = 0
        self._consume_errors = 0
        self._start_time = time.time()

    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.RABBITMQ

    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=True,
            supports_transactions=True,
            supports_clustering=True,
            supports_compression=False,  # Handled at application level
            supports_encryption=self.ssl,
            supports_authentication=True,
            supports_authorization=True,
            supports_dead_letter=True,
            supports_message_ordering=True,
            supports_exactly_once=False,  # At-least-once delivery
            max_message_size=128 * 1024 * 1024,  # 128MB default
            max_pipe_length=255,  # Queue name limit
        )

    async def connect(self) -> None:
        """Establish connection to RabbitMQ server."""
        if self._connection_state == ConnectionState.CONNECTED:
            return

        self._connection_state = ConnectionState.CONNECTING

        try:
            # Build connection URL
            connection_url = self._build_connection_url()

            # Create robust connection with automatic recovery
            self._connection = await connect_robust(
                connection_url,
                heartbeat=self.heartbeat,
                connection_timeout=self.connection_timeout,
            )

            # Create channel
            self._channel = await self._connection.channel()

            # Set QoS (prefetch count)
            await self._channel.set_qos(prefetch_count=self.prefetch_count)

            # Declare exchange
            self._exchange = await self._channel.declare_exchange(
                name=self.exchange_name,
                type=ExchangeType(self.exchange_type),
                durable=self.exchange_durable,
            )

            self._connection_state = ConnectionState.CONNECTED

        except Exception as e:
            self._connection_state = ConnectionState.FAILED
            raise ConnectionError(
                f"Failed to connect to RabbitMQ: {e}",
                broker_type=self.broker_type,
            ) from e

    async def disconnect(self) -> None:
        """Close connection to RabbitMQ server."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return

        self._connection_state = ConnectionState.DISCONNECTED

        # Cancel all consumers
        for consumer_tag, consumer in self._consumers.items():
            try:
                await consumer.cancel()
            except Exception:
                pass
        self._consumers.clear()

        # Close channel
        if self._channel and not self._channel.is_closed:
            await self._channel.close()
        self._channel = None

        # Close connection
        if self._connection and not self._connection.is_closed:
            await self._connection.close()
        self._connection = None

        # Clear state
        self._exchange = None
        self._queues.clear()
        self._subscriptions.clear()

    def _build_connection_url(self) -> str:
        """Build RabbitMQ connection URL."""
        scheme = "amqps" if self.ssl else "amqp"

        # URL encode virtual host
        vhost = self.virtual_host.replace("/", "%2F") if self.virtual_host != "/" else ""

        url = f"{scheme}://{self.username}:{self.password}@{self.host}:{self.port}/{vhost}"

        # Add SSL parameters
        if self.ssl:
            ssl_params = []
            if not self.ssl_verify:
                ssl_params.append("verify=no")
            if ssl_params:
                url += "?" + "&".join(ssl_params)

        return url

    async def dispatch(
        self,
        pipe: str,
        data: Any,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """Send a message to the specified routing key."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to RabbitMQ server",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        try:
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

            # Prepare message headers
            message_headers = headers.copy() if headers else {}
            message_headers.update({
                "hybridpipe_message_id": metadata.message_id,
                "hybridpipe_timestamp": metadata.timestamp.isoformat(),
                "hybridpipe_broker_type": str(metadata.broker_type.value),
            })

            # Create AMQP message
            message = Message(
                body=serialized_data,
                headers=message_headers,
                delivery_mode=self.delivery_mode,
                message_id=metadata.message_id,
                timestamp=metadata.timestamp,
            )

            # Publish message
            await self._exchange.publish(
                message=message,
                routing_key=pipe,
            )

            self._messages_published += 1

        except Exception as e:
            self._publish_errors += 1
            raise ProtocolError(
                f"Failed to publish message to RabbitMQ routing key '{pipe}': {e}",
                broker_type=self.broker_type,
                pipe=pipe,
            ) from e

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
        """Subscribe to messages from the specified routing key pattern."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to RabbitMQ server",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        # Add callback to subscriptions
        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
        self._subscriptions[pipe].append(callback)

        # Create queue if not exists
        if pipe not in self._queues:
            await self._create_queue_for_pipe(pipe)

        # Start consuming if not already consuming
        if pipe not in self._consumers:
            await self._start_consuming(pipe)

    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified routing key pattern."""
        if pipe in self._subscriptions:
            del self._subscriptions[pipe]

        # Cancel consumer
        if pipe in self._consumers:
            consumer = self._consumers[pipe]
            await consumer.cancel()
            del self._consumers[pipe]

        # Delete queue if configured for auto-delete
        if pipe in self._queues and self.queue_auto_delete:
            queue = self._queues[pipe]
            await queue.delete()
            del self._queues[pipe]

    async def _create_queue_for_pipe(self, pipe: str) -> None:
        """Create a queue for the specified pipe (routing key pattern)."""
        # Generate unique queue name for this consumer
        queue_name = f"hybridpipe.{pipe}.{uuid.uuid4().hex[:8]}"

        # Declare queue
        queue = await self._channel.declare_queue(
            name=queue_name,
            durable=self.queue_durable,
            auto_delete=self.queue_auto_delete,
            arguments=self._get_queue_arguments(),
        )

        # Bind queue to exchange with routing key pattern
        await queue.bind(self._exchange, routing_key=pipe)

        self._queues[pipe] = queue

    def _get_queue_arguments(self) -> Dict[str, Any]:
        """Get queue arguments from configuration."""
        arguments = {}

        # Message TTL
        message_ttl = self._config.get("message_ttl")
        if message_ttl:
            arguments["x-message-ttl"] = message_ttl

        # Max priority
        max_priority = self._config.get("max_priority")
        if max_priority:
            arguments["x-max-priority"] = max_priority

        # Dead letter exchange
        dead_letter_exchange = self._config.get("dead_letter_exchange")
        if dead_letter_exchange:
            arguments["x-dead-letter-exchange"] = dead_letter_exchange

        # Dead letter routing key
        dead_letter_routing_key = self._config.get("dead_letter_routing_key")
        if dead_letter_routing_key:
            arguments["x-dead-letter-routing-key"] = dead_letter_routing_key

        return arguments

    async def _start_consuming(self, pipe: str) -> None:
        """Start consuming messages from the queue for the specified pipe."""
        if pipe not in self._queues:
            return

        queue = self._queues[pipe]

        # Create message handler
        async def message_handler(message: IncomingMessage) -> None:
            await self._process_message(pipe, message)

        # Start consuming
        consumer = await queue.consume(
            callback=message_handler,
            no_ack=self.auto_ack,
        )

        self._consumers[pipe] = consumer

    async def _process_message(self, pipe: str, message: IncomingMessage) -> None:
        """Process a received RabbitMQ message."""
        if pipe not in self._subscriptions:
            if not self.auto_ack:
                await message.ack()
            return

        try:
            # Extract headers
            headers = {}
            if message.headers:
                for key, value in message.headers.items():
                    if isinstance(value, bytes):
                        headers[key] = value.decode('utf-8', errors='ignore')
                    else:
                        headers[key] = str(value)

            # Create metadata
            metadata = MessageMetadata(
                message_id=headers.get("hybridpipe_message_id", message.message_id or str(uuid.uuid4())),
                timestamp=message.timestamp or datetime.now(),
                pipe=pipe,
                broker_type=self.broker_type,
                serialization_format=SerializationFormat.JSON,
                size_bytes=len(message.body),
                headers=headers,
            )

            # Get message data
            data = message.body

            # Call all subscribers
            callbacks = self._subscriptions[pipe][:]  # Copy to avoid modification during iteration
            for callback in callbacks:
                try:
                    result = callback(data, metadata)
                    if asyncio.iscoroutine(result):
                        await result
                except Exception as e:
                    print(f"Error in RabbitMQ message callback for pipe {pipe}: {e}")

            # Acknowledge message if not auto-ack
            if not self.auto_ack:
                await message.ack()

            self._messages_consumed += 1

        except Exception as e:
            self._consume_errors += 1

            # Reject message if not auto-ack
            if not self.auto_ack:
                await message.reject(requeue=False)

            print(f"Error processing RabbitMQ message for pipe {pipe}: {e}")

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()

        # Add RabbitMQ-specific health information
        uptime = time.time() - self._start_time

        # Get connection info
        connection_info = {}
        if self.is_connected and self._connection:
            try:
                connection_info = {
                    "server_properties": getattr(self._connection, "server_properties", {}),
                    "is_closed": self._connection.is_closed,
                    "channel_count": len(getattr(self._connection, "_channels", {})),
                }
            except Exception:
                connection_info = {"error": "Failed to get connection info"}

        base_health.update({
            "uptime_seconds": uptime,
            "messages_published": self._messages_published,
            "messages_consumed": self._messages_consumed,
            "publish_errors": self._publish_errors,
            "consume_errors": self._consume_errors,
            "subscribed_pipes": list(self._subscriptions.keys()),
            "subscription_count": len(self._subscriptions),
            "queue_count": len(self._queues),
            "consumer_count": len(self._consumers),
            "connection_info": connection_info,
            "config": {
                "host": self.host,
                "port": self.port,
                "virtual_host": self.virtual_host,
                "username": self.username,
                "ssl": self.ssl,
                "exchange_name": self.exchange_name,
                "exchange_type": self.exchange_type,
                "queue_durable": self.queue_durable,
                "delivery_mode": str(self.delivery_mode),
            },
        })

        return base_health

    # Exchange and queue management methods
    async def create_exchange(
        self,
        name: str,
        exchange_type: str = "topic",
        durable: bool = True,
        auto_delete: bool = False,
        arguments: Optional[Dict[str, Any]] = None,
    ) -> None:
        """Create a RabbitMQ exchange."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to RabbitMQ server",
                broker_type=self.broker_type,
            )

        await self._channel.declare_exchange(
            name=name,
            type=ExchangeType(exchange_type),
            durable=durable,
            auto_delete=auto_delete,
            arguments=arguments or {},
        )

    async def delete_exchange(self, name: str) -> None:
        """Delete a RabbitMQ exchange."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to RabbitMQ server",
                broker_type=self.broker_type,
            )

        exchange = await self._channel.get_exchange(name)
        await exchange.delete()

    async def create_queue(
        self,
        name: str,
        durable: bool = True,
        auto_delete: bool = False,
        arguments: Optional[Dict[str, Any]] = None,
    ) -> None:
        """Create a RabbitMQ queue."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to RabbitMQ server",
                broker_type=self.broker_type,
            )

        await self._channel.declare_queue(
            name=name,
            durable=durable,
            auto_delete=auto_delete,
            arguments=arguments or {},
        )

    async def delete_queue(self, name: str) -> None:
        """Delete a RabbitMQ queue."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to RabbitMQ server",
                broker_type=self.broker_type,
            )

        queue = await self._channel.get_queue(name)
        await queue.delete()

    async def bind_queue(
        self,
        queue_name: str,
        exchange_name: str,
        routing_key: str,
        arguments: Optional[Dict[str, Any]] = None,
    ) -> None:
        """Bind a queue to an exchange with a routing key."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to RabbitMQ server",
                broker_type=self.broker_type,
            )

        queue = await self._channel.get_queue(queue_name)
        exchange = await self._channel.get_exchange(exchange_name)
        await queue.bind(exchange, routing_key=routing_key, arguments=arguments or {})
