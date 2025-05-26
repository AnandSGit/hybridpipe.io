"""
MQTT protocol implementation for HybridPipe.

This module provides MQTT messaging support using paho-mqtt
with QoS levels, retained messages, and TLS encryption.
"""

import asyncio
import json
import time
import uuid
import ssl
from typing import Any, Dict, List, Optional, Set, Callable
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor
import threading

try:
    import paho.mqtt.client as mqtt
    MQTT_AVAILABLE = True
except ImportError:
    MQTT_AVAILABLE = False

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
    BrokerType.MQTT,
    default_config={
        "host": "localhost",
        "port": 1883,
        "keepalive": 60,
        "client_id": None,  # Auto-generated if None
        "clean_session": True,
        "username": None,
        "password": None,
        "protocol": "MQTTv311",  # MQTTv31, MQTTv311, MQTTv5
        "transport": "tcp",  # tcp, websockets
        "tls": False,
        "tls_ca_certs": None,
        "tls_certfile": None,
        "tls_keyfile": None,
        "tls_cert_reqs": "required",  # none, optional, required
        "tls_version": None,  # Auto-select
        "tls_ciphers": None,
        "will_topic": None,
        "will_payload": None,
        "will_qos": 0,
        "will_retain": False,
        "default_qos": 1,
        "default_retain": False,
        "max_inflight_messages": 20,
        "max_queued_messages": 0,  # 0 = unlimited
        "message_retry": 5,
        "reconnect_delay": 1,
        "max_reconnect_delay": 120,
    },
    metadata={
        "description": "MQTT messaging with QoS levels and retained messages",
        "supports_persistence": True,
        "supports_clustering": False,
        "supports_transactions": False,
        "supports_qos": True,
    },
)
class MQTTHybridPipe(HybridPipe):
    """
    MQTT implementation of HybridPipe.

    This implementation provides full MQTT functionality including
    QoS levels, retained messages, will messages, and TLS encryption.

    Features:
    - QoS levels 0, 1, and 2
    - Retained messages
    - Will messages (last will and testament)
    - TLS/SSL encryption
    - Topic pattern subscriptions
    - Connection keep-alive and reconnection
    - Clean session handling
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize MQTT HybridPipe implementation.

        Args:
            config: Configuration dictionary with MQTT-specific options
        """
        if not MQTT_AVAILABLE:
            raise ImportError(
                "paho-mqtt is required for MQTT support. "
                "Install with: pip install paho-mqtt"
            )

        super().__init__(config)

        # MQTT configuration
        self.host = self._config.get("host", "localhost")
        self.port = self._config.get("port", 1883)
        self.keepalive = self._config.get("keepalive", 60)
        self.client_id = self._config.get("client_id") or f"hybridpipe-{uuid.uuid4().hex[:8]}"
        self.clean_session = self._config.get("clean_session", True)
        self.username = self._config.get("username")
        self.password = self._config.get("password")

        # Protocol configuration
        protocol_map = {
            "MQTTv31": mqtt.MQTTv31,
            "MQTTv311": mqtt.MQTTv311,
        }
        if hasattr(mqtt, "MQTTv5"):
            protocol_map["MQTTv5"] = mqtt.MQTTv5

        protocol_name = self._config.get("protocol", "MQTTv311")
        self.protocol = protocol_map.get(protocol_name, mqtt.MQTTv311)

        self.transport = self._config.get("transport", "tcp")

        # TLS configuration
        self.tls = self._config.get("tls", False)
        self.tls_ca_certs = self._config.get("tls_ca_certs")
        self.tls_certfile = self._config.get("tls_certfile")
        self.tls_keyfile = self._config.get("tls_keyfile")
        self.tls_cert_reqs = self._config.get("tls_cert_reqs", "required")
        self.tls_version = self._config.get("tls_version")
        self.tls_ciphers = self._config.get("tls_ciphers")

        # Will message configuration
        self.will_topic = self._config.get("will_topic")
        self.will_payload = self._config.get("will_payload")
        self.will_qos = self._config.get("will_qos", 0)
        self.will_retain = self._config.get("will_retain", False)

        # Message configuration
        self.default_qos = self._config.get("default_qos", 1)
        self.default_retain = self._config.get("default_retain", False)

        # Connection state
        self._connection_state = ConnectionState.DISCONNECTED
        self._client: Optional[mqtt.Client] = None
        self._connect_event = asyncio.Event()
        self._disconnect_event = asyncio.Event()

        # Subscription management
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._subscribed_topics: Set[str] = set()

        # Threading for MQTT operations
        self._executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="mqtt-")
        self._client_lock = threading.Lock()

        # Metrics
        self._messages_published = 0
        self._messages_received = 0
        self._publish_errors = 0
        self._receive_errors = 0
        self._start_time = time.time()

        # Message tracking for QoS > 0
        self._pending_publishes: Dict[int, asyncio.Future] = {}
        self._message_id_counter = 1

    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.MQTT

    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=True,
            supports_transactions=False,
            supports_clustering=False,
            supports_compression=False,
            supports_encryption=self.tls,
            supports_authentication=bool(self.username),
            supports_authorization=False,  # Depends on broker
            supports_dead_letter=False,
            supports_message_ordering=True,  # QoS 1 and 2 provide ordering
            supports_exactly_once=True,  # QoS 2
            max_message_size=256 * 1024 * 1024,  # 256MB theoretical limit
            max_pipe_length=65535,  # Topic length limit
        )

    async def connect(self) -> None:
        """Establish connection to MQTT broker."""
        if self._connection_state == ConnectionState.CONNECTED:
            return

        self._connection_state = ConnectionState.CONNECTING

        try:
            # Create MQTT client
            self._client = mqtt.Client(
                client_id=self.client_id,
                clean_session=self.clean_session,
                protocol=self.protocol,
                transport=self.transport,
            )

            # Set callbacks
            self._client.on_connect = self._on_connect
            self._client.on_disconnect = self._on_disconnect
            self._client.on_message = self._on_message
            self._client.on_publish = self._on_publish
            self._client.on_subscribe = self._on_subscribe
            self._client.on_unsubscribe = self._on_unsubscribe

            # Set authentication
            if self.username:
                self._client.username_pw_set(self.username, self.password)

            # Configure TLS
            if self.tls:
                self._configure_tls()

            # Configure will message
            if self.will_topic:
                self._client.will_set(
                    topic=self.will_topic,
                    payload=self.will_payload,
                    qos=self.will_qos,
                    retain=self.will_retain,
                )

            # Configure client options
            self._client.max_inflight_messages_set(
                self._config.get("max_inflight_messages", 20)
            )
            self._client.max_queued_messages_set(
                self._config.get("max_queued_messages", 0)
            )
            self._client.message_retry_set(
                self._config.get("message_retry", 5)
            )
            self._client.reconnect_delay_set(
                min_delay=self._config.get("reconnect_delay", 1),
                max_delay=self._config.get("max_reconnect_delay", 120),
            )

            # Start client loop in thread
            self._client.loop_start()

            # Connect to broker
            result = self._client.connect(
                host=self.host,
                port=self.port,
                keepalive=self.keepalive,
            )

            if result != mqtt.MQTT_ERR_SUCCESS:
                raise ConnectionError(f"MQTT connect failed with code {result}")

            # Wait for connection
            await asyncio.wait_for(self._connect_event.wait(), timeout=30.0)

            self._connection_state = ConnectionState.CONNECTED

        except Exception as e:
            self._connection_state = ConnectionState.FAILED
            if self._client:
                self._client.loop_stop()
                self._client = None
            raise ConnectionError(
                f"Failed to connect to MQTT broker: {e}",
                broker_type=self.broker_type,
            ) from e

    async def disconnect(self) -> None:
        """Close connection to MQTT broker."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return

        self._connection_state = ConnectionState.DISCONNECTED

        if self._client:
            # Disconnect from broker
            self._client.disconnect()

            # Wait for disconnection
            try:
                await asyncio.wait_for(self._disconnect_event.wait(), timeout=10.0)
            except asyncio.TimeoutError:
                pass

            # Stop client loop
            self._client.loop_stop()
            self._client = None

        # Clear state
        self._subscriptions.clear()
        self._subscribed_topics.clear()
        self._pending_publishes.clear()
        self._connect_event.clear()
        self._disconnect_event.clear()

    def _configure_tls(self) -> None:
        """Configure TLS/SSL for MQTT client."""
        # Map cert requirements
        cert_reqs_map = {
            "none": ssl.CERT_NONE,
            "optional": ssl.CERT_OPTIONAL,
            "required": ssl.CERT_REQUIRED,
        }
        cert_reqs = cert_reqs_map.get(self.tls_cert_reqs, ssl.CERT_REQUIRED)

        # Configure TLS
        self._client.tls_set(
            ca_certs=self.tls_ca_certs,
            certfile=self.tls_certfile,
            keyfile=self.tls_keyfile,
            cert_reqs=cert_reqs,
            tls_version=self.tls_version,
            ciphers=self.tls_ciphers,
        )

    # MQTT callback methods
    def _on_connect(self, client, userdata, flags, rc) -> None:
        """Callback for MQTT connection."""
        if rc == 0:
            self._connect_event.set()
        else:
            print(f"MQTT connection failed with return code: {rc}")

    def _on_disconnect(self, client, userdata, rc) -> None:
        """Callback for MQTT disconnection."""
        self._disconnect_event.set()
        if rc != 0:
            print(f"MQTT unexpected disconnection with return code: {rc}")

    def _on_message(self, client, userdata, message) -> None:
        """Callback for received MQTT message."""
        try:
            # Schedule message processing in async context
            loop = asyncio.get_event_loop()
            asyncio.run_coroutine_threadsafe(
                self._process_message(message),
                loop
            )
        except Exception as e:
            print(f"Error scheduling MQTT message processing for topic {message.topic}: {e}")

    def _on_publish(self, client, userdata, mid) -> None:
        """Callback for message publish confirmation."""
        if mid in self._pending_publishes:
            future = self._pending_publishes.pop(mid)
            if not future.done():
                future.set_result(None)

    def _on_subscribe(self, client, userdata, mid, granted_qos) -> None:
        """Callback for subscription confirmation."""
        print(f"MQTT subscription confirmed: mid={mid}, granted_qos={granted_qos}")

    def _on_unsubscribe(self, client, userdata, mid) -> None:
        """Callback for unsubscription confirmation."""
        print(f"MQTT unsubscription confirmed: mid={mid}")

    async def dispatch(
        self,
        pipe: str,
        data: Any,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """Send a message to the specified MQTT topic."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to MQTT broker",
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

            # Prepare message payload
            message_data = {
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

            # Get QoS and retain from headers or use defaults
            qos = headers.get("qos", self.default_qos) if headers else self.default_qos
            retain = headers.get("retain", self.default_retain) if headers else self.default_retain

            # Publish message
            loop = asyncio.get_event_loop()

            def publish_message():
                with self._client_lock:
                    result = self._client.publish(
                        topic=pipe,
                        payload=serialized_data,
                        qos=qos,
                        retain=retain,
                    )
                    return result

            result = await loop.run_in_executor(self._executor, publish_message)

            if result.rc != mqtt.MQTT_ERR_SUCCESS:
                raise ProtocolError(f"MQTT publish failed with code {result.rc}")

            # Wait for publish confirmation if QoS > 0
            if qos > 0:
                future = asyncio.Future()
                self._pending_publishes[result.mid] = future
                try:
                    await asyncio.wait_for(future, timeout=30.0)
                except asyncio.TimeoutError:
                    self._pending_publishes.pop(result.mid, None)
                    raise TimeoutError("MQTT publish confirmation timeout")

            self._messages_published += 1

        except Exception as e:
            self._publish_errors += 1
            if isinstance(e, (ConnectionError, ProtocolError, TimeoutError)):
                raise
            raise ProtocolError(
                f"Failed to publish message to MQTT topic '{pipe}': {e}",
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
        """Subscribe to messages from the specified MQTT topic pattern."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to MQTT broker",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        # Add callback to subscriptions
        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
        self._subscriptions[pipe].append(callback)

        # Subscribe to topic if not already subscribed
        if pipe not in self._subscribed_topics:
            qos = headers.get("qos", self.default_qos) if headers else self.default_qos

            loop = asyncio.get_event_loop()

            def subscribe_topic():
                with self._client_lock:
                    result = self._client.subscribe(pipe, qos=qos)
                    return result

            result = await loop.run_in_executor(self._executor, subscribe_topic)

            if result[0] != mqtt.MQTT_ERR_SUCCESS:
                raise ProtocolError(f"MQTT subscribe failed with code {result[0]}")

            self._subscribed_topics.add(pipe)

    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified MQTT topic pattern."""
        if pipe in self._subscriptions:
            del self._subscriptions[pipe]

        if pipe in self._subscribed_topics:
            loop = asyncio.get_event_loop()

            def unsubscribe_topic():
                with self._client_lock:
                    result = self._client.unsubscribe(pipe)
                    return result

            result = await loop.run_in_executor(self._executor, unsubscribe_topic)

            if result[0] != mqtt.MQTT_ERR_SUCCESS:
                print(f"MQTT unsubscribe failed for topic {pipe}: return_code={result[0]}")

            self._subscribed_topics.remove(pipe)

    async def _process_message(self, message) -> None:
        """Process a received MQTT message."""
        topic = message.topic

        # Find matching subscriptions (support wildcards)
        matching_subscriptions = []
        for subscribed_topic, callbacks in self._subscriptions.items():
            if self._topic_matches(topic, subscribed_topic):
                matching_subscriptions.extend(callbacks)

        if not matching_subscriptions:
            return

        try:
            # Decode message payload
            try:
                message_data = decode(message.payload, SerializationFormat.JSON)
                data = message_data.get("data", message.payload)
                metadata_dict = message_data.get("metadata", {})
            except Exception:
                # Fallback to raw payload
                data = message.payload
                metadata_dict = {}

            # Create metadata
            metadata = MessageMetadata(
                message_id=metadata_dict.get("message_id", str(uuid.uuid4())),
                timestamp=datetime.fromisoformat(
                    metadata_dict.get("timestamp", datetime.now().isoformat())
                ),
                pipe=topic,
                broker_type=self.broker_type,
                serialization_format=SerializationFormat.JSON,
                size_bytes=len(message.payload),
                headers=metadata_dict.get("headers", {}),
            )

            # Call all matching subscribers
            for callback in matching_subscriptions:
                try:
                    result = callback(data if isinstance(data, bytes) else encode(data), metadata)
                    if asyncio.iscoroutine(result):
                        await result
                except Exception as e:
                    print(f"Error in MQTT message callback for topic {topic}: {e}")

            self._messages_received += 1

        except Exception as e:
            self._receive_errors += 1
            print(f"Error processing MQTT message for topic {topic}: {e}")

    def _topic_matches(self, topic: str, pattern: str) -> bool:
        """Check if a topic matches a subscription pattern with MQTT wildcards."""
        # Convert MQTT wildcards to regex
        # + matches single level, # matches multiple levels
        import re

        # Escape special regex characters except + and #
        escaped_pattern = re.escape(pattern)

        # Replace escaped wildcards with regex equivalents
        escaped_pattern = escaped_pattern.replace(r"\+", "[^/]+")
        escaped_pattern = escaped_pattern.replace(r"\#", ".*")

        # Ensure full match
        regex_pattern = f"^{escaped_pattern}$"

        return bool(re.match(regex_pattern, topic))

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()

        # Add MQTT-specific health information
        uptime = time.time() - self._start_time

        # Get client info
        client_info = {}
        if self.is_connected and self._client:
            try:
                client_info = {
                    "client_id": self.client_id,
                    "clean_session": self.clean_session,
                    "protocol": str(self.protocol),
                    "transport": self.transport,
                    "keepalive": self.keepalive,
                    "is_connected": self._client.is_connected(),
                }
            except Exception:
                client_info = {"error": "Failed to get client info"}

        base_health.update({
            "uptime_seconds": uptime,
            "messages_published": self._messages_published,
            "messages_received": self._messages_received,
            "publish_errors": self._publish_errors,
            "receive_errors": self._receive_errors,
            "subscribed_topics": list(self._subscribed_topics),
            "subscription_count": len(self._subscriptions),
            "pending_publishes": len(self._pending_publishes),
            "client_info": client_info,
            "config": {
                "host": self.host,
                "port": self.port,
                "client_id": self.client_id,
                "protocol": str(self.protocol),
                "transport": self.transport,
                "tls": self.tls,
                "default_qos": self.default_qos,
                "default_retain": self.default_retain,
                "will_topic": self.will_topic,
            },
        })

        return base_health