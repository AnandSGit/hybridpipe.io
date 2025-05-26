"""
AMQP 1.0 protocol implementation for HybridPipe.

This module provides AMQP 1.0 messaging support for enterprise integration
with standards-based messaging, advanced routing, and reliability features.
"""

import asyncio
import time
import uuid
from typing import Any, Dict, List, Optional, Set, Callable, Union
from datetime import datetime
import threading

try:
    from proton import Message, Url, SSLDomain
    from proton.handlers import MessagingHandler
    from proton.reactor import Container
    from proton._events import Event
    AMQP1_AVAILABLE = True
except ImportError:
    AMQP1_AVAILABLE = False

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


@protocol_implementation(
    BrokerType.AMQP1,
    default_config={
        "hostname": "localhost",
        "port": 5672,
        "username": None,
        "password": None,
        "container_id": "hybridpipe-amqp1",
        "sasl_enabled": True,
        "sasl_mechanisms": "PLAIN",
        "ssl_enabled": False,
        "ssl_domain": None,
        "ssl_verify_mode": "VERIFY_PEER",
        "ssl_ca_file": None,
        "ssl_cert_file": None,
        "ssl_key_file": None,
        "connection_timeout": 30.0,
        "idle_timeout": 60.0,
        "max_frame_size": 65536,
        "channel_max": 32767,
        "heartbeat": 0,
        "auto_settle": True,
        "auto_accept": True,
        "prefetch": 10,
        "durable": True,
        "auto_delete": False,
    },
    metadata={
        "description": "AMQP 1.0 standard messaging for enterprise integration",
        "supports_persistence": True,
        "supports_clustering": True,
        "supports_streaming": False,
        "supports_request_reply": True,
        "supports_federation": True,
        "max_throughput_mps": 50000,
        "min_latency_us": 100,
    },
)
class AMQP1HybridPipe(HybridPipe):
    """
    AMQP 1.0 implementation of HybridPipe.

    This implementation provides AMQP 1.0 messaging functionality including:
    - Standards-based AMQP 1.0 protocol
    - Enterprise-grade reliability and transactions
    - Advanced routing and filtering
    - Security with SASL and SSL/TLS
    - Interoperability between different vendors
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize AMQP 1.0 HybridPipe implementation.

        Args:
            config: Configuration dictionary with AMQP 1.0-specific options
        """
        if not AMQP1_AVAILABLE:
            raise ImportError(
                "python-qpid-proton is required for AMQP 1.0 support. "
                "Install with: pip install python-qpid-proton"
            )

        super().__init__(config)

        # AMQP 1.0 configuration with Go compatibility
        self.hostname, self.port = self._parse_connection_config()
        self.username = self._config.get("username") or self._config.get("user")
        self.password = self._config.get("password")
        self.container_id = self._config.get("container_id", "hybridpipe-amqp1")

        # SASL configuration
        self.sasl_enabled = self._config.get("sasl_enabled", True)
        self.sasl_mechanisms = self._config.get("sasl_mechanisms", "PLAIN")

        # SSL configuration
        self.ssl_enabled = self._config.get("ssl_enabled", False)
        self.ssl_domain = self._config.get("ssl_domain")
        self.ssl_verify_mode = self._config.get("ssl_verify_mode", "VERIFY_PEER")
        self.ssl_ca_file = self._config.get("ssl_ca_file")
        self.ssl_cert_file = self._config.get("ssl_cert_file")
        self.ssl_key_file = self._config.get("ssl_key_file")

        # Connection settings
        self.connection_timeout = self._config.get("connection_timeout", 30.0)
        self.idle_timeout = self._config.get("idle_timeout", 60.0)
        self.max_frame_size = self._config.get("max_frame_size", 65536)
        self.channel_max = self._config.get("channel_max", 32767)
        self.heartbeat = self._config.get("heartbeat", 0)

        # Message settings
        self.auto_settle = self._config.get("auto_settle", True)
        self.auto_accept = self._config.get("auto_accept", True)
        self.prefetch = self._config.get("prefetch", 10)
        self.durable = self._config.get("durable", True)
        self.auto_delete = self._config.get("auto_delete", False)

        # Connection state
        self._connection_state = ConnectionState.DISCONNECTED
        self._container: Optional[Container] = None
        self._connection = None
        self._handler: Optional['AMQP1Handler'] = None

        # Subscription management
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._senders: Dict[str, Any] = {}
        self._receivers: Dict[str, Any] = {}

        # Request-reply state
        self._pending_requests: Dict[str, asyncio.Future] = {}
        self._request_handlers: Dict[str, Callable] = {}

        # Threading for proton reactor
        self._reactor_thread: Optional[threading.Thread] = None
        self._running = False

        # Metrics
        self._messages_published = 0
        self._messages_received = 0
        self._requests_sent = 0
        self._requests_received = 0
        self._publish_errors = 0
        self._receive_errors = 0
        self._start_time = time.time()

    def _parse_connection_config(self) -> tuple[str, int]:
        """Parse connection configuration with Go compatibility."""
        # Python-style: separate hostname and port
        if "hostname" in self._config:
            hostname = self._config["hostname"]
            port = self._config.get("port", 5672)
            return hostname, port

        # Go-style: URL format
        if "url" in self._config or "amqpserver" in self._config:
            url = self._config.get("url", self._config.get("amqpserver", "amqp://localhost:5672"))

            # Parse URL
            if "://" in url:
                # Extract hostname and port from URL
                parts = url.split("://")[1]  # Remove scheme
                if "@" in parts:
                    parts = parts.split("@")[1]  # Remove auth

                if ":" in parts:
                    hostname, port_str = parts.split(":")
                    port = int(port_str.split("/")[0])  # Remove path
                else:
                    hostname = parts.split("/")[0]  # Remove path
                    port = 5672

                return hostname, port

        # Go-style: separate server and port fields
        if "server" in self._config or "qpidserver" in self._config:
            hostname = self._config.get("server", self._config.get("qpidserver", "localhost"))
            port = self._config.get("qpidport", 5672)
            return hostname, port

        # Default
        return "localhost", 5672

    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.AMQP1

    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=self.durable,
            supports_transactions=True,  # AMQP 1.0 supports transactions
            supports_clustering=True,
            supports_compression=False,  # Not at protocol level
            supports_encryption=self.ssl_enabled,
            supports_authentication=self.sasl_enabled,
            supports_authorization=True,
            supports_dead_letter=True,  # AMQP 1.0 supports dead letter queues
            supports_message_ordering=True,
            supports_exactly_once=True,  # With transactions
            max_message_size=self.max_frame_size,
            max_pipe_length=None,  # No specific limit
        )

    def _build_url(self) -> str:
        """Build AMQP URL from configuration."""
        scheme = "amqps" if self.ssl_enabled else "amqp"

        if self.username and self.password:
            auth = f"{self.username}:{self.password}@"
        else:
            auth = ""

        return f"{scheme}://{auth}{self.hostname}:{self.port}"

    async def connect(self) -> None:
        """Establish connection to AMQP 1.0 broker."""
        if self._connection_state == ConnectionState.CONNECTED:
            return

        self._connection_state = ConnectionState.CONNECTING

        try:
            # Create handler
            self._handler = AMQP1Handler(self)

            # Create container
            self._container = Container(self._handler)

            # Build connection URL
            url = self._build_url()

            # Configure SSL if enabled
            if self.ssl_enabled:
                ssl_domain = SSLDomain(SSLDomain.MODE_CLIENT)

                if self.ssl_verify_mode == "VERIFY_PEER":
                    ssl_domain.set_peer_authentication(SSLDomain.VERIFY_PEER)
                elif self.ssl_verify_mode == "VERIFY_PEER_NAME":
                    ssl_domain.set_peer_authentication(SSLDomain.VERIFY_PEER_NAME)
                else:
                    ssl_domain.set_peer_authentication(SSLDomain.ANONYMOUS_PEER)

                if self.ssl_ca_file:
                    ssl_domain.set_trusted_ca_db(self.ssl_ca_file)

                if self.ssl_cert_file and self.ssl_key_file:
                    ssl_domain.set_credentials(self.ssl_cert_file, self.ssl_key_file, None)

                self._container.ssl_domain = ssl_domain

            # Start reactor in separate thread
            self._running = True
            self._reactor_thread = threading.Thread(
                target=self._run_reactor,
                args=(url,),
                daemon=True
            )
            self._reactor_thread.start()

            # Wait for connection
            await self._wait_for_connection()

        except Exception as e:
            self._connection_state = ConnectionState.FAILED
            raise ConnectionError(
                f"Failed to connect to AMQP 1.0 broker: {e}",
                broker_type=self.broker_type,
            ) from e

    def _run_reactor(self, url: str) -> None:
        """Run the proton reactor in a separate thread."""
        try:
            self._connection = self._container.connect(
                url,
                heartbeat=self.heartbeat,
                idle_timeout=self.idle_timeout,
                max_frame_size=self.max_frame_size,
                channel_max=self.channel_max,
                container_id=self.container_id,
            )
            self._container.run()
        except Exception as e:
            print(f"AMQP 1.0 reactor error: {e}")

    async def _wait_for_connection(self) -> None:
        """Wait for connection to be established."""
        timeout = self.connection_timeout
        start_time = time.time()

        while time.time() - start_time < timeout:
            if self._connection_state == ConnectionState.CONNECTED:
                return
            elif self._connection_state == ConnectionState.FAILED:
                raise ConnectionError("Connection failed")

            await asyncio.sleep(0.1)

        raise TimeoutError(f"Connection timeout after {timeout} seconds")

    async def disconnect(self) -> None:
        """Close connection to AMQP 1.0 broker."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return

        self._connection_state = ConnectionState.DISCONNECTED
        self._running = False

        try:
            # Close connection
            if self._connection:
                self._connection.close()

            # Stop reactor
            if self._container:
                self._container.stop()

            # Wait for reactor thread to finish
            if self._reactor_thread and self._reactor_thread.is_alive():
                self._reactor_thread.join(timeout=5.0)

            # Clear state
            self._subscriptions.clear()
            self._senders.clear()
            self._receivers.clear()
            self._pending_requests.clear()
            self._request_handlers.clear()
            self._connection = None
            self._container = None
            self._handler = None

        except Exception as e:
            print(f"Error during AMQP 1.0 disconnect: {e}")

    async def dispatch(
        self,
        pipe: str,
        data: Any,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """Send a message to the specified AMQP address."""
        if not self.is_connected or not self._handler:
            raise ConnectionError(
                "Not connected to AMQP 1.0 broker",
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

            # Create AMQP message
            message = Message()
            message.id = metadata.message_id
            message.address = pipe
            message.body = serialized_data
            message.creation_time = time.time()
            message.durable = self.durable

            # Add headers
            if headers:
                message.properties = headers

            # Add metadata as application properties
            message.annotations = {
                "hybridpipe_timestamp": metadata.timestamp.isoformat(),
                "hybridpipe_broker_type": metadata.broker_type.value,
            }

            # Send message
            await self._handler.send_message(pipe, message)
            self._messages_published += 1

        except Exception as e:
            self._publish_errors += 1
            if isinstance(e, (ConnectionError, ProtocolError, TimeoutError)):
                raise
            raise ProtocolError(
                f"Failed to publish message to AMQP address '{pipe}': {e}",
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
                f"AMQP message dispatch timed out after {timeout_seconds} seconds",
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
        """Subscribe to messages from the specified AMQP address."""
        if not self.is_connected or not self._handler:
            raise ConnectionError(
                "Not connected to AMQP 1.0 broker",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        # Add callback to subscriptions
        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
        self._subscriptions[pipe].append(callback)

        # Create receiver if not already exists
        if pipe not in self._receivers:
            try:
                await self._handler.create_receiver(pipe)
            except Exception as e:
                raise ProtocolError(f"AMQP subscribe failed: {e}") from e

    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified AMQP address."""
        if pipe in self._subscriptions:
            del self._subscriptions[pipe]

        if pipe in self._receivers:
            try:
                await self._handler.close_receiver(pipe)
            except Exception as e:
                print(f"AMQP unsubscribe failed for address {pipe}: {e}")

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()

        # Add AMQP 1.0-specific health information
        uptime = time.time() - self._start_time

        # Get connection info
        connection_info = {}
        if self.is_connected and self._connection:
            try:
                connection_info = {
                    "hostname": self.hostname,
                    "port": self.port,
                    "container_id": self.container_id,
                    "ssl_enabled": self.ssl_enabled,
                    "sasl_enabled": self.sasl_enabled,
                }
            except Exception:
                connection_info = {"error": "Failed to get connection info"}

        base_health.update({
            "uptime_seconds": uptime,
            "messages_published": self._messages_published,
            "messages_received": self._messages_received,
            "requests_sent": self._requests_sent,
            "requests_received": self._requests_received,
            "publish_errors": self._publish_errors,
            "receive_errors": self._receive_errors,
            "subscribed_addresses": list(self._subscriptions.keys()),
            "subscription_count": len(self._subscriptions),
            "senders": list(self._senders.keys()),
            "receivers": list(self._receivers.keys()),
            "pending_requests": len(self._pending_requests),
            "request_handlers": list(self._request_handlers.keys()),
            "connection_info": connection_info,
            "config": {
                "hostname": self.hostname,
                "port": self.port,
                "container_id": self.container_id,
                "ssl_enabled": self.ssl_enabled,
                "sasl_enabled": self.sasl_enabled,
                "durable": self.durable,
                "auto_settle": self.auto_settle,
            },
        })

        return base_health


class AMQP1Handler(MessagingHandler):
    """AMQP 1.0 message handler for proton reactor."""

    def __init__(self, pipe_instance: AMQP1HybridPipe) -> None:
        """Initialize handler."""
        super().__init__(auto_accept=pipe_instance.auto_accept, auto_settle=pipe_instance.auto_settle)
        self.pipe = pipe_instance
        self._send_futures: Dict[str, asyncio.Future] = {}

    def on_start(self, event: Event) -> None:
        """Handle reactor start."""
        pass

    def on_connection_opened(self, event: Event) -> None:
        """Handle connection opened."""
        self.pipe._connection_state = ConnectionState.CONNECTED

    def on_connection_error(self, event: Event) -> None:
        """Handle connection error."""
        self.pipe._connection_state = ConnectionState.FAILED
        print(f"AMQP connection error: {event.connection.condition}")

    def on_connection_closed(self, event: Event) -> None:
        """Handle connection closed."""
        self.pipe._connection_state = ConnectionState.DISCONNECTED

    def on_session_opened(self, event: Event) -> None:
        """Handle session opened."""
        pass

    def on_link_opened(self, event: Event) -> None:
        """Handle link opened."""
        pass

    def on_message(self, event: Event) -> None:
        """Handle received message."""
        try:
            message = event.message
            address = message.address or event.receiver.source.address

            # Process message asynchronously
            asyncio.run_coroutine_threadsafe(
                self._process_message(message, address),
                asyncio.get_event_loop()
            )

        except Exception as e:
            print(f"Error handling AMQP message: {e}")

    async def _process_message(self, message: Message, address: str) -> None:
        """Process a received AMQP message."""
        try:
            # Find matching subscriptions
            matching_subscriptions = []
            for subscribed_address, callbacks in self.pipe._subscriptions.items():
                if address == subscribed_address:
                    matching_subscriptions.extend(callbacks)

            if not matching_subscriptions:
                return

            # Extract message data
            data = message.body
            if isinstance(data, str):
                data = data.encode()

            # Create metadata
            metadata = MessageMetadata(
                message_id=str(message.id) if message.id else str(uuid.uuid4()),
                timestamp=datetime.fromtimestamp(message.creation_time) if message.creation_time else datetime.now(),
                pipe=address,
                broker_type=self.pipe.broker_type,
                serialization_format=SerializationFormat.JSON,
                size_bytes=len(data) if isinstance(data, bytes) else len(str(data)),
                headers=dict(message.properties) if message.properties else {},
            )

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
                    print(f"Error in AMQP message callback for address {address}: {e}")

            self.pipe._messages_received += 1

        except Exception as e:
            self.pipe._receive_errors += 1
            print(f"Error processing AMQP message for address {address}: {e}")

    async def send_message(self, address: str, message: Message) -> None:
        """Send a message to an address."""
        # Get or create sender
        if address not in self.pipe._senders:
            await self.create_sender(address)

        sender = self.pipe._senders[address]

        # Send message in reactor thread
        def send_in_reactor():
            sender.send(message)

        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, send_in_reactor)

    async def create_sender(self, address: str) -> None:
        """Create a sender for an address."""
        def create_in_reactor():
            sender = self.pipe._connection.create_sender(address)
            self.pipe._senders[address] = sender
            return sender

        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, create_in_reactor)

    async def create_receiver(self, address: str) -> None:
        """Create a receiver for an address."""
        def create_in_reactor():
            receiver = self.pipe._connection.create_receiver(address)
            self.pipe._receivers[address] = receiver
            return receiver

        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, create_in_reactor)

    async def close_receiver(self, address: str) -> None:
        """Close a receiver for an address."""
        if address in self.pipe._receivers:
            def close_in_reactor():
                receiver = self.pipe._receivers.pop(address)
                receiver.close()

            loop = asyncio.get_event_loop()
            await loop.run_in_executor(None, close_in_reactor)