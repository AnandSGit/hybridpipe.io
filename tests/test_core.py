"""
Tests for HybridPipe core functionality.

This module tests the core interfaces, registry, serialization,
and configuration management.
"""

import pytest
import asyncio
from typing import Any, Dict, Optional
from datetime import datetime

from hybridpipe.core.interface import HybridPipe
from hybridpipe.core.types import BrokerType, MessageCallback, MessageMetadata, ProtocolCapabilities
from hybridpipe.core.registry import HybridPipeRegistry, deploy_router, register_factory
from hybridpipe.core.errors import BrokerNotSupportedError, ConfigurationError
from hybridpipe.serialization.engine import encode, decode, SerializationOptions
from hybridpipe.core.config import HybridPipeConfig


class TestHybridPipeInterface:
    """Test the core HybridPipe interface."""

    def test_abstract_interface(self):
        """Test that HybridPipe is abstract and cannot be instantiated."""
        with pytest.raises(TypeError):
            HybridPipe()

    def test_broker_type_enum(self):
        """Test BrokerType enumeration."""
        # Phase 1: Core protocols
        assert BrokerType.MOCK == 1
        assert BrokerType.TCP == 2
        assert BrokerType.REDIS == 3

        # Phase 2: Production messaging protocols
        assert BrokerType.KAFKA == 4
        assert BrokerType.RABBITMQ == 5
        assert BrokerType.MQTT == 6

        # Phase 3: Advanced protocols
        assert BrokerType.NATS == 7
        assert BrokerType.ZEROMQ == 8
        assert BrokerType.AMQP1 == 9

    def test_protocol_capabilities(self):
        """Test ProtocolCapabilities dataclass."""
        caps = ProtocolCapabilities(
            supports_persistence=True,
            supports_transactions=False,
            max_message_size=1024,
        )

        assert caps.supports_persistence is True
        assert caps.supports_transactions is False
        assert caps.supports_clustering is False  # Default
        assert caps.max_message_size == 1024


class TestRegistry:
    """Test the HybridPipe registry system."""

    def setup_method(self):
        """Set up test fixtures."""
        self.registry = HybridPipeRegistry()

        # Clear any existing registrations
        for broker_type in list(self.registry.get_registered_protocols()):
            try:
                self.registry.unregister(broker_type)
            except:
                pass

    def test_singleton_pattern(self):
        """Test that registry follows singleton pattern."""
        registry1 = HybridPipeRegistry()
        registry2 = HybridPipeRegistry()
        assert registry1 is registry2

    def test_register_factory(self):
        """Test factory registration."""
        def mock_factory(config: Optional[Dict[str, Any]] = None):
            return MockHybridPipe(config)

        self.registry.register(BrokerType.MOCK, mock_factory)
        assert BrokerType.MOCK in self.registry.get_registered_protocols()

    def test_duplicate_registration(self):
        """Test that duplicate registration raises error."""
        def mock_factory(config: Optional[Dict[str, Any]] = None):
            return MockHybridPipe(config)

        self.registry.register(BrokerType.MOCK, mock_factory)

        with pytest.raises(ValueError):
            self.registry.register(BrokerType.MOCK, mock_factory)

    @pytest.mark.asyncio
    async def test_deploy_router(self):
        """Test router deployment."""
        def mock_factory(config: Optional[Dict[str, Any]] = None):
            return MockHybridPipe(config)

        self.registry.register(BrokerType.MOCK, mock_factory)

        router = await self.registry.deploy_router(BrokerType.MOCK, auto_connect=False)
        assert isinstance(router, MockHybridPipe)
        assert router.broker_type == BrokerType.MOCK

    @pytest.mark.asyncio
    async def test_deploy_unsupported_broker(self):
        """Test deployment of unsupported broker type."""
        with pytest.raises(BrokerNotSupportedError):
            await self.registry.deploy_router(BrokerType.KAFKA)

    def test_unregister(self):
        """Test factory unregistration."""
        def mock_factory(config: Optional[Dict[str, Any]] = None):
            return MockHybridPipe(config)

        self.registry.register(BrokerType.MOCK, mock_factory)
        assert BrokerType.MOCK in self.registry.get_registered_protocols()

        self.registry.unregister(BrokerType.MOCK)
        assert BrokerType.MOCK not in self.registry.get_registered_protocols()


class TestSerialization:
    """Test serialization engine."""

    def test_json_serialization(self):
        """Test JSON serialization."""
        data = {"message": "hello", "number": 42, "list": [1, 2, 3]}

        encoded = encode(data)
        decoded = decode(encoded)

        assert decoded == data

    def test_serialization_with_options(self):
        """Test serialization with custom options."""
        from hybridpipe.serialization.engine import encode_with_options, decode_with_options
        from hybridpipe.core.types import SerializationFormat

        data = {"message": "hello world" * 100}  # Large enough for compression

        options = SerializationOptions(
            format=SerializationFormat.JSON,
            compression=True,
            compression_threshold=50,
        )

        encoded = encode_with_options(data, options)
        decoded = decode_with_options(encoded, options)

        assert decoded == data

    def test_wire_format_compatibility(self):
        """Test wire format header structure."""
        from hybridpipe.serialization.engine import encode_with_options, HEADER_SIZE, MAGIC_BYTES

        data = {"test": "data"}
        options = SerializationOptions()

        encoded = encode_with_options(data, options)

        # Check header structure
        assert len(encoded) >= HEADER_SIZE
        assert encoded[0:2] == MAGIC_BYTES
        assert encoded[2] == 1  # Version

    def test_invalid_data_serialization(self):
        """Test serialization of invalid data."""
        from hybridpipe.core.errors import SerializationError

        # Test with non-serializable object
        class NonSerializable:
            pass

        with pytest.raises(SerializationError):
            encode(NonSerializable())


class TestConfiguration:
    """Test configuration management."""

    def test_default_config(self):
        """Test default configuration."""
        config = HybridPipeConfig()

        assert config.default_broker == BrokerType.MOCK
        assert config.serialization.default_format.value == 1  # JSON
        assert config.monitoring.enabled is True

    def test_config_validation(self):
        """Test configuration validation."""
        # Test invalid port
        with pytest.raises(ValueError):
            HybridPipeConfig(
                redis={"port": 70000}  # Invalid port number
            )

    def test_environment_override(self):
        """Test environment variable override."""
        import os

        # Save original value
        original_value = os.environ.get("HYBRIDPIPE_DEFAULT_BROKER")

        try:
            # Set environment variable
            os.environ["HYBRIDPIPE_DEFAULT_BROKER"] = "1"  # KAFKA

            # Create new config instance
            config = HybridPipeConfig()

            # Debug: print actual values
            print(f"Environment value: {os.environ.get('HYBRIDPIPE_DEFAULT_BROKER')}")
            print(f"Config default_broker: {config.default_broker}")
            print(f"Expected: {BrokerType.KAFKA}")

            assert config.default_broker == BrokerType.KAFKA
        finally:
            # Clean up
            if original_value is not None:
                os.environ["HYBRIDPIPE_DEFAULT_BROKER"] = original_value
            else:
                os.environ.pop("HYBRIDPIPE_DEFAULT_BROKER", None)


# Mock implementation for testing
class MockHybridPipe(HybridPipe):
    """Mock HybridPipe implementation for testing."""

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        super().__init__(config)
        self._connected = False
        self._subscriptions = {}

    @property
    def broker_type(self) -> BrokerType:
        return BrokerType.MOCK

    @property
    def capabilities(self) -> ProtocolCapabilities:
        return ProtocolCapabilities()

    async def connect(self) -> None:
        self._connected = True

    async def disconnect(self) -> None:
        self._connected = False

    @property
    def is_connected(self) -> bool:
        return self._connected

    async def dispatch(self, pipe: str, data: Any, headers=None, metadata=None) -> None:
        if not self._connected:
            from hybridpipe.core.errors import ConnectionError
            raise ConnectionError("Not connected")

    async def dispatch_with_timeout(self, pipe: str, data: Any, timeout_seconds: float, headers=None, metadata=None) -> None:
        await self.dispatch(pipe, data, headers, metadata)

    async def subscribe(self, pipe: str, callback: MessageCallback, headers=None) -> None:
        if not self._connected:
            from hybridpipe.core.errors import ConnectionError
            raise ConnectionError("Not connected")

        if pipe not in self._subscriptions:
            self._subscriptions[pipe] = []
        self._subscriptions[pipe].append(callback)

    async def unsubscribe(self, pipe: str) -> None:
        self._subscriptions.pop(pipe, None)


if __name__ == "__main__":
    pytest.main([__file__])
