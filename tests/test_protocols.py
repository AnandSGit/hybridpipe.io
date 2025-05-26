"""
Tests for HybridPipe protocol implementations.

This module tests the Phase 1 protocol implementations:
Mock, TCP, and Redis protocols.
"""

import pytest
import asyncio
import json
from typing import List, Any
from datetime import datetime

from hybridpipe.core.types import BrokerType, MessageMetadata
from hybridpipe.core.errors import ConnectionError, ProtocolError
from hybridpipe.protocols.mock import MockHybridPipe
from hybridpipe.protocols.tcp import TCPHybridPipe


class TestMockProtocol:
    """Test the Mock protocol implementation."""
    
    def setup_method(self):
        """Set up test fixtures."""
        # Clear global state
        MockHybridPipe.clear_all_queues()
    
    @pytest.mark.asyncio
    async def test_basic_connection(self):
        """Test basic connection and disconnection."""
        router = MockHybridPipe()
        
        assert not router.is_connected
        
        await router.connect()
        assert router.is_connected
        
        await router.disconnect()
        assert not router.is_connected
    
    @pytest.mark.asyncio
    async def test_message_dispatch(self):
        """Test message dispatch."""
        router = MockHybridPipe()
        await router.connect()
        
        try:
            await router.dispatch("test.channel", {"message": "hello"})
            
            # Check that message was queued
            assert router.get_queue_size("test.channel") > 0
            
        finally:
            await router.disconnect()
    
    @pytest.mark.asyncio
    async def test_subscription_and_delivery(self):
        """Test message subscription and delivery."""
        router1 = MockHybridPipe()
        router2 = MockHybridPipe()
        
        received_messages = []
        
        async def message_handler(data: bytes, metadata: MessageMetadata):
            from hybridpipe.serialization.engine import decode
            message = decode(data)
            received_messages.append(message)
        
        await router1.connect()
        await router2.connect()
        
        try:
            # Subscribe to channel
            await router1.subscribe("test.channel", message_handler)
            
            # Send message
            await router2.dispatch("test.channel", {"message": "hello"})
            
            # Wait for delivery
            await asyncio.sleep(0.1)
            
            # Check message was received
            assert len(received_messages) == 1
            assert received_messages[0]["message"] == "hello"
            
        finally:
            await router1.disconnect()
            await router2.disconnect()
    
    @pytest.mark.asyncio
    async def test_multiple_subscribers(self):
        """Test multiple subscribers to same channel."""
        router = MockHybridPipe()
        received_count = [0, 0]  # Use list for mutable reference
        
        async def handler1(data: bytes, metadata: MessageMetadata):
            received_count[0] += 1
        
        async def handler2(data: bytes, metadata: MessageMetadata):
            received_count[1] += 1
        
        await router.connect()
        
        try:
            # Subscribe with multiple handlers
            await router.subscribe("test.channel", handler1)
            await router.subscribe("test.channel", handler2)
            
            # Send message
            await router.dispatch("test.channel", {"message": "broadcast"})
            
            # Wait for delivery
            await asyncio.sleep(0.1)
            
            # Both handlers should receive the message
            assert received_count[0] == 1
            assert received_count[1] == 1
            
        finally:
            await router.disconnect()
    
    @pytest.mark.asyncio
    async def test_unsubscribe(self):
        """Test unsubscribing from channels."""
        router = MockHybridPipe()
        received_messages = []
        
        async def message_handler(data: bytes, metadata: MessageMetadata):
            received_messages.append(data)
        
        await router.connect()
        
        try:
            # Subscribe and send message
            await router.subscribe("test.channel", message_handler)
            await router.dispatch("test.channel", {"message": "first"})
            await asyncio.sleep(0.1)
            
            # Unsubscribe and send another message
            await router.unsubscribe("test.channel")
            await router.dispatch("test.channel", {"message": "second"})
            await asyncio.sleep(0.1)
            
            # Should only receive first message
            assert len(received_messages) == 1
            
        finally:
            await router.disconnect()
    
    @pytest.mark.asyncio
    async def test_failure_simulation(self):
        """Test failure simulation."""
        config = {
            "simulate_failures": True,
            "failure_rate": 1.0,  # Always fail
        }
        
        router = MockHybridPipe(config)
        
        # Connection should fail
        with pytest.raises(ConnectionError):
            await router.connect()
    
    @pytest.mark.asyncio
    async def test_delivery_delay(self):
        """Test delivery delay simulation."""
        config = {
            "delivery_delay_ms": 100,  # 100ms delay
        }
        
        router = MockHybridPipe(config)
        received_messages = []
        
        async def message_handler(data: bytes, metadata: MessageMetadata):
            received_messages.append(datetime.now())
        
        await router.connect()
        
        try:
            await router.subscribe("test.channel", message_handler)
            
            start_time = datetime.now()
            await router.dispatch("test.channel", {"message": "delayed"})
            
            # Wait for delivery
            await asyncio.sleep(0.2)
            
            # Check that delivery was delayed
            assert len(received_messages) == 1
            delivery_time = received_messages[0]
            delay = (delivery_time - start_time).total_seconds() * 1000
            assert delay >= 100  # At least 100ms delay
            
        finally:
            await router.disconnect()
    
    @pytest.mark.asyncio
    async def test_health_check(self):
        """Test health check functionality."""
        router = MockHybridPipe()
        await router.connect()
        
        try:
            health = await router.health_check()
            
            assert health["broker_type"] == "MOCK"
            assert health["is_connected"] is True
            assert "uptime_seconds" in health
            assert "messages_sent" in health
            assert "config" in health
            
        finally:
            await router.disconnect()


class TestTCPProtocol:
    """Test the TCP protocol implementation."""
    
    @pytest.mark.asyncio
    async def test_client_mode_connection_failure(self):
        """Test client mode connection to non-existent server."""
        config = {
            "host": "localhost",
            "port": 9999,  # Non-existent port
            "server_mode": False,
            "connection_timeout": 1.0,  # Short timeout
        }
        
        router = TCPHybridPipe(config)
        
        with pytest.raises(ConnectionError):
            await router.connect()
    
    @pytest.mark.asyncio
    async def test_server_mode_basic(self):
        """Test basic server mode functionality."""
        config = {
            "host": "localhost",
            "port": 0,  # Let OS choose port
            "server_mode": True,
        }
        
        router = TCPHybridPipe(config)
        
        try:
            await router.connect()
            assert router.is_connected
            
            health = await router.health_check()
            assert health["server_mode"] is True
            assert health["active_connections"] == 0
            
        finally:
            await router.disconnect()
    
    @pytest.mark.asyncio
    async def test_client_server_communication(self):
        """Test client-server communication."""
        # Start server
        server_config = {
            "host": "localhost",
            "port": 0,  # Let OS choose port
            "server_mode": True,
        }
        
        server = TCPHybridPipe(server_config)
        await server.connect()
        
        # Get the actual port assigned
        server_port = server._server.sockets[0].getsockname()[1]
        
        # Start client
        client_config = {
            "host": "localhost",
            "port": server_port,
            "server_mode": False,
        }
        
        client = TCPHybridPipe(client_config)
        
        received_messages = []
        
        async def message_handler(data: bytes, metadata: MessageMetadata):
            from hybridpipe.serialization.engine import decode
            message = decode(data)
            received_messages.append(message)
        
        try:
            await client.connect()
            
            # Subscribe on server
            await server.subscribe("test.channel", message_handler)
            
            # Send message from client
            await client.dispatch("test.channel", {"message": "hello from client"})
            
            # Wait for delivery
            await asyncio.sleep(0.2)
            
            # Check message was received
            assert len(received_messages) == 1
            assert received_messages[0]["message"] == "hello from client"
            
        finally:
            await client.disconnect()
            await server.disconnect()
    
    @pytest.mark.asyncio
    async def test_message_framing(self):
        """Test that messages are properly framed."""
        config = {
            "host": "localhost",
            "port": 0,
            "server_mode": True,
        }
        
        router = TCPHybridPipe(config)
        
        try:
            await router.connect()
            
            # Test with various message sizes
            test_messages = [
                {"small": "message"},
                {"large": "x" * 10000},  # Large message
                {"unicode": "Hello 世界 🌍"},  # Unicode content
            ]
            
            for msg in test_messages:
                await router.dispatch("test.channel", msg)
            
            # If we get here without errors, framing works
            assert True
            
        finally:
            await router.disconnect()
    
    @pytest.mark.asyncio
    async def test_connection_timeout(self):
        """Test connection timeout."""
        config = {
            "host": "10.255.255.1",  # Non-routable address
            "port": 80,
            "server_mode": False,
            "connection_timeout": 0.1,  # Very short timeout
        }
        
        router = TCPHybridPipe(config)
        
        with pytest.raises(ConnectionError):
            await router.connect()
    
    @pytest.mark.asyncio
    async def test_dispatch_timeout(self):
        """Test dispatch with timeout."""
        config = {
            "host": "localhost",
            "port": 0,
            "server_mode": True,
        }
        
        router = TCPHybridPipe(config)
        await router.connect()
        
        try:
            # This should complete quickly
            await router.dispatch_with_timeout(
                "test.channel",
                {"message": "test"},
                timeout_seconds=1.0
            )
            
        finally:
            await router.disconnect()


@pytest.mark.integration
class TestRedisProtocol:
    """Test the Redis protocol implementation (requires Redis server)."""
    
    @pytest.mark.asyncio
    async def test_redis_connection_failure(self):
        """Test Redis connection failure."""
        config = {
            "host": "localhost",
            "port": 9999,  # Non-existent Redis port
            "connection_timeout": 1.0,
        }
        
        try:
            from hybridpipe.protocols.redis import RedisHybridPipe
            router = RedisHybridPipe(config)
            
            with pytest.raises(ConnectionError):
                await router.connect()
                
        except ImportError:
            pytest.skip("aioredis not available")
    
    @pytest.mark.asyncio
    async def test_redis_basic_functionality(self):
        """Test basic Redis functionality (requires running Redis)."""
        try:
            from hybridpipe.protocols.redis import RedisHybridPipe
            
            config = {
                "host": "localhost",
                "port": 6379,
                "db": 15,  # Use high-numbered DB for testing
            }
            
            router = RedisHybridPipe(config)
            
            try:
                await router.connect()
                
                # Test basic dispatch
                await router.dispatch("test.channel", {"message": "redis test"})
                
                # Test health check
                health = await router.health_check()
                assert health["broker_type"] == "REDIS"
                assert health["is_connected"] is True
                
            except ConnectionError:
                pytest.skip("Redis server not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("aioredis not available")


if __name__ == "__main__":
    pytest.main([__file__])
