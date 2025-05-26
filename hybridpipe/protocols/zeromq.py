"""
ZeroMQ protocol implementation for HybridPipe.

This module provides ZeroMQ messaging support with multiple patterns,
ultra-low latency messaging, and brokerless architecture.
"""

import asyncio
import time
import uuid
from typing import Any, Dict, List, Optional, Set, Callable, Union
from datetime import datetime
from enum import Enum
import threading

try:
    import zmq
    import zmq.asyncio
    ZMQ_AVAILABLE = True
except ImportError:
    ZMQ_AVAILABLE = False

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
)
from hybridpipe.core.decorators import protocol_implementation
from hybridpipe.serialization.engine import encode, decode


class ZMQPattern(Enum):
    """ZeroMQ messaging patterns."""
    PUB_SUB = "PUB_SUB"
    REQ_REP = "REQ_REP"
    PUSH_PULL = "PUSH_PULL"
    PAIR = "PAIR"
    ROUTER_DEALER = "ROUTER_DEALER"


@protocol_implementation(
    BrokerType.ZEROMQ,
    default_config={
        "pattern": "PUB_SUB",
        "bind_addresses": [],
        "connect_addresses": [],
        "socket_type": None,  # Auto-determined from pattern
        "high_water_mark": 1000,
        "linger": 0,
        "receive_timeout": 1000,  # milliseconds
        "send_timeout": 1000,  # milliseconds
        "identity": None,
        "subscribe_filters": [],  # For SUB sockets
        "conflate": False,  # Keep only last message
        "immediate": False,  # Queue messages only to completed connections
        "tcp_keepalive": True,
        "tcp_keepalive_idle": 300,
        "tcp_keepalive_interval": 300,
        "tcp_keepalive_count": 3,
        "max_sockets": 1024,
        "io_threads": 1,
    },
    metadata={
        "description": "ZeroMQ high-performance messaging with multiple patterns",
        "supports_persistence": False,
        "supports_clustering": False,
        "supports_streaming": True,
        "supports_request_reply": True,
        "supports_brokerless": True,
        "max_throughput_mps": 1000000,
        "min_latency_us": 10,
    },
)
class ZeroMQHybridPipe(HybridPipe):
    """
    ZeroMQ implementation of HybridPipe.

    This implementation provides ZeroMQ messaging functionality including:
    - Multiple messaging patterns (PUB/SUB, REQ/REP, PUSH/PULL, PAIR)
    - Ultra-low latency messaging (<10μs)
    - Brokerless architecture
    - High throughput (1M+ messages/second)
    - Built-in load balancing and failover
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize ZeroMQ HybridPipe implementation.

        Args:
            config: Configuration dictionary with ZeroMQ-specific options
        """
        if not ZMQ_AVAILABLE:
            raise ImportError(
                "pyzmq is required for ZeroMQ support. "
                "Install with: pip install pyzmq"
            )

        super().__init__(config)

        # ZeroMQ configuration with Go compatibility
        self.pattern = ZMQPattern(self._config.get("pattern", "PUB_SUB"))

        # Parse addresses with Go compatibility
        self.bind_addresses, self.connect_addresses = self._parse_addresses_config()

        self.socket_type = self._config.get("socket_type")

        # Handle Go-style HWM configuration
        self.high_water_mark = self._config.get(
            "high_water_mark",
            self._config.get("publisher_hwm", self._config.get("subscriber_hwm", 1000))
        )

        # Handle Go-style linger configuration
        self.linger = self._config.get(
            "linger",
            self._config.get("publisher_linger", self._config.get("subscriber_linger", 0))
        )

        self.receive_timeout = self._config.get("receive_timeout", 1000)
        self.send_timeout = self._config.get("send_timeout", 1000)
        self.identity = self._config.get("identity")
        self.subscribe_filters = self._config.get("subscribe_filters", [])
        self.conflate = self._config.get("conflate", False)
        self.immediate = self._config.get("immediate", False)

        # TCP keepalive settings
        self.tcp_keepalive = self._config.get("tcp_keepalive", True)
        self.tcp_keepalive_idle = self._config.get("tcp_keepalive_idle", 300)
        self.tcp_keepalive_interval = self._config.get("tcp_keepalive_interval", 300)
        self.tcp_keepalive_count = self._config.get("tcp_keepalive_count", 3)

        # Context settings
        self.max_sockets = self._config.get("max_sockets", 1024)
        self.io_threads = self._config.get("io_threads", 1)

        # Connection state
        self._connection_state = ConnectionState.DISCONNECTED
        self._context: Optional[zmq.asyncio.Context] = None
        self._socket: Optional[zmq.asyncio.Socket] = None

        # Subscription management
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._running = False
        self._receive_task: Optional[asyncio.Task] = None

        # Request-reply state
        self._pending_requests: Dict[str, asyncio.Future] = {}
        self._request_handlers: Dict[str, Callable] = {}

        # Metrics
        self._messages_published = 0
        self._messages_received = 0
        self._requests_sent = 0
        self._requests_received = 0
        self._publish_errors = 0
        self._receive_errors = 0
        self._start_time = time.time()

    def _parse_addresses_config(self) -> tuple[List[str], List[str]]:
        """Parse address configuration with Go compatibility."""
        bind_addresses = []
        connect_addresses = []

        # Python-style configuration
        if "bind_addresses" in self._config:
            bind_addresses = self._config["bind_addresses"]
        if "connect_addresses" in self._config:
            connect_addresses = self._config["connect_addresses"]

        # Go-style configuration
        if "publisher_endpoint" in self._config or "subscriber_endpoint" in self._config:
            pub_endpoint = self._config.get("publisher_endpoint", "tcp://127.0.0.1:5555")
            sub_endpoint = self._config.get("subscriber_endpoint", "tcp://127.0.0.1:5556")

            pub_bind = self._config.get("publisher_bind", False)
            sub_bind = self._config.get("subscriber_bind", False)

            # Determine role based on pattern and bind flags
            if self.pattern == ZMQPattern.PUB_SUB:
                if pub_bind:
                    bind_addresses.append(pub_endpoint)
                else:
                    connect_addresses.append(pub_endpoint)

                if sub_bind:
                    bind_addresses.append(sub_endpoint)
                else:
                    connect_addresses.append(sub_endpoint)
            else:
                # For other patterns, use publisher endpoint as primary
                if pub_bind:
                    bind_addresses.append(pub_endpoint)
                else:
                    connect_addresses.append(pub_endpoint)

        # Legacy Go-style with separate host/port/protocol
        if not bind_addresses and not connect_addresses:
            pub_host = self._config.get("publisher_host", "127.0.0.1")
            pub_port = self._config.get("publisher_port", 5555)
            pub_protocol = self._config.get("publisher_protocol", "tcp")
            pub_bind = self._config.get("publisher_bind", False)

            endpoint = f"{pub_protocol}://{pub_host}:{pub_port}"
            if pub_bind:
                bind_addresses.append(endpoint)
            else:
                connect_addresses.append(endpoint)

        return bind_addresses, connect_addresses

    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.ZEROMQ

    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=False,  # ZMQ is in-memory only
            supports_transactions=False,  # No traditional transactions
            supports_clustering=False,  # No built-in clustering
            supports_compression=False,  # No protocol-level compression
            supports_encryption=True,  # CurveZMQ encryption available
            supports_authentication=True,  # CurveZMQ authentication
            supports_authorization=False,  # No built-in authorization
            supports_dead_letter=False,  # No dead letter queues
            supports_message_ordering=True,  # FIFO ordering within connections
            supports_exactly_once=False,  # At-most-once delivery
            max_message_size=None,  # No hard limit
            max_pipe_length=None,  # No specific limit
        )

    def _get_socket_type(self) -> int:
        """Get ZMQ socket type based on pattern and role."""
        if self.socket_type:
            return getattr(zmq, self.socket_type)

        pattern_map = {
            ZMQPattern.PUB_SUB: zmq.PUB if self.bind_addresses else zmq.SUB,
            ZMQPattern.REQ_REP: zmq.REP if self.bind_addresses else zmq.REQ,
            ZMQPattern.PUSH_PULL: zmq.PULL if self.bind_addresses else zmq.PUSH,
            ZMQPattern.PAIR: zmq.PAIR,
            ZMQPattern.ROUTER_DEALER: zmq.ROUTER if self.bind_addresses else zmq.DEALER,
        }

        return pattern_map.get(self.pattern, zmq.PUB)

    async def connect(self) -> None:
        """Establish ZeroMQ socket connections."""
        if self._connection_state == ConnectionState.CONNECTED:
            return

        self._connection_state = ConnectionState.CONNECTING

        try:
            # Create ZMQ context
            self._context = zmq.asyncio.Context(io_threads=self.io_threads)
            self._context.max_sockets = self.max_sockets

            # Create socket
            socket_type = self._get_socket_type()
            self._socket = self._context.socket(socket_type)

            # Configure socket options
            self._socket.setsockopt(zmq.SNDHWM, self.high_water_mark)
            self._socket.setsockopt(zmq.RCVHWM, self.high_water_mark)
            self._socket.setsockopt(zmq.LINGER, self.linger)
            self._socket.setsockopt(zmq.RCVTIMEO, self.receive_timeout)
            self._socket.setsockopt(zmq.SNDTIMEO, self.send_timeout)

            if self.identity:
                self._socket.setsockopt_string(zmq.IDENTITY, self.identity)

            if self.conflate:
                self._socket.setsockopt(zmq.CONFLATE, 1)

            if self.immediate:
                self._socket.setsockopt(zmq.IMMEDIATE, 1)

            # TCP keepalive settings
            if self.tcp_keepalive:
                self._socket.setsockopt(zmq.TCP_KEEPALIVE, 1)
                self._socket.setsockopt(zmq.TCP_KEEPALIVE_IDLE, self.tcp_keepalive_idle)
                self._socket.setsockopt(zmq.TCP_KEEPALIVE_INTVL, self.tcp_keepalive_interval)
                self._socket.setsockopt(zmq.TCP_KEEPALIVE_CNT, self.tcp_keepalive_count)

            # Bind to addresses
            for address in self.bind_addresses:
                self._socket.bind(address)

            # Connect to addresses
            for address in self.connect_addresses:
                self._socket.connect(address)

            # Set up subscriptions for SUB socket
            if socket_type == zmq.SUB:
                if self.subscribe_filters:
                    for filter_str in self.subscribe_filters:
                        self._socket.setsockopt_string(zmq.SUBSCRIBE, filter_str)
                else:
                    # Subscribe to all messages
                    self._socket.setsockopt(zmq.SUBSCRIBE, b"")

            self._connection_state = ConnectionState.CONNECTED
            self._running = True

            # Start receive loop for patterns that need it
            if socket_type in [zmq.SUB, zmq.PULL, zmq.REP, zmq.DEALER, zmq.ROUTER, zmq.PAIR]:
                self._receive_task = asyncio.create_task(self._receive_loop())

        except Exception as e:
            self._connection_state = ConnectionState.FAILED
            raise ConnectionError(
                f"Failed to connect ZeroMQ socket: {e}",
                broker_type=self.broker_type,
            ) from e

    async def disconnect(self) -> None:
        """Close ZeroMQ socket connections."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return

        self._connection_state = ConnectionState.DISCONNECTED
        self._running = False

        try:
            # Cancel receive task
            if self._receive_task:
                self._receive_task.cancel()
                try:
                    await self._receive_task
                except asyncio.CancelledError:
                    pass

            # Close socket
            if self._socket:
                self._socket.close()

            # Terminate context
            if self._context:
                self._context.term()

            # Clear state
            self._subscriptions.clear()
            self._pending_requests.clear()
            self._request_handlers.clear()
            self._socket = None
            self._context = None

        except Exception as e:
            print(f"Error during ZeroMQ disconnect: {e}")

    async def dispatch(
        self,
        pipe: str,
        data: Any,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """Send a message via ZeroMQ socket."""
        if not self.is_connected or not self._socket:
            raise ConnectionError(
                "ZeroMQ socket not connected",
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

            # Prepare message
            message_data = {
                "pipe": pipe,
                "data": data,
                "metadata": {
                    "message_id": metadata.message_id,
                    "timestamp": metadata.timestamp.isoformat(),
                    "broker_type": metadata.broker_type.value,
                    "headers": metadata.headers,
                }
            }

            # Serialize the message
            serialized_data = encode(message_data, SerializationFormat.JSON)
            metadata.size_bytes = len(serialized_data)

            # Send message based on socket type
            socket_type = self._socket.socket_type

            if socket_type in [zmq.PUB, zmq.PUSH, zmq.PAIR]:
                # Simple send
                await self._socket.send(serialized_data)
            elif socket_type == zmq.REQ:
                # Request-reply pattern
                await self._socket.send(serialized_data)
                # Wait for reply (handled in request method)
            elif socket_type in [zmq.DEALER, zmq.ROUTER]:
                # Multi-part message for DEALER/ROUTER
                if headers and "routing_id" in headers:
                    await self._socket.send_multipart([
                        headers["routing_id"].encode(),
                        serialized_data
                    ])
                else:
                    await self._socket.send(serialized_data)
            else:
                await self._socket.send(serialized_data)

            self._messages_published += 1

        except Exception as e:
            self._publish_errors += 1
            if isinstance(e, (ConnectionError, ProtocolError, TimeoutError)):
                raise
            raise ProtocolError(
                f"Failed to send ZeroMQ message to pipe '{pipe}': {e}",
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
                f"ZeroMQ message dispatch timed out after {timeout_seconds} seconds",
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
        """Subscribe to messages from the specified pipe pattern."""
        if not self.is_connected:
            raise ConnectionError(
                "ZeroMQ socket not connected",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        # Add callback to subscriptions
        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
        self._subscriptions[pipe].append(callback)

        # For SUB sockets, add subscription filter if needed
        if self._socket and self._socket.socket_type == zmq.SUB:
            if pipe not in self.subscribe_filters:
                self._socket.setsockopt_string(zmq.SUBSCRIBE, pipe)

    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified pipe pattern."""
        if pipe in self._subscriptions:
            del self._subscriptions[pipe]

        # For SUB sockets, remove subscription filter
        if self._socket and self._socket.socket_type == zmq.SUB:
            try:
                self._socket.setsockopt_string(zmq.UNSUBSCRIBE, pipe)
            except Exception as e:
                print(f"ZeroMQ unsubscribe failed for pipe {pipe}: {e}")

    async def _receive_loop(self) -> None:
        """Main receive loop for processing incoming messages."""
        while self._running and self._socket:
            try:
                # Receive message
                if self._socket.socket_type in [zmq.DEALER, zmq.ROUTER]:
                    # Multi-part message
                    parts = await self._socket.recv_multipart()
                    if len(parts) >= 2:
                        routing_id = parts[0]
                        message_data = parts[1]
                    else:
                        message_data = parts[0] if parts else b""
                        routing_id = None
                else:
                    # Single-part message
                    message_data = await self._socket.recv()
                    routing_id = None

                # Process message
                await self._process_message(message_data, routing_id)

            except zmq.Again:
                # Timeout, continue
                continue
            except Exception as e:
                if self._running:
                    self._receive_errors += 1
                    print(f"Error in ZeroMQ receive loop: {e}")
                    await asyncio.sleep(0.1)  # Brief pause before retry

    async def _process_message(self, message_data: bytes, routing_id: Optional[bytes] = None) -> None:
        """Process a received ZeroMQ message."""
        try:
            # Decode message
            try:
                message = decode(message_data, SerializationFormat.JSON)
                pipe = message.get("pipe", "")
                data = message.get("data")
                metadata_dict = message.get("metadata", {})
            except Exception:
                # Fallback to raw message
                pipe = ""
                data = message_data
                metadata_dict = {}

            # Create metadata
            metadata = MessageMetadata(
                message_id=metadata_dict.get("message_id", str(uuid.uuid4())),
                timestamp=datetime.fromisoformat(
                    metadata_dict.get("timestamp", datetime.now().isoformat())
                ),
                pipe=pipe,
                broker_type=self.broker_type,
                serialization_format=SerializationFormat.JSON,
                size_bytes=len(message_data),
                headers=metadata_dict.get("headers", {}),
            )

            # Add routing ID to headers if present
            if routing_id:
                metadata.headers["routing_id"] = routing_id.decode()

            # Handle based on socket type
            socket_type = self._socket.socket_type

            if socket_type == zmq.REP:
                # Request-reply server
                await self._handle_request(data, metadata)
            elif socket_type == zmq.REQ:
                # Request-reply client (handle reply)
                await self._handle_reply(data, metadata)
            else:
                # Regular message processing
                await self._handle_message(data, metadata, pipe)

            self._messages_received += 1

        except Exception as e:
            self._receive_errors += 1
            print(f"Error processing ZeroMQ message: {e}")

    async def _handle_message(self, data: Any, metadata: MessageMetadata, pipe: str) -> None:
        """Handle regular messages."""
        # Find matching subscriptions
        matching_subscriptions = []
        for subscribed_pipe, callbacks in self._subscriptions.items():
            if self._pipe_matches(pipe, subscribed_pipe):
                matching_subscriptions.extend(callbacks)

        # Call all matching subscribers
        for callback in matching_subscriptions:
            try:
                if isinstance(data, bytes):
                    result = callback(data, metadata)
                else:
                    result = callback(encode(data, SerializationFormat.JSON), metadata)

                if asyncio.iscoroutine(result):
                    await result
            except Exception as e:
                print(f"Error in ZeroMQ message callback for pipe {pipe}: {e}")

    async def _handle_request(self, data: Any, metadata: MessageMetadata) -> None:
        """Handle request in REP socket."""
        try:
            # Find request handler
            pipe = metadata.pipe
            if pipe in self._request_handlers:
                handler = self._request_handlers[pipe]

                # Call handler
                if isinstance(data, bytes):
                    result = handler(data, metadata)
                else:
                    result = handler(encode(data, SerializationFormat.JSON), metadata)

                if asyncio.iscoroutine(result):
                    result = await result

                # Send reply
                if result is not None:
                    if isinstance(result, bytes):
                        reply_data = result
                    else:
                        reply_data = encode(result, SerializationFormat.JSON)

                    await self._socket.send(reply_data)
                else:
                    # Send empty reply
                    await self._socket.send(b"")

                self._requests_received += 1

        except Exception as e:
            print(f"Error handling ZeroMQ request: {e}")
            # Send error reply
            error_response = encode({"error": str(e)}, SerializationFormat.JSON)
            await self._socket.send(error_response)

    async def _handle_reply(self, data: Any, metadata: MessageMetadata) -> None:
        """Handle reply in REQ socket."""
        # Find pending request
        request_id = metadata.message_id
        if request_id in self._pending_requests:
            future = self._pending_requests.pop(request_id)
            if not future.done():
                future.set_result(data)

    def _pipe_matches(self, pipe: str, pattern: str) -> bool:
        """Check if a pipe matches a subscription pattern."""
        # Simple string matching for now
        # Could be extended with wildcards if needed
        return pipe == pattern or pattern == "" or pattern == "*"

    # ZeroMQ-specific methods

    async def request(
        self,
        pipe: str,
        data: Any,
        timeout: float = 5.0,
        headers: Optional[Dict[str, Any]] = None,
    ) -> Any:
        """Send a request and wait for a reply (REQ/REP pattern)."""
        if not self.is_connected or not self._socket:
            raise ConnectionError(
                "ZeroMQ socket not connected",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        if self._socket.socket_type != zmq.REQ:
            raise ProtocolError("Request method only available for REQ sockets")

        try:
            # Create request ID
            request_id = str(uuid.uuid4())

            # Create future for reply
            reply_future = asyncio.Future()
            self._pending_requests[request_id] = reply_future

            # Send request
            await self.dispatch(pipe, data, headers)
            self._requests_sent += 1

            # Wait for reply
            try:
                reply = await asyncio.wait_for(reply_future, timeout=timeout)
                return reply
            except asyncio.TimeoutError:
                self._pending_requests.pop(request_id, None)
                raise TimeoutError(
                    f"ZeroMQ request timed out after {timeout} seconds",
                    timeout_seconds=timeout,
                    operation="request",
                    broker_type=self.broker_type,
                    pipe=pipe,
                )

        except Exception as e:
            if isinstance(e, (ConnectionError, ProtocolError, TimeoutError)):
                raise
            raise ProtocolError(
                f"ZeroMQ request failed for pipe '{pipe}': {e}",
                broker_type=self.broker_type,
                pipe=pipe,
            ) from e

    async def reply(
        self,
        pipe: str,
        callback: Callable[[bytes, MessageMetadata], Union[bytes, Any]],
    ) -> None:
        """Handle requests on a pipe (REP pattern)."""
        if not self.is_connected:
            raise ConnectionError(
                "ZeroMQ socket not connected",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        if self._socket.socket_type != zmq.REP:
            raise ProtocolError("Reply method only available for REP sockets")

        # Store request handler
        self._request_handlers[pipe] = callback

    async def bind(self, address: str) -> None:
        """Bind socket to an address."""
        if not self._socket:
            raise ConnectionError("ZeroMQ socket not initialized")

        try:
            self._socket.bind(address)
            if address not in self.bind_addresses:
                self.bind_addresses.append(address)
        except Exception as e:
            raise ProtocolError(f"Failed to bind to address '{address}': {e}") from e

    async def connect_to(self, address: str) -> None:
        """Connect socket to an address."""
        if not self._socket:
            raise ConnectionError("ZeroMQ socket not initialized")

        try:
            self._socket.connect(address)
            if address not in self.connect_addresses:
                self.connect_addresses.append(address)
        except Exception as e:
            raise ProtocolError(f"Failed to connect to address '{address}': {e}") from e

    async def set_pattern(self, pattern: ZMQPattern) -> None:
        """Change messaging pattern (requires reconnection)."""
        if self.is_connected:
            await self.disconnect()

        self.pattern = pattern
        await self.connect()

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()

        # Add ZeroMQ-specific health information
        uptime = time.time() - self._start_time

        # Get socket info
        socket_info = {}
        if self.is_connected and self._socket:
            try:
                socket_info = {
                    "socket_type": self._socket.socket_type,
                    "socket_type_name": {
                        zmq.PUB: "PUB", zmq.SUB: "SUB",
                        zmq.REQ: "REQ", zmq.REP: "REP",
                        zmq.PUSH: "PUSH", zmq.PULL: "PULL",
                        zmq.PAIR: "PAIR",
                        zmq.ROUTER: "ROUTER", zmq.DEALER: "DEALER",
                    }.get(self._socket.socket_type, "UNKNOWN"),
                    "hwm_send": self._socket.getsockopt(zmq.SNDHWM),
                    "hwm_recv": self._socket.getsockopt(zmq.RCVHWM),
                    "linger": self._socket.getsockopt(zmq.LINGER),
                }
            except Exception:
                socket_info = {"error": "Failed to get socket info"}

        base_health.update({
            "uptime_seconds": uptime,
            "messages_published": self._messages_published,
            "messages_received": self._messages_received,
            "requests_sent": self._requests_sent,
            "requests_received": self._requests_received,
            "publish_errors": self._publish_errors,
            "receive_errors": self._receive_errors,
            "subscribed_pipes": list(self._subscriptions.keys()),
            "subscription_count": len(self._subscriptions),
            "pending_requests": len(self._pending_requests),
            "request_handlers": list(self._request_handlers.keys()),
            "pattern": self.pattern.value,
            "bind_addresses": self.bind_addresses,
            "connect_addresses": self.connect_addresses,
            "socket_info": socket_info,
            "config": {
                "pattern": self.pattern.value,
                "high_water_mark": self.high_water_mark,
                "linger": self.linger,
                "receive_timeout": self.receive_timeout,
                "send_timeout": self.send_timeout,
            },
        })

        return base_health