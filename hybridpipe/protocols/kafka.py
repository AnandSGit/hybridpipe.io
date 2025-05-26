"""
Kafka protocol implementation for HybridPipe.

This module provides Apache Kafka messaging support using confluent-kafka-python
with full producer/consumer functionality, consumer groups, and security features.
"""

import asyncio
import json
import time
import uuid
from typing import Any, Dict, List, Optional, Set
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor
import threading

try:
    from confluent_kafka import Producer, Consumer, KafkaError, KafkaException
    from confluent_kafka.admin import AdminClient, NewTopic
    from confluent_kafka.schema_registry import SchemaRegistryClient
    from confluent_kafka.schema_registry.avro import AvroSerializer, AvroDeserializer
    KAFKA_AVAILABLE = True
except ImportError:
    KAFKA_AVAILABLE = False

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
    BrokerType.KAFKA,
    default_config={
        "bootstrap_servers": ["localhost:9092"],
        "client_id": "hybridpipe-python",
        "group_id": "hybridpipe-group",
        "auto_offset_reset": "latest",
        "enable_auto_commit": True,
        "auto_commit_interval_ms": 5000,
        "session_timeout_ms": 30000,
        "heartbeat_interval_ms": 3000,
        "max_poll_records": 500,
        "security_protocol": "PLAINTEXT",
        "compression_type": "none",
        "acks": "1",
        "retries": 3,
        "batch_size": 16384,
        "linger_ms": 0,
        "buffer_memory": 33554432,
        "max_request_size": 1048576,
        "request_timeout_ms": 30000,
        "delivery_timeout_ms": 120000,
    },
    metadata={
        "description": "Apache Kafka messaging with producer/consumer support",
        "supports_persistence": True,
        "supports_clustering": True,
        "supports_transactions": True,
        "supports_exactly_once": True,
    },
)
class KafkaHybridPipe(HybridPipe):
    """
    Apache Kafka implementation of HybridPipe.

    This implementation provides full Kafka producer/consumer functionality
    with support for consumer groups, transactions, and security protocols.

    Features:
    - Producer/consumer with error handling
    - Consumer groups and partition management
    - SASL/SSL authentication
    - Schema registry integration
    - Transactional messaging
    - Connection recovery and monitoring
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize Kafka HybridPipe implementation.

        Args:
            config: Configuration dictionary with Kafka-specific options
        """
        if not KAFKA_AVAILABLE:
            raise ImportError(
                "confluent-kafka-python is required for Kafka support. "
                "Install with: pip install confluent-kafka"
            )

        super().__init__(config)

        # Kafka configuration
        self.bootstrap_servers = self._config.get("bootstrap_servers", ["localhost:9092"])
        self.client_id = self._config.get("client_id", "hybridpipe-python")
        self.group_id = self._config.get("group_id", "hybridpipe-group")
        self.auto_offset_reset = self._config.get("auto_offset_reset", "latest")
        self.enable_auto_commit = self._config.get("enable_auto_commit", True)
        self.security_protocol = self._config.get("security_protocol", "PLAINTEXT")
        self.compression_type = self._config.get("compression_type", "none")
        self.acks = self._config.get("acks", "1")
        self.retries = self._config.get("retries", 3)

        # Connection state
        self._connection_state = ConnectionState.DISCONNECTED
        self._producer: Optional[Producer] = None
        self._consumer: Optional[Consumer] = None
        self._admin_client: Optional[AdminClient] = None
        self._schema_registry_client: Optional[SchemaRegistryClient] = None

        # Subscription management
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._subscribed_topics: Set[str] = set()
        self._consumer_task: Optional[asyncio.Task] = None
        self._stop_consuming = False

        # Threading for Kafka operations
        self._executor = ThreadPoolExecutor(max_workers=4, thread_name_prefix="kafka-")
        self._consumer_lock = threading.Lock()

        # Metrics
        self._messages_produced = 0
        self._messages_consumed = 0
        self._produce_errors = 0
        self._consume_errors = 0
        self._start_time = time.time()

        # Transaction support
        self._transactional_id = self._config.get("transactional_id")
        self._enable_transactions = bool(self._transactional_id)

    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.KAFKA

    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=True,
            supports_transactions=self._enable_transactions,
            supports_clustering=True,
            supports_compression=True,
            supports_encryption=self.security_protocol in ["SSL", "SASL_SSL"],
            supports_authentication=self.security_protocol in ["SASL_PLAINTEXT", "SASL_SSL"],
            supports_authorization=True,
            supports_dead_letter=True,
            supports_message_ordering=True,
            supports_exactly_once=self._enable_transactions,
            max_message_size=self._config.get("max_request_size", 1048576),
            max_pipe_length=255,  # Kafka topic name limit
        )

    async def connect(self) -> None:
        """Establish connection to Kafka cluster."""
        if self._connection_state == ConnectionState.CONNECTED:
            return

        self._connection_state = ConnectionState.CONNECTING

        try:
            # Create producer configuration
            producer_config = self._build_producer_config()

            # Create consumer configuration
            consumer_config = self._build_consumer_config()

            # Create admin configuration
            admin_config = self._build_admin_config()

            # Initialize clients in thread pool
            loop = asyncio.get_event_loop()

            self._producer = await loop.run_in_executor(
                self._executor, lambda: Producer(producer_config)
            )

            self._consumer = await loop.run_in_executor(
                self._executor, lambda: Consumer(consumer_config)
            )

            self._admin_client = await loop.run_in_executor(
                self._executor, lambda: AdminClient(admin_config)
            )

            # Initialize schema registry if configured
            schema_registry_url = self._config.get("schema_registry_url")
            if schema_registry_url:
                self._schema_registry_client = SchemaRegistryClient({
                    "url": schema_registry_url
                })

            # Initialize transactions if enabled
            if self._enable_transactions:
                await loop.run_in_executor(
                    self._executor, self._producer.init_transactions
                )

            # Test connection by getting metadata
            await self._test_connection()

            self._connection_state = ConnectionState.CONNECTED

            # Start consumer task
            self._stop_consuming = False
            self._consumer_task = asyncio.create_task(self._consumer_loop())

        except Exception as e:
            self._connection_state = ConnectionState.FAILED
            raise ConnectionError(
                f"Failed to connect to Kafka: {e}",
                broker_type=self.broker_type,
            ) from e

    async def disconnect(self) -> None:
        """Close connection to Kafka cluster."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return

        self._connection_state = ConnectionState.DISCONNECTED

        # Stop consumer task
        self._stop_consuming = True
        if self._consumer_task:
            self._consumer_task.cancel()
            try:
                await self._consumer_task
            except asyncio.CancelledError:
                pass
            self._consumer_task = None

        # Close clients
        loop = asyncio.get_event_loop()

        if self._producer:
            await loop.run_in_executor(
                self._executor, lambda: self._producer.flush(timeout=10)
            )
            self._producer = None

        if self._consumer:
            await loop.run_in_executor(
                self._executor, self._consumer.close
            )
            self._consumer = None

        self._admin_client = None
        self._schema_registry_client = None

        # Clear subscriptions
        self._subscriptions.clear()
        self._subscribed_topics.clear()

    def _build_producer_config(self) -> Dict[str, Any]:
        """Build producer configuration."""
        config = {
            "bootstrap.servers": ",".join(self.bootstrap_servers),
            "client.id": f"{self.client_id}-producer",
            "compression.type": self.compression_type,
            "acks": self.acks,
            "retries": self.retries,
            "batch.size": self._config.get("batch_size", 16384),
            "linger.ms": self._config.get("linger_ms", 0),
            "buffer.memory": self._config.get("buffer_memory", 33554432),
            "max.request.size": self._config.get("max_request_size", 1048576),
            "request.timeout.ms": self._config.get("request_timeout_ms", 30000),
            "delivery.timeout.ms": self._config.get("delivery_timeout_ms", 120000),
            "security.protocol": self.security_protocol,
        }

        # Add transactional configuration
        if self._enable_transactions:
            config["transactional.id"] = self._transactional_id
            config["enable.idempotence"] = True

        # Add security configuration
        self._add_security_config(config)

        return config

    def _build_consumer_config(self) -> Dict[str, Any]:
        """Build consumer configuration."""
        config = {
            "bootstrap.servers": ",".join(self.bootstrap_servers),
            "client.id": f"{self.client_id}-consumer",
            "group.id": self.group_id,
            "auto.offset.reset": self.auto_offset_reset,
            "enable.auto.commit": self.enable_auto_commit,
            "auto.commit.interval.ms": self._config.get("auto_commit_interval_ms", 5000),
            "session.timeout.ms": self._config.get("session_timeout_ms", 30000),
            "heartbeat.interval.ms": self._config.get("heartbeat_interval_ms", 3000),
            "max.poll.records": self._config.get("max_poll_records", 500),
            "security.protocol": self.security_protocol,
        }

        # Add security configuration
        self._add_security_config(config)

        return config

    def _build_admin_config(self) -> Dict[str, Any]:
        """Build admin client configuration."""
        config = {
            "bootstrap.servers": ",".join(self.bootstrap_servers),
            "client.id": f"{self.client_id}-admin",
            "security.protocol": self.security_protocol,
        }

        # Add security configuration
        self._add_security_config(config)

        return config

    def _add_security_config(self, config: Dict[str, Any]) -> None:
        """Add security configuration to Kafka config."""
        # SSL configuration
        if self.security_protocol in ["SSL", "SASL_SSL"]:
            ssl_config = {
                "ssl.ca.location": self._config.get("ssl_ca_location"),
                "ssl.certificate.location": self._config.get("ssl_certificate_location"),
                "ssl.key.location": self._config.get("ssl_key_location"),
                "ssl.key.password": self._config.get("ssl_key_password"),
            }
            config.update({k: v for k, v in ssl_config.items() if v is not None})

        # SASL configuration
        if self.security_protocol in ["SASL_PLAINTEXT", "SASL_SSL"]:
            sasl_config = {
                "sasl.mechanism": self._config.get("sasl_mechanism", "PLAIN"),
                "sasl.username": self._config.get("sasl_username"),
                "sasl.password": self._config.get("sasl_password"),
            }
            config.update({k: v for k, v in sasl_config.items() if v is not None})

    async def _test_connection(self) -> None:
        """Test Kafka connection by getting cluster metadata."""
        loop = asyncio.get_event_loop()

        def get_metadata():
            metadata = self._admin_client.list_topics(timeout=10)
            if metadata.topics is None:
                raise ConnectionError("Failed to get cluster metadata")
            return metadata

        await loop.run_in_executor(self._executor, get_metadata)

    async def dispatch(
        self,
        pipe: str,
        data: Any,
        headers: Optional[Dict[str, Any]] = None,
        metadata: Optional[MessageMetadata] = None,
    ) -> None:
        """Send a message to the specified Kafka topic."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to Kafka cluster",
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

            # Prepare Kafka headers
            kafka_headers = []
            if headers:
                for key, value in headers.items():
                    if isinstance(value, str):
                        kafka_headers.append((key, value.encode('utf-8')))
                    elif isinstance(value, bytes):
                        kafka_headers.append((key, value))
                    else:
                        kafka_headers.append((key, str(value).encode('utf-8')))

            # Add metadata headers
            kafka_headers.extend([
                ("hybridpipe_message_id", metadata.message_id.encode('utf-8')),
                ("hybridpipe_timestamp", metadata.timestamp.isoformat().encode('utf-8')),
                ("hybridpipe_broker_type", str(metadata.broker_type.value).encode('utf-8')),
            ])

            # Produce message
            loop = asyncio.get_event_loop()

            def produce_message():
                self._producer.produce(
                    topic=pipe,
                    value=serialized_data,
                    headers=kafka_headers,
                    callback=self._delivery_callback,
                )
                self._producer.poll(0)  # Trigger delivery callbacks

            await loop.run_in_executor(self._executor, produce_message)
            self._messages_produced += 1

        except Exception as e:
            self._produce_errors += 1
            raise ProtocolError(
                f"Failed to produce message to Kafka topic '{pipe}': {e}",
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
        """Subscribe to messages from the specified Kafka topic."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to Kafka cluster",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        # Add callback to subscriptions
        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
        self._subscriptions[pipe].append(callback)

        # Subscribe to topic if not already subscribed
        if pipe not in self._subscribed_topics:
            self._subscribed_topics.add(pipe)
            await self._update_consumer_subscription()

    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified Kafka topic."""
        if pipe in self._subscriptions:
            del self._subscriptions[pipe]

        if pipe in self._subscribed_topics:
            self._subscribed_topics.remove(pipe)
            await self._update_consumer_subscription()

    async def _update_consumer_subscription(self) -> None:
        """Update consumer topic subscription."""
        if not self._consumer or not self._subscribed_topics:
            return

        loop = asyncio.get_event_loop()

        def update_subscription():
            with self._consumer_lock:
                self._consumer.subscribe(list(self._subscribed_topics))

        await loop.run_in_executor(self._executor, update_subscription)

    async def _consumer_loop(self) -> None:
        """Background task for consuming messages."""
        while not self._stop_consuming and self._consumer:
            try:
                await self._poll_messages()
                await asyncio.sleep(0.01)  # Small delay to prevent busy waiting

            except asyncio.CancelledError:
                break
            except Exception as e:
                self._consume_errors += 1
                print(f"Error in Kafka consumer loop: {e}")
                await asyncio.sleep(1)  # Longer delay on error

    async def _poll_messages(self) -> None:
        """Poll for messages from Kafka."""
        if not self._consumer:
            return

        loop = asyncio.get_event_loop()

        def poll_consumer():
            with self._consumer_lock:
                return self._consumer.poll(timeout=1.0)

        message = await loop.run_in_executor(self._executor, poll_consumer)

        if message is None:
            return

        if message.error():
            if message.error().code() == KafkaError._PARTITION_EOF:
                # End of partition - not an error
                return
            else:
                self._consume_errors += 1
                raise ProtocolError(
                    f"Kafka consumer error: {message.error()}",
                    broker_type=self.broker_type,
                )

        # Process message
        await self._process_message(message)
        self._messages_consumed += 1

    async def _process_message(self, message) -> None:
        """Process a received Kafka message."""
        topic = message.topic()

        if topic not in self._subscriptions:
            return

        try:
            # Extract headers
            headers = {}
            if message.headers():
                for key, value in message.headers():
                    if isinstance(value, bytes):
                        headers[key] = value.decode('utf-8', errors='ignore')
                    else:
                        headers[key] = str(value)

            # Create metadata
            metadata = MessageMetadata(
                message_id=headers.get("hybridpipe_message_id", str(uuid.uuid4())),
                timestamp=datetime.fromisoformat(
                    headers.get("hybridpipe_timestamp", datetime.now().isoformat())
                ),
                pipe=topic,
                broker_type=self.broker_type,
                serialization_format=SerializationFormat.JSON,
                size_bytes=len(message.value()) if message.value() else 0,
                headers=headers,
            )

            # Get message data
            data = message.value() if message.value() else b""

            # Call all subscribers
            callbacks = self._subscriptions[topic][:]  # Copy to avoid modification during iteration
            for callback in callbacks:
                try:
                    result = callback(data, metadata)
                    if asyncio.iscoroutine(result):
                        await result
                except Exception as e:
                    print(f"Error in Kafka message callback for topic {topic}: {e}")

        except Exception as e:
            print(f"Error processing Kafka message for topic {topic}: {e}")

    def _delivery_callback(self, err, msg) -> None:
        """Callback for message delivery confirmation."""
        if err is not None:
            self._produce_errors += 1
            topic = msg.topic() if msg else "unknown"
            print(f"Kafka message delivery failed for topic {topic}: {err}")

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()

        # Add Kafka-specific health information
        uptime = time.time() - self._start_time

        # Get cluster metadata if connected
        cluster_info = {}
        if self.is_connected and self._admin_client:
            try:
                loop = asyncio.get_event_loop()
                metadata = await loop.run_in_executor(
                    self._executor,
                    lambda: self._admin_client.list_topics(timeout=5)
                )
                cluster_info = {
                    "cluster_id": getattr(metadata, "cluster_id", "unknown"),
                    "broker_count": len(metadata.brokers) if metadata.brokers else 0,
                    "topic_count": len(metadata.topics) if metadata.topics else 0,
                }
            except Exception:
                cluster_info = {"error": "Failed to get cluster metadata"}

        base_health.update({
            "uptime_seconds": uptime,
            "messages_produced": self._messages_produced,
            "messages_consumed": self._messages_consumed,
            "produce_errors": self._produce_errors,
            "consume_errors": self._consume_errors,
            "subscribed_topics": list(self._subscribed_topics),
            "subscription_count": len(self._subscriptions),
            "cluster_info": cluster_info,
            "config": {
                "bootstrap_servers": self.bootstrap_servers,
                "client_id": self.client_id,
                "group_id": self.group_id,
                "security_protocol": self.security_protocol,
                "enable_transactions": self._enable_transactions,
            },
        })

        return base_health

    # Transaction support methods
    async def begin_transaction(self) -> None:
        """Begin a Kafka transaction."""
        if not self._enable_transactions:
            raise ProtocolError(
                "Transactions not enabled. Set transactional_id in config.",
                broker_type=self.broker_type,
            )

        if not self.is_connected:
            raise ConnectionError(
                "Not connected to Kafka cluster",
                broker_type=self.broker_type,
            )

        loop = asyncio.get_event_loop()
        await loop.run_in_executor(self._executor, self._producer.begin_transaction)

    async def commit_transaction(self) -> None:
        """Commit the current Kafka transaction."""
        if not self._enable_transactions:
            raise ProtocolError(
                "Transactions not enabled",
                broker_type=self.broker_type,
            )

        loop = asyncio.get_event_loop()
        await loop.run_in_executor(self._executor, self._producer.commit_transaction)

    async def abort_transaction(self) -> None:
        """Abort the current Kafka transaction."""
        if not self._enable_transactions:
            raise ProtocolError(
                "Transactions not enabled",
                broker_type=self.broker_type,
            )

        loop = asyncio.get_event_loop()
        await loop.run_in_executor(self._executor, self._producer.abort_transaction)

    # Topic management methods
    async def create_topic(
        self,
        topic: str,
        num_partitions: int = 1,
        replication_factor: int = 1,
        config: Optional[Dict[str, str]] = None,
    ) -> None:
        """Create a Kafka topic."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to Kafka cluster",
                broker_type=self.broker_type,
            )

        new_topic = NewTopic(
            topic=topic,
            num_partitions=num_partitions,
            replication_factor=replication_factor,
            config=config or {},
        )

        loop = asyncio.get_event_loop()

        def create_topics():
            futures = self._admin_client.create_topics([new_topic])
            for topic_name, future in futures.items():
                try:
                    future.result(timeout=10)
                except Exception as e:
                    if "already exists" not in str(e).lower():
                        raise

        await loop.run_in_executor(self._executor, create_topics)

    async def delete_topic(self, topic: str) -> None:
        """Delete a Kafka topic."""
        if not self.is_connected:
            raise ConnectionError(
                "Not connected to Kafka cluster",
                broker_type=self.broker_type,
            )

        loop = asyncio.get_event_loop()

        def delete_topics():
            futures = self._admin_client.delete_topics([topic])
            for topic_name, future in futures.items():
                future.result(timeout=10)

        await loop.run_in_executor(self._executor, delete_topics)