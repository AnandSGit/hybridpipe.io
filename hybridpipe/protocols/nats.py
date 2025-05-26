"""
NATS protocol implementation for HybridPipe.

This module provides NATS messaging support with JetStream streaming,
request-reply patterns, clustering, and subject-based routing.
"""

import asyncio
import time
import uuid
from typing import Any, Dict, List, Optional, Set, Callable, Union
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor
import threading

try:
    import nats
    from nats.aio.client import Client as NATS
    from nats.aio.subscription import Subscription
    from nats.js import JetStreamContext
    from nats.js.api import StreamConfig, ConsumerConfig, DeliverPolicy
    from nats.errors import TimeoutError as NATSTimeoutError, ConnectionClosedError
    NATS_AVAILABLE = True
except ImportError:
    NATS_AVAILABLE = False

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


# Type aliases for NATS-specific callbacks
RequestCallback = Callable[[bytes, MessageMetadata], Union[bytes, Any]]


@protocol_implementation(
    BrokerType.NATS,
    default_config={
        "servers": ["nats://localhost:4222"],
        "name": "hybridpipe-nats",
        "max_reconnect_attempts": -1,  # Infinite
        "reconnect_time_wait": 2.0,
        "ping_interval": 120,
        "max_outstanding_pings": 2,
        "dont_randomize": False,
        "flusher_queue_size": 1024,
        "no_echo": False,
        "tls": None,
        "user": None,
        "password": None,
        "token": None,
        "drain_timeout": 30.0,
        # JetStream configuration
        "jetstream": True,
        "js_domain": None,
        "stream_config": {
            "retention": "limits",
            "max_age": 86400,  # 24 hours in seconds
            "max_msgs": 1000000,
            "max_bytes": -1,  # Unlimited
            "max_msg_size": -1,  # Unlimited
            "storage": "file",
            "num_replicas": 1,
            "discard": "old",
        },
        "consumer_config": {
            "deliver_policy": "all",
            "ack_policy": "explicit",
            "ack_wait": 30.0,
            "max_deliver": -1,  # Unlimited
            "replay_policy": "instant",
        },
    },
    metadata={
        "description": "NATS messaging with JetStream streaming and request-reply",
        "supports_persistence": True,
        "supports_clustering": True,
        "supports_streaming": True,
        "supports_request_reply": True,
        "supports_federation": True,
        "max_throughput_mps": 100000,
        "min_latency_us": 50,
    },
)
class NATSHybridPipe(HybridPipe):
    """
    NATS implementation of HybridPipe.

    This implementation provides full NATS messaging functionality including:
    - Core NATS pub/sub with subject-based routing
    - JetStream for persistence and streaming
    - Request-reply patterns
    - Clustering and federation support
    - Subject wildcards (* and >)
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize NATS HybridPipe implementation.

        Args:
            config: Configuration dictionary with NATS-specific options
        """
        if not NATS_AVAILABLE:
            raise ImportError(
                "nats-py is required for NATS support. "
                "Install with: pip install nats-py"
            )

        super().__init__(config)

        # NATS configuration with Go compatibility
        self.servers = self._parse_servers_config()
        self.name = self._config.get("name", "hybridpipe-nats")

        # Handle Go-style reconnect configuration
        max_reconnects = self._config.get("max_reconnect_attempts", self._config.get("max_reconnects", -1))
        self.max_reconnect_attempts = max_reconnects if max_reconnects != 0 else -1

        reconnect_wait = self._config.get("reconnect_time_wait", self._config.get("reconnect_wait", 2.0))
        self.reconnect_time_wait = float(reconnect_wait)

        self.ping_interval = self._config.get("ping_interval", 120)
        self.drain_timeout = self._config.get("drain_timeout", 30.0)

        # Authentication with Go compatibility
        self.user = self._config.get("user") or self._config.get("username")
        self.password = self._config.get("password")
        self.token = self._config.get("token")
        self.tls = self._parse_tls_config()

        # JetStream configuration
        self.jetstream_enabled = self._config.get("jetstream", True)
        self.js_domain = self._config.get("js_domain")
        self.stream_config = self._config.get("stream_config", {})
        self.consumer_config = self._config.get("consumer_config", {})

        # Connection state
        self._connection_state = ConnectionState.DISCONNECTED
        self._nc: Optional[NATS] = None
        self._js: Optional[JetStreamContext] = None

        # Subscription management
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._nats_subscriptions: Dict[str, Subscription] = {}
        self._request_handlers: Dict[str, RequestCallback] = {}
        self._streams: Set[str] = set()

        # Metrics
        self._messages_published = 0
        self._messages_received = 0
        self._requests_sent = 0
        self._requests_received = 0
        self._publish_errors = 0
        self._receive_errors = 0
        self._start_time = time.time()

    def _parse_servers_config(self) -> List[str]:
        """Parse servers configuration with Go compatibility."""
        # Python-style: servers array
        if "servers" in self._config:
            return self._config["servers"]

        # Go-style: separate server and port
        if "server" in self._config or "nserver" in self._config:
            server = self._config.get("server", self._config.get("nserver", "localhost"))
            port = self._config.get("port", self._config.get("nlport", 4222))

            # Build URL if not already a URL
            if "://" not in server:
                server = f"nats://{server}:{port}"

            return [server]

        # Default
        return ["nats://localhost:4222"]

    def _parse_tls_config(self) -> Optional[Dict[str, Any]]:
        """Parse TLS configuration with Go compatibility."""
        tls_config = self._config.get("tls")
        if tls_config:
            return tls_config

        # Go-style TLS configuration
        cert_file = self._config.get("cert_file", self._config.get("natscertfile"))
        key_file = self._config.get("key_file", self._config.get("natskeyfile"))
        ca_file = self._config.get("ca_file", self._config.get("natscafile"))

        if cert_file or key_file or ca_file:
            tls_config = {}
            if cert_file:
                tls_config["certfile"] = cert_file
            if key_file:
                tls_config["keyfile"] = key_file
            if ca_file:
                tls_config["ca_certs"] = ca_file
            return tls_config

        return None

    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.NATS

    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=self.jetstream_enabled,
            supports_transactions=False,  # NATS doesn't have traditional transactions
            supports_clustering=True,
            supports_compression=False,  # NATS doesn't compress at protocol level
            supports_encryption=bool(self.tls),
            supports_authentication=bool(self.user or self.token),
            supports_authorization=True,
            supports_dead_letter=False,  # Not directly supported
            supports_message_ordering=self.jetstream_enabled,
            supports_exactly_once=False,  # NATS provides at-most-once or at-least-once
            max_message_size=1024 * 1024,  # 1MB default NATS limit
            max_pipe_length=None,  # No specific limit on subject length
        )

    async def connect(self) -> None:
        """Establish connection to NATS server(s)."""
        if self._connection_state == ConnectionState.CONNECTED:
            return

        self._connection_state = ConnectionState.CONNECTING

        try:
            # Build connection options
            options = {
                "servers": self.servers,
                "name": self.name,
                "max_reconnect_attempts": self.max_reconnect_attempts,
                "reconnect_time_wait": self.reconnect_time_wait,
                "ping_interval": self.ping_interval,
                "max_outstanding_pings": self._config.get("max_outstanding_pings", 2),
                "dont_randomize": self._config.get("dont_randomize", False),
                "flusher_queue_size": self._config.get("flusher_queue_size", 1024),
                "no_echo": self._config.get("no_echo", False),
                "drain_timeout": self.drain_timeout,
            }

            # Add authentication if configured
            if self.user and self.password:
                options["user"] = self.user
                options["password"] = self.password
            elif self.token:
                options["token"] = self.token

            # Add TLS if configured
            if self.tls:
                options["tls"] = self.tls

            # Add connection callbacks
            options["disconnected_cb"] = self._on_disconnected
            options["reconnected_cb"] = self._on_reconnected
            options["error_cb"] = self._on_error
            options["closed_cb"] = self._on_closed

            # Connect to NATS
            self._nc = NATS()
            await self._nc.connect(**options)

            # Initialize JetStream if enabled
            if self.jetstream_enabled:
                self._js = self._nc.jetstream(domain=self.js_domain)

            self._connection_state = ConnectionState.CONNECTED

        except Exception as e:
            self._connection_state = ConnectionState.FAILED
            raise ConnectionError(
                f"Failed to connect to NATS: {e}",
                broker_type=self.broker_type,
            ) from e

    async def disconnect(self) -> None:
        """Close connection to NATS server(s)."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return

        self._connection_state = ConnectionState.DISCONNECTED

        try:
            if self._nc and not self._nc.is_closed:
                # Drain and close connection
                await self._nc.drain()

            # Clear state
            self._subscriptions.clear()
            self._nats_subscriptions.clear()
            self._request_handlers.clear()
            self._streams.clear()
            self._js = None
            self._nc = None

        except Exception as e:
            print(f"Error during NATS disconnect: {e}")

    # Connection event handlers
    async def _on_disconnected(self):
        """Handle disconnection from NATS."""
        print("NATS connection lost")

    async def _on_reconnected(self):
        """Handle reconnection to NATS."""
        print("NATS connection restored")

    async def _on_error(self, error):
        """Handle NATS connection errors."""
        print(f"NATS connection error: {error}")

    async def _on_closed(self):
        """Handle NATS connection closure."""
        print("NATS connection closed")

    async def dispatch(
        self,
        pipe: str,
        data: Any,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """Send a message to the specified NATS subject."""
        if not self.is_connected or not self._nc:
            raise ConnectionError(
                "Not connected to NATS server",
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

            # Prepare NATS headers
            nats_headers = {}
            if headers:
                nats_headers.update(headers)

            # Add metadata headers
            nats_headers.update({
                "hybridpipe_message_id": metadata.message_id,
                "hybridpipe_timestamp": metadata.timestamp.isoformat(),
                "hybridpipe_broker_type": metadata.broker_type.value,
            })

            # Publish message
            if self.jetstream_enabled and self._js:
                # Use JetStream for persistence
                await self._js.publish(
                    subject=pipe,
                    payload=serialized_data,
                    headers=nats_headers,
                )
            else:
                # Use core NATS
                await self._nc.publish(
                    subject=pipe,
                    payload=serialized_data,
                    headers=nats_headers,
                )

            self._messages_published += 1

        except Exception as e:
            self._publish_errors += 1
            if isinstance(e, (ConnectionError, ProtocolError, TimeoutError)):
                raise
            raise ProtocolError(
                f"Failed to publish message to NATS subject '{pipe}': {e}",
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
        """Subscribe to messages from the specified NATS subject pattern."""
        if not self.is_connected or not self._nc:
            raise ConnectionError(
                "Not connected to NATS server",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        # Add callback to subscriptions
        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
        self._subscriptions[pipe].append(callback)

        # Create NATS subscription if not already exists
        if pipe not in self._nats_subscriptions:
            try:
                # Create message handler
                async def nats_handler(msg):
                    await self._process_message(msg, pipe)

                # Subscribe to subject
                if self.jetstream_enabled and self._js and headers and headers.get("durable"):
                    # Create durable consumer for JetStream
                    consumer_config = ConsumerConfig(
                        durable_name=headers["durable"],
                        deliver_policy=DeliverPolicy.ALL,
                        ack_policy=headers.get("ack_policy", "explicit"),
                        ack_wait=headers.get("ack_wait", 30.0),
                        max_deliver=headers.get("max_deliver", -1),
                    )

                    subscription = await self._js.subscribe(
                        subject=pipe,
                        cb=nats_handler,
                        config=consumer_config,
                    )
                else:
                    # Use core NATS subscription
                    subscription = await self._nc.subscribe(
                        subject=pipe,
                        cb=nats_handler,
                        queue=headers.get("queue") if headers else None,
                    )

                self._nats_subscriptions[pipe] = subscription

            except Exception as e:
                raise ProtocolError(f"NATS subscribe failed: {e}") from e

    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified NATS subject pattern."""
        if pipe in self._subscriptions:
            del self._subscriptions[pipe]

        if pipe in self._nats_subscriptions:
            try:
                subscription = self._nats_subscriptions[pipe]
                await subscription.unsubscribe()
                del self._nats_subscriptions[pipe]
            except Exception as e:
                print(f"NATS unsubscribe failed for subject {pipe}: {e}")

    async def _process_message(self, msg, subject: str) -> None:
        """Process a received NATS message."""
        # Find matching subscriptions
        matching_subscriptions = []
        for subscribed_subject, callbacks in self._subscriptions.items():
            if self._subject_matches(subject, subscribed_subject):
                matching_subscriptions.extend(callbacks)

        if not matching_subscriptions:
            return

        try:
            # Extract message data
            data = msg.data
            headers = msg.headers or {}

            # Create metadata
            metadata = MessageMetadata(
                message_id=headers.get("hybridpipe_message_id", str(uuid.uuid4())),
                timestamp=datetime.fromisoformat(
                    headers.get("hybridpipe_timestamp", datetime.now().isoformat())
                ),
                pipe=subject,
                broker_type=self.broker_type,
                serialization_format=SerializationFormat.JSON,
                size_bytes=len(data),
                headers=dict(headers),
            )

            # Call all matching subscribers
            for callback in matching_subscriptions:
                try:
                    result = callback(data, metadata)
                    if asyncio.iscoroutine(result):
                        await result
                except Exception as e:
                    print(f"Error in NATS message callback for subject {subject}: {e}")

            # Acknowledge message if using JetStream
            if hasattr(msg, 'ack'):
                await msg.ack()

            self._messages_received += 1

        except Exception as e:
            self._receive_errors += 1
            print(f"Error processing NATS message for subject {subject}: {e}")

    def _subject_matches(self, subject: str, pattern: str) -> bool:
        """Check if a subject matches a subscription pattern with NATS wildcards."""
        # NATS wildcards: * matches single token, > matches multiple tokens
        import re

        # Convert NATS wildcards to regex
        escaped_pattern = re.escape(pattern)
        escaped_pattern = escaped_pattern.replace(r"\*", "[^.]+")  # * matches single token
        escaped_pattern = escaped_pattern.replace(r"\>", ".*")     # > matches multiple tokens

        # Ensure full match
        regex_pattern = f"^{escaped_pattern}$"

        return bool(re.match(regex_pattern, subject))

    # NATS-specific methods

    async def request(
        self,
        subject: str,
        data: Any,
        timeout: float = 5.0,
        headers: Optional[Dict[str, Any]] = None,
    ) -> Any:
        """Send a request and wait for a reply."""
        if not self.is_connected or not self._nc:
            raise ConnectionError(
                "Not connected to NATS server",
                broker_type=self.broker_type,
                pipe=subject,
            )

        try:
            # Serialize request data
            serialized_data = encode(data, SerializationFormat.JSON)

            # Prepare headers
            nats_headers = {}
            if headers:
                nats_headers.update(headers)

            # Send request and wait for reply
            response = await self._nc.request(
                subject=subject,
                payload=serialized_data,
                timeout=timeout,
                headers=nats_headers,
            )

            self._requests_sent += 1

            # Decode response
            return decode(response.data, SerializationFormat.JSON)

        except NATSTimeoutError:
            raise TimeoutError(
                f"NATS request timed out after {timeout} seconds",
                timeout_seconds=timeout,
                operation="request",
                broker_type=self.broker_type,
                pipe=subject,
            )
        except Exception as e:
            raise ProtocolError(
                f"NATS request failed for subject '{subject}': {e}",
                broker_type=self.broker_type,
                pipe=subject,
            ) from e

    async def reply(
        self,
        subject: str,
        callback: RequestCallback,
        queue: Optional[str] = None,
    ) -> None:
        """Handle requests on a subject."""
        if not self.is_connected or not self._nc:
            raise ConnectionError(
                "Not connected to NATS server",
                broker_type=self.broker_type,
                pipe=subject,
            )

        # Store request handler
        self._request_handlers[subject] = callback

        try:
            async def request_handler(msg):
                try:
                    # Create metadata
                    metadata = MessageMetadata(
                        message_id=str(uuid.uuid4()),
                        timestamp=datetime.now(),
                        pipe=subject,
                        broker_type=self.broker_type,
                        serialization_format=SerializationFormat.JSON,
                        size_bytes=len(msg.data),
                        headers=dict(msg.headers or {}),
                    )

                    # Call the callback
                    result = callback(msg.data, metadata)
                    if asyncio.iscoroutine(result):
                        result = await result

                    # Send reply
                    if result is not None:
                        if isinstance(result, bytes):
                            reply_data = result
                        else:
                            reply_data = encode(result, SerializationFormat.JSON)

                        await msg.respond(reply_data)

                    self._requests_received += 1

                except Exception as e:
                    print(f"Error in NATS request handler for subject {subject}: {e}")
                    # Send error response
                    error_response = encode({"error": str(e)}, SerializationFormat.JSON)
                    await msg.respond(error_response)

            # Subscribe to requests
            await self._nc.subscribe(subject, cb=request_handler, queue=queue)

        except Exception as e:
            raise ProtocolError(f"NATS reply setup failed: {e}") from e

    async def create_stream(
        self,
        name: str,
        subjects: List[str],
        config: Optional[Dict[str, Any]] = None,
    ) -> None:
        """Create a JetStream stream."""
        if not self.jetstream_enabled or not self._js:
            raise ProtocolError("JetStream is not enabled")

        try:
            # Build stream configuration
            stream_config = StreamConfig(
                name=name,
                subjects=subjects,
                retention=config.get("retention", "limits") if config else "limits",
                max_age=config.get("max_age", 86400) if config else 86400,
                max_msgs=config.get("max_msgs", 1000000) if config else 1000000,
                max_bytes=config.get("max_bytes", -1) if config else -1,
                max_msg_size=config.get("max_msg_size", -1) if config else -1,
                storage=config.get("storage", "file") if config else "file",
                num_replicas=config.get("num_replicas", 1) if config else 1,
                discard=config.get("discard", "old") if config else "old",
            )

            # Create stream
            await self._js.add_stream(config=stream_config)
            self._streams.add(name)

        except Exception as e:
            raise ProtocolError(f"Failed to create NATS stream '{name}': {e}") from e

    async def delete_stream(self, name: str) -> None:
        """Delete a JetStream stream."""
        if not self.jetstream_enabled or not self._js:
            raise ProtocolError("JetStream is not enabled")

        try:
            await self._js.delete_stream(name)
            self._streams.discard(name)
        except Exception as e:
            raise ProtocolError(f"Failed to delete NATS stream '{name}': {e}") from e

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()

        # Add NATS-specific health information
        uptime = time.time() - self._start_time

        # Get connection info
        connection_info = {}
        if self.is_connected and self._nc:
            try:
                connection_info = {
                    "connected_url": str(self._nc.connected_url) if self._nc.connected_url else None,
                    "server_info": dict(self._nc.server_info) if self._nc.server_info else {},
                    "is_draining": self._nc.is_draining,
                    "is_reconnecting": self._nc.is_reconnecting,
                    "stats": {
                        "in_msgs": self._nc.stats["in_msgs"],
                        "out_msgs": self._nc.stats["out_msgs"],
                        "in_bytes": self._nc.stats["in_bytes"],
                        "out_bytes": self._nc.stats["out_bytes"],
                        "reconnects": self._nc.stats["reconnects"],
                    },
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
            "subscribed_subjects": list(self._subscriptions.keys()),
            "subscription_count": len(self._subscriptions),
            "request_handlers": list(self._request_handlers.keys()),
            "jetstream_enabled": self.jetstream_enabled,
            "streams": list(self._streams),
            "connection_info": connection_info,
            "config": {
                "servers": self.servers,
                "name": self.name,
                "jetstream": self.jetstream_enabled,
                "max_reconnect_attempts": self.max_reconnect_attempts,
                "reconnect_time_wait": self.reconnect_time_wait,
            },
        })

        return base_health