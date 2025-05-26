"""
Tests for Phase 2 protocol implementations.

This module tests Kafka, RabbitMQ, and MQTT protocol implementations.
These tests require the respective message brokers to be running.
"""

import pytest
import asyncio
import json
from typing import List, Any
from datetime import datetime

from hybridpipe.core.types import BrokerType, MessageMetadata
from hybridpipe.core.errors import ConnectionError, ProtocolError
from hybridpipe import deploy_router


class TestKafkaProtocol:
    """Test the Kafka protocol implementation."""
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_kafka_connection_failure(self):
        """Test Kafka connection failure."""
        config = {
            "bootstrap_servers": ["localhost:9999"],  # Non-existent port
            "connection_timeout": 1.0,
        }
        
        try:
            router = await deploy_router(BrokerType.KAFKA, config=config)
            
            with pytest.raises(ConnectionError):
                await router.connect()
                
        except ImportError:
            pytest.skip("confluent-kafka-python not available")
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_kafka_basic_functionality(self):
        """Test basic Kafka functionality (requires running Kafka)."""
        try:
            config = {
                "bootstrap_servers": ["localhost:9092"],
                "group_id": "hybridpipe-test",
                "auto_offset_reset": "latest",
            }
            
            router = await deploy_router(BrokerType.KAFKA, config=config)
            
            try:
                await router.connect()
                
                # Test basic dispatch
                await router.dispatch("test.topic", {"message": "kafka test"})
                
                # Test health check
                health = await router.health_check()
                assert health["broker_type"] == "KAFKA"
                assert health["is_connected"] is True
                
                # Test topic creation
                await router.create_topic("test.topic", num_partitions=1)
                
            except ConnectionError:
                pytest.skip("Kafka server not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("confluent-kafka-python not available")
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_kafka_producer_consumer(self):
        """Test Kafka producer/consumer functionality."""
        try:
            config = {
                "bootstrap_servers": ["localhost:9092"],
                "group_id": "hybridpipe-test-consumer",
                "auto_offset_reset": "latest",
            }
            
            router = await deploy_router(BrokerType.KAFKA, config=config)
            
            received_messages = []
            
            async def message_handler(data: bytes, metadata: MessageMetadata):
                from hybridpipe.serialization.engine import decode
                message = decode(data)
                received_messages.append(message)
            
            try:
                await router.connect()
                
                # Subscribe to topic
                await router.subscribe("test.messages", message_handler)
                
                # Wait a moment for subscription to be established
                await asyncio.sleep(1.0)
                
                # Send test messages
                test_messages = [
                    {"id": 1, "content": "First message"},
                    {"id": 2, "content": "Second message"},
                    {"id": 3, "content": "Third message"},
                ]
                
                for msg in test_messages:
                    await router.dispatch("test.messages", msg)
                
                # Wait for messages to be consumed
                await asyncio.sleep(2.0)
                
                # Check that messages were received
                assert len(received_messages) >= len(test_messages)
                
            except ConnectionError:
                pytest.skip("Kafka server not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("confluent-kafka-python not available")
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_kafka_transactions(self):
        """Test Kafka transaction support."""
        try:
            config = {
                "bootstrap_servers": ["localhost:9092"],
                "transactional_id": "hybridpipe-test-tx",
                "group_id": "hybridpipe-test-tx-group",
            }
            
            router = await deploy_router(BrokerType.KAFKA, config=config)
            
            try:
                await router.connect()
                
                # Test transaction operations
                await router.begin_transaction()
                await router.dispatch("test.tx.topic", {"message": "transactional"})
                await router.commit_transaction()
                
                # Test transaction abort
                await router.begin_transaction()
                await router.dispatch("test.tx.topic", {"message": "aborted"})
                await router.abort_transaction()
                
            except ConnectionError:
                pytest.skip("Kafka server not available")
            except ProtocolError as e:
                if "transactions not enabled" in str(e).lower():
                    pytest.skip("Kafka transactions not configured")
                raise
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("confluent-kafka-python not available")


class TestRabbitMQProtocol:
    """Test the RabbitMQ protocol implementation."""
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_rabbitmq_connection_failure(self):
        """Test RabbitMQ connection failure."""
        config = {
            "host": "localhost",
            "port": 9999,  # Non-existent port
            "connection_timeout": 1.0,
        }
        
        try:
            router = await deploy_router(BrokerType.RABBITMQ, config=config)
            
            with pytest.raises(ConnectionError):
                await router.connect()
                
        except ImportError:
            pytest.skip("aio-pika not available")
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_rabbitmq_basic_functionality(self):
        """Test basic RabbitMQ functionality (requires running RabbitMQ)."""
        try:
            config = {
                "host": "localhost",
                "port": 5672,
                "username": "guest",
                "password": "guest",
                "virtual_host": "/",
            }
            
            router = await deploy_router(BrokerType.RABBITMQ, config=config)
            
            try:
                await router.connect()
                
                # Test basic dispatch
                await router.dispatch("test.routing.key", {"message": "rabbitmq test"})
                
                # Test health check
                health = await router.health_check()
                assert health["broker_type"] == "RABBITMQ"
                assert health["is_connected"] is True
                
                # Test exchange creation
                await router.create_exchange("test.exchange", exchange_type="topic")
                
            except ConnectionError:
                pytest.skip("RabbitMQ server not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("aio-pika not available")
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_rabbitmq_publisher_subscriber(self):
        """Test RabbitMQ publisher/subscriber functionality."""
        try:
            config = {
                "host": "localhost",
                "port": 5672,
                "username": "guest",
                "password": "guest",
                "exchange_name": "hybridpipe.test",
                "queue_auto_delete": True,
            }
            
            router = await deploy_router(BrokerType.RABBITMQ, config=config)
            
            received_messages = []
            
            async def message_handler(data: bytes, metadata: MessageMetadata):
                from hybridpipe.serialization.engine import decode
                message = decode(data)
                received_messages.append(message)
            
            try:
                await router.connect()
                
                # Subscribe to routing key pattern
                await router.subscribe("test.messages.*", message_handler)
                
                # Wait a moment for subscription to be established
                await asyncio.sleep(1.0)
                
                # Send test messages
                test_messages = [
                    {"id": 1, "content": "First message"},
                    {"id": 2, "content": "Second message"},
                    {"id": 3, "content": "Third message"},
                ]
                
                for i, msg in enumerate(test_messages):
                    await router.dispatch(f"test.messages.{i}", msg)
                
                # Wait for messages to be consumed
                await asyncio.sleep(2.0)
                
                # Check that messages were received
                assert len(received_messages) >= len(test_messages)
                
            except ConnectionError:
                pytest.skip("RabbitMQ server not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("aio-pika not available")


class TestMQTTProtocol:
    """Test the MQTT protocol implementation."""
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_mqtt_connection_failure(self):
        """Test MQTT connection failure."""
        config = {
            "host": "localhost",
            "port": 9999,  # Non-existent port
            "keepalive": 5,
        }
        
        try:
            router = await deploy_router(BrokerType.MQTT, config=config)
            
            with pytest.raises(ConnectionError):
                await router.connect()
                
        except ImportError:
            pytest.skip("paho-mqtt not available")
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_mqtt_basic_functionality(self):
        """Test basic MQTT functionality (requires running MQTT broker)."""
        try:
            config = {
                "host": "localhost",
                "port": 1883,
                "client_id": "hybridpipe-test",
                "clean_session": True,
            }
            
            router = await deploy_router(BrokerType.MQTT, config=config)
            
            try:
                await router.connect()
                
                # Test basic dispatch
                await router.dispatch("test/topic", {"message": "mqtt test"})
                
                # Test health check
                health = await router.health_check()
                assert health["broker_type"] == "MQTT"
                assert health["is_connected"] is True
                
            except ConnectionError:
                pytest.skip("MQTT broker not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("paho-mqtt not available")
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_mqtt_publisher_subscriber(self):
        """Test MQTT publisher/subscriber functionality."""
        try:
            config = {
                "host": "localhost",
                "port": 1883,
                "client_id": "hybridpipe-test-pubsub",
                "default_qos": 1,
            }
            
            router = await deploy_router(BrokerType.MQTT, config=config)
            
            received_messages = []
            
            async def message_handler(data: bytes, metadata: MessageMetadata):
                from hybridpipe.serialization.engine import decode
                try:
                    message = decode(data)
                    received_messages.append(message)
                except:
                    # Handle raw data
                    received_messages.append({"raw": data.decode()})
            
            try:
                await router.connect()
                
                # Subscribe to topic pattern
                await router.subscribe("test/messages/+", message_handler)
                
                # Wait a moment for subscription to be established
                await asyncio.sleep(1.0)
                
                # Send test messages
                test_messages = [
                    {"id": 1, "content": "First message"},
                    {"id": 2, "content": "Second message"},
                    {"id": 3, "content": "Third message"},
                ]
                
                for i, msg in enumerate(test_messages):
                    await router.dispatch(f"test/messages/{i}", msg)
                
                # Wait for messages to be consumed
                await asyncio.sleep(2.0)
                
                # Check that messages were received
                assert len(received_messages) >= len(test_messages)
                
            except ConnectionError:
                pytest.skip("MQTT broker not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("paho-mqtt not available")
    
    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_mqtt_qos_levels(self):
        """Test MQTT QoS levels."""
        try:
            config = {
                "host": "localhost",
                "port": 1883,
                "client_id": "hybridpipe-test-qos",
            }
            
            router = await deploy_router(BrokerType.MQTT, config=config)
            
            try:
                await router.connect()
                
                # Test different QoS levels
                for qos in [0, 1, 2]:
                    await router.dispatch(
                        "test/qos",
                        {"message": f"QoS {qos} test"},
                        headers={"qos": qos}
                    )
                
            except ConnectionError:
                pytest.skip("MQTT broker not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass
                    
        except ImportError:
            pytest.skip("paho-mqtt not available")


if __name__ == "__main__":
    pytest.main([__file__])
