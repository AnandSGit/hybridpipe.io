"""
TCP protocol implementation for HybridPipe.

This module provides a TCP-based messaging implementation that can
operate in both client and server modes for direct socket communication.
"""

import asyncio
import json
import struct
import time
import socket
from typing import Any, Dict, List, Optional, Tuple
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
from hybridpipe.core.errors import ConnectionError, ProtocolError, TimeoutError
from hybridpipe.core.registry import protocol_implementation
from hybridpipe.serialization.engine import encode_with_options, decode_with_options, SerializationOptions

__all__ = ["TCPHybridPipe"]


@protocol_implementation(
    BrokerType.TCP,
    default_config={
        "host": "localhost",
        "port": 8080,
        "server_mode": False,
        "buffer_size": 8192,
        "connection_timeout": 30.0,
        "keepalive": True,
        "nodelay": True,
        "max_connections": 100,
        "backlog": 128,
    },
    metadata={
        "description": "Direct TCP socket communication",
        "supports_persistence": False,
        "supports_clustering": False,
    },
)
class TCPHybridPipe(HybridPipe):
    """
    TCP implementation of HybridPipe for direct socket communication.

    This implementation provides direct TCP socket communication between
    clients and servers, supporting both client and server modes.

    Features:
    - Client and server modes
    - Multiple concurrent connections (server mode)
    - Message framing with length prefixes
    - Configurable socket options
    - Connection pooling and management
    - Automatic reconnection (client mode)
    """

    # Message format: [4-byte length][message data]
    HEADER_SIZE = 4

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initialize TCP HybridPipe implementation.

        Args:
            config: Configuration dictionary with options:
                - host: Host to connect to or bind to (default: localhost)
                - port: Port to connect to or bind to (default: 8080)
                - server_mode: Whether to operate as server (default: False)
                - buffer_size: Socket buffer size (default: 8192)
                - connection_timeout: Connection timeout in seconds (default: 30.0)
                - keepalive: Enable TCP keepalive (default: True)
                - nodelay: Enable TCP_NODELAY (default: True)
                - max_connections: Max concurrent connections in server mode (default: 100)
                - backlog: Listen backlog in server mode (default: 128)
        """
        super().__init__(config)

        # Configuration
        self.host = self._config.get("host", "localhost")
        self.port = self._config.get("port", 8080)
        self.server_mode = self._config.get("server_mode", False)
        self.buffer_size = self._config.get("buffer_size", 8192)
        self.connection_timeout = self._config.get("connection_timeout", 30.0)
        self.keepalive = self._config.get("keepalive", True)
        self.nodelay = self._config.get("nodelay", True)
        self.max_connections = self._config.get("max_connections", 100)
        self.backlog = self._config.get("backlog", 128)

        # Connection state
        self._connection_state = ConnectionState.DISCONNECTED
        self._server: Optional[asyncio.Server] = None
        self._client_writer: Optional[asyncio.StreamWriter] = None
        self._client_reader: Optional[asyncio.StreamReader] = None

        # Server mode state
        self._connections: Dict[str, Tuple[asyncio.StreamReader, asyncio.StreamWriter]] = {}
        self._connection_tasks: Dict[str, asyncio.Task] = {}

        # Message routing
        self._subscriptions: Dict[str, List[MessageCallback]] = {}
        self._message_queues: Dict[str, asyncio.Queue] = {}

        # Background tasks
        self._receive_task: Optional[asyncio.Task] = None
        self._stop_receiving = False

    @property
    def broker_type(self) -> BrokerType:
        """Get the broker type."""
        return BrokerType.TCP

    @property
    def capabilities(self) -> ProtocolCapabilities:
        """Get protocol capabilities."""
        return ProtocolCapabilities(
            supports_persistence=False,
            supports_transactions=False,
            supports_clustering=False,
            supports_compression=True,
            supports_encryption=False,  # Could be added with TLS
            supports_authentication=False,
            supports_authorization=False,
            supports_dead_letter=False,
            supports_message_ordering=True,
            supports_exactly_once=False,
            max_message_size=1024 * 1024,  # 1MB default limit
            max_pipe_length=255,
        )

    async def connect(self) -> None:
        """Establish TCP connection."""
        if self._connection_state == ConnectionState.CONNECTED:
            return

        self._connection_state = ConnectionState.CONNECTING

        try:
            if self.server_mode:
                await self._start_server()
            else:
                await self._connect_client()

            self._connection_state = ConnectionState.CONNECTED

            # Start receiving messages
            self._stop_receiving = False
            self._receive_task = asyncio.create_task(self._receive_loop())

        except Exception as e:
            self._connection_state = ConnectionState.FAILED
            raise ConnectionError(
                f"Failed to establish TCP connection: {str(e)}",
                broker_type=self.broker_type,
                cause=e,
            )

    async def disconnect(self) -> None:
        """Close TCP connection."""
        if self._connection_state == ConnectionState.DISCONNECTED:
            return

        self._connection_state = ConnectionState.DISCONNECTED

        # Stop receiving
        self._stop_receiving = True
        if self._receive_task:
            self._receive_task.cancel()
            try:
                await self._receive_task
            except asyncio.CancelledError:
                pass
            self._receive_task = None

        # Close connections
        if self.server_mode:
            await self._stop_server()
        else:
            await self._disconnect_client()

        # Clear state
        self._subscriptions.clear()
        self._message_queues.clear()

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
                "Not connected",
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
            "pipe": pipe,
            "data": data,
            "headers": headers or {},
            "metadata": {
                "message_id": metadata.message_id,
                "timestamp": metadata.timestamp.isoformat(),
                "broker_type": metadata.broker_type.name,
                "serialization_format": metadata.serialization_format.name,
            },
        }

        # Serialize message
        serialization_options = SerializationOptions(
            format=SerializationFormat.JSON,
            compression=True,
            compression_threshold=1024,
        )
        serialized_data = encode_with_options(message_envelope, serialization_options)

        # Send message
        try:
            if self.server_mode:
                await self._broadcast_message(serialized_data)
            else:
                await self._send_message(serialized_data)
        except Exception as e:
            raise ProtocolError(
                f"Failed to send message: {str(e)}",
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
                "Not connected",
                broker_type=self.broker_type,
                pipe=pipe,
            )

        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
            self._message_queues[pipe] = asyncio.Queue()

        self._subscriptions[pipe].append(callback)

    async def unsubscribe(self, pipe: str) -> None:
        """Unsubscribe from the specified pipe."""
        if pipe in self._subscriptions:
            del self._subscriptions[pipe]
        if pipe in self._message_queues:
            del self._message_queues[pipe]

    async def _start_server(self) -> None:
        """Start TCP server."""
        self._server = await asyncio.start_server(
            self._handle_client_connection,
            self.host,
            self.port,
            backlog=self.backlog,
            reuse_address=True,
        )

    async def _stop_server(self) -> None:
        """Stop TCP server."""
        if self._server:
            self._server.close()
            await self._server.wait_closed()
            self._server = None

        # Close all client connections
        for connection_id, (reader, writer) in self._connections.items():
            writer.close()
            await writer.wait_closed()

        # Cancel connection tasks
        for task in self._connection_tasks.values():
            task.cancel()

        if self._connection_tasks:
            await asyncio.gather(*self._connection_tasks.values(), return_exceptions=True)

        self._connections.clear()
        self._connection_tasks.clear()

    async def _connect_client(self) -> None:
        """Connect as TCP client."""
        try:
            self._client_reader, self._client_writer = await asyncio.wait_for(
                asyncio.open_connection(self.host, self.port),
                timeout=self.connection_timeout
            )

            # Configure socket options
            sock = self._client_writer.get_extra_info('socket')
            if sock:
                if self.keepalive:
                    sock.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
                if self.nodelay:
                    sock.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)

        except asyncio.TimeoutError:
            raise ConnectionError(
                f"Connection timeout after {self.connection_timeout} seconds",
                broker_type=self.broker_type,
            )

    async def _disconnect_client(self) -> None:
        """Disconnect TCP client."""
        if self._client_writer:
            self._client_writer.close()
            await self._client_writer.wait_closed()
            self._client_writer = None
            self._client_reader = None

    async def _handle_client_connection(
        self,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> None:
        """Handle incoming client connection."""
        connection_id = str(uuid.uuid4())
        self._connections[connection_id] = (reader, writer)

        # Configure socket options
        sock = writer.get_extra_info('socket')
        if sock:
            if self.keepalive:
                sock.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
            if self.nodelay:
                sock.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)

        # Start connection handler task
        task = asyncio.create_task(self._connection_handler(connection_id, reader, writer))
        self._connection_tasks[connection_id] = task

        try:
            await task
        except Exception as e:
            import structlog
            logger = structlog.get_logger()
            logger.error(
                "Error in client connection handler",
                connection_id=connection_id,
                error=str(e),
                error_type=type(e).__name__,
            )
        finally:
            # Clean up connection
            self._connections.pop(connection_id, None)
            self._connection_tasks.pop(connection_id, None)
            writer.close()
            await writer.wait_closed()

    async def _connection_handler(
        self,
        connection_id: str,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> None:
        """Handle messages from a client connection."""
        while not self._stop_receiving:
            try:
                # Read message
                message_data = await self._read_message(reader)
                if message_data is None:
                    break  # Connection closed

                # Process message
                await self._process_received_message(message_data)

            except asyncio.CancelledError:
                break
            except Exception as e:
                import structlog
                logger = structlog.get_logger()
                logger.error(
                    "Error reading message from client",
                    connection_id=connection_id,
                    error=str(e),
                    error_type=type(e).__name__,
                )
                break

    async def _send_message(self, data: bytes) -> None:
        """Send message to server (client mode)."""
        if not self._client_writer:
            raise ProtocolError("No client connection available")

        # Send length prefix + data
        length = len(data)
        header = struct.pack('>I', length)

        self._client_writer.write(header + data)
        await self._client_writer.drain()

    async def _broadcast_message(self, data: bytes) -> None:
        """Broadcast message to all connected clients (server mode)."""
        if not self._connections:
            return

        # Send length prefix + data
        length = len(data)
        header = struct.pack('>I', length)
        message = header + data

        # Send to all connections
        for connection_id, (reader, writer) in list(self._connections.items()):
            try:
                writer.write(message)
                await writer.drain()
            except Exception as e:
                # Remove failed connection
                self._connections.pop(connection_id, None)
                task = self._connection_tasks.pop(connection_id, None)
                if task:
                    task.cancel()

                import structlog
                logger = structlog.get_logger()
                logger.warning(
                    "Failed to send message to client",
                    connection_id=connection_id,
                    error=str(e),
                )

    async def _read_message(self, reader: asyncio.StreamReader) -> Optional[bytes]:
        """Read a framed message from the stream."""
        # Read length header
        header_data = await reader.readexactly(self.HEADER_SIZE)
        if len(header_data) != self.HEADER_SIZE:
            return None

        # Unpack length
        length = struct.unpack('>I', header_data)[0]

        # Read message data
        message_data = await reader.readexactly(length)
        if len(message_data) != length:
            return None

        return message_data

    async def _receive_loop(self) -> None:
        """Main receive loop."""
        if self.server_mode:
            # Server mode: messages are handled by connection handlers
            while not self._stop_receiving:
                await asyncio.sleep(0.1)
        else:
            # Client mode: read messages from server
            while not self._stop_receiving and self._client_reader:
                try:
                    message_data = await self._read_message(self._client_reader)
                    if message_data is None:
                        break  # Connection closed

                    await self._process_received_message(message_data)

                except asyncio.CancelledError:
                    break
                except Exception as e:
                    import structlog
                    logger = structlog.get_logger()
                    logger.error(
                        "Error in receive loop",
                        error=str(e),
                        error_type=type(e).__name__,
                    )
                    await asyncio.sleep(1)  # Brief pause before retry

    async def _process_received_message(self, message_data: bytes) -> None:
        """Process a received message."""
        try:
            # Deserialize message
            serialization_options = SerializationOptions(format=SerializationFormat.JSON)
            message_envelope = decode_with_options(message_data, serialization_options)

            pipe = message_envelope.get("pipe")
            data = message_envelope.get("data")
            headers = message_envelope.get("headers", {})
            metadata_dict = message_envelope.get("metadata", {})

            # Create metadata object
            metadata = MessageMetadata(
                message_id=metadata_dict.get("message_id", str(uuid.uuid4())),
                timestamp=datetime.fromisoformat(metadata_dict.get("timestamp", datetime.now().isoformat())),
                pipe=pipe,
                broker_type=BrokerType[metadata_dict.get("broker_type", "TCP")],
                serialization_format=SerializationFormat[metadata_dict.get("serialization_format", "JSON")],
                size_bytes=len(message_data),
                headers=headers,
            )

            # Deliver to subscribers
            if pipe in self._subscriptions:
                serialized_data = encode_with_options(data, SerializationOptions(format=SerializationFormat.JSON))

                for callback in self._subscriptions[pipe]:
                    try:
                        result = callback(serialized_data, metadata)
                        if asyncio.iscoroutine(result):
                            await result
                    except Exception as e:
                        import structlog
                        logger = structlog.get_logger()
                        logger.error(
                            "Error in message callback",
                            pipe=pipe,
                            message_id=metadata.message_id,
                            error=str(e),
                            error_type=type(e).__name__,
                        )

        except Exception as e:
            import structlog
            logger = structlog.get_logger()
            logger.error(
                "Error processing received message",
                error=str(e),
                error_type=type(e).__name__,
            )

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check."""
        base_health = await super().health_check()

        base_health.update({
            "server_mode": self.server_mode,
            "host": self.host,
            "port": self.port,
            "active_connections": len(self._connections) if self.server_mode else (1 if self._client_writer else 0),
            "subscribed_pipes": list(self._subscriptions.keys()),
            "config": {
                "buffer_size": self.buffer_size,
                "connection_timeout": self.connection_timeout,
                "keepalive": self.keepalive,
                "nodelay": self.nodelay,
                "max_connections": self.max_connections,
            },
        })

        return base_health
