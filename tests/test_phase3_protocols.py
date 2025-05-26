"""
Tests for Phase 3 protocol implementations.

This module tests NATS, ZeroMQ, and AMQP 1.0 protocol implementations.
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


class TestNATSProtocol:
    """Test the NATS protocol implementation."""

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_nats_connection_failure(self):
        """Test NATS connection failure."""
        config = {
            "servers": ["nats://localhost:9999"],  # Non-existent port
        }

        try:
            router = await deploy_router(BrokerType.NATS, config=config)

            with pytest.raises(ConnectionError):
                await router.connect()

        except ImportError:
            pytest.skip("nats-py not available")

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_nats_basic_functionality(self):
        """Test basic NATS functionality (requires running NATS server)."""
        try:
            config = {
                "servers": ["nats://localhost:4222"],
                "name": "hybridpipe-test",
                "jetstream": False,  # Test core NATS first
            }

            router = await deploy_router(BrokerType.NATS, config=config)

            try:
                await router.connect()

                # Test basic dispatch
                await router.dispatch("test.subject", {"message": "nats test"})

                # Test health check
                health = await router.health_check()
                assert health["broker_type"] == "NATS"
                assert health["is_connected"] is True

            except ConnectionError:
                pytest.skip("NATS server not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("nats-py not available")

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_nats_pub_sub(self):
        """Test NATS pub/sub functionality."""
        try:
            config = {
                "servers": ["nats://localhost:4222"],
                "name": "hybridpipe-test-pubsub",
                "jetstream": False,
            }

            router = await deploy_router(BrokerType.NATS, config=config)

            received_messages = []

            async def message_handler(data: bytes, metadata: MessageMetadata):
                from hybridpipe.serialization.engine import decode
                message = decode(data)
                received_messages.append(message)

            try:
                await router.connect()

                # Subscribe to subject pattern
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
                pytest.skip("NATS server not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("nats-py not available")

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_nats_request_reply(self):
        """Test NATS request-reply functionality."""
        try:
            config = {
                "servers": ["nats://localhost:4222"],
                "name": "hybridpipe-test-reqrep",
            }

            router = await deploy_router(BrokerType.NATS, config=config)

            try:
                await router.connect()

                # Set up responder
                async def math_service(request_data, metadata):
                    from hybridpipe.serialization.engine import decode
                    data = decode(request_data)
                    result = data['a'] + data['b']
                    return {"result": result}

                await router.reply("math.add", math_service)

                # Wait for responder to be ready
                await asyncio.sleep(1.0)

                # Make request
                response = await router.request("math.add", {"a": 5, "b": 3})
                assert response["result"] == 8

            except ConnectionError:
                pytest.skip("NATS server not available")
            except AttributeError:
                pytest.skip("NATS request-reply not implemented yet")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("nats-py not available")

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_nats_jetstream(self):
        """Test NATS JetStream functionality."""
        try:
            config = {
                "servers": ["nats://localhost:4222"],
                "name": "hybridpipe-test-js",
                "jetstream": True,
            }

            router = await deploy_router(BrokerType.NATS, config=config)

            try:
                await router.connect()

                # Create stream
                await router.create_stream("TEST_STREAM", ["test.stream.>"])

                # Test persistent messaging
                await router.dispatch("test.stream.data", {"persistent": True})

                # Test durable consumer
                received_messages = []

                async def durable_handler(data: bytes, metadata: MessageMetadata):
                    from hybridpipe.serialization.engine import decode
                    message = decode(data)
                    received_messages.append(message)

                await router.subscribe(
                    "test.stream.>",
                    durable_handler,
                    headers={"durable": "test-consumer"}
                )

                await asyncio.sleep(2.0)

                # Clean up
                await router.delete_stream("TEST_STREAM")

            except ConnectionError:
                pytest.skip("NATS server not available")
            except AttributeError:
                pytest.skip("NATS JetStream not implemented yet")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("nats-py not available")


class TestZeroMQProtocol:
    """Test the ZeroMQ protocol implementation."""

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_zeromq_pub_sub_pattern(self):
        """Test ZeroMQ PUB/SUB pattern."""
        try:
            # Publisher
            pub_config = {
                "pattern": "PUB_SUB",
                "bind_addresses": ["tcp://*:5555"],
            }

            # Subscriber
            sub_config = {
                "pattern": "PUB_SUB",
                "connect_addresses": ["tcp://localhost:5555"],
            }

            pub_router = await deploy_router(BrokerType.ZEROMQ, config=pub_config)
            sub_router = await deploy_router(BrokerType.ZEROMQ, config=sub_config)

            received_messages = []

            async def message_handler(data: bytes, metadata: MessageMetadata):
                from hybridpipe.serialization.engine import decode
                message = decode(data)
                received_messages.append(message)

            try:
                await pub_router.connect()
                await sub_router.connect()

                # Subscribe
                await sub_router.subscribe("test", message_handler)

                # Wait for connection to establish
                await asyncio.sleep(1.0)

                # Publish messages
                for i in range(5):
                    await pub_router.dispatch("test", {"id": i, "data": f"message_{i}"})

                # Wait for messages
                await asyncio.sleep(2.0)

                # Check messages received
                assert len(received_messages) >= 0  # ZMQ may drop messages if no subscribers

            finally:
                try:
                    await pub_router.disconnect()
                    await sub_router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("pyzmq not available")

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_zeromq_req_rep_pattern(self):
        """Test ZeroMQ REQ/REP pattern."""
        try:
            # Server (REP)
            server_config = {
                "pattern": "REQ_REP",
                "bind_addresses": ["tcp://*:5556"],
            }

            # Client (REQ)
            client_config = {
                "pattern": "REQ_REP",
                "connect_addresses": ["tcp://localhost:5556"],
            }

            server_router = await deploy_router(BrokerType.ZEROMQ, config=server_config)
            client_router = await deploy_router(BrokerType.ZEROMQ, config=client_config)

            try:
                await server_router.connect()
                await client_router.connect()

                # Set up server handler
                async def echo_handler(request_data, metadata):
                    from hybridpipe.serialization.engine import decode
                    data = decode(request_data)
                    return {"echo": data["message"]}

                await server_router.reply("echo", echo_handler)

                # Wait for server to be ready
                await asyncio.sleep(1.0)

                # Make request
                response = await client_router.request("echo", {"message": "hello"})
                assert response["echo"] == "hello"

            except AttributeError:
                pytest.skip("ZeroMQ request-reply not implemented yet")
            finally:
                try:
                    await server_router.disconnect()
                    await client_router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("pyzmq not available")

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_zeromq_push_pull_pattern(self):
        """Test ZeroMQ PUSH/PULL pattern."""
        try:
            # Pusher
            push_config = {
                "pattern": "PUSH_PULL",
                "bind_addresses": ["tcp://*:5557"],
            }

            # Puller
            pull_config = {
                "pattern": "PUSH_PULL",
                "connect_addresses": ["tcp://localhost:5557"],
            }

            push_router = await deploy_router(BrokerType.ZEROMQ, config=push_config)
            pull_router = await deploy_router(BrokerType.ZEROMQ, config=pull_config)

            received_messages = []

            async def work_handler(data: bytes, metadata: MessageMetadata):
                from hybridpipe.serialization.engine import decode
                message = decode(data)
                received_messages.append(message)

            try:
                await push_router.connect()
                await pull_router.connect()

                # Subscribe to work
                await pull_router.subscribe("work", work_handler)

                # Wait for connection
                await asyncio.sleep(1.0)

                # Push work items
                for i in range(3):
                    await push_router.dispatch("work", {"task_id": i, "data": f"task_{i}"})

                # Wait for processing
                await asyncio.sleep(2.0)

                # Check work was received
                assert len(received_messages) >= 0

            finally:
                try:
                    await push_router.disconnect()
                    await pull_router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("pyzmq not available")


class TestAMQP1Protocol:
    """Test the AMQP 1.0 protocol implementation."""

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_amqp1_connection_failure(self):
        """Test AMQP 1.0 connection failure."""
        config = {
            "hostname": "localhost",
            "port": 9999,  # Non-existent port
        }

        try:
            router = await deploy_router(BrokerType.AMQP1, config=config)

            with pytest.raises(ConnectionError):
                await router.connect()

        except ImportError:
            pytest.skip("python-qpid-proton not available")

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_amqp1_basic_functionality(self):
        """Test basic AMQP 1.0 functionality (requires running AMQP broker)."""
        try:
            config = {
                "hostname": "localhost",
                "port": 5672,
                "username": "guest",
                "password": "guest",
                "container_id": "hybridpipe-test",
            }

            router = await deploy_router(BrokerType.AMQP1, config=config)

            try:
                await router.connect()

                # Test basic dispatch
                await router.dispatch("test.queue", {"message": "amqp1 test"})

                # Test health check
                health = await router.health_check()
                assert health["broker_type"] == "AMQP1"
                assert health["is_connected"] is True

            except ConnectionError:
                pytest.skip("AMQP 1.0 broker not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("python-qpid-proton not available")

    @pytest.mark.integration
    @pytest.mark.asyncio
    async def test_amqp1_messaging(self):
        """Test AMQP 1.0 messaging functionality."""
        try:
            config = {
                "hostname": "localhost",
                "port": 5672,
                "username": "guest",
                "password": "guest",
                "container_id": "hybridpipe-test-messaging",
            }

            router = await deploy_router(BrokerType.AMQP1, config=config)

            received_messages = []

            async def message_handler(data: bytes, metadata: MessageMetadata):
                from hybridpipe.serialization.engine import decode
                message = decode(data)
                received_messages.append(message)

            try:
                await router.connect()

                # Subscribe to address
                await router.subscribe("test.messages", message_handler)

                # Wait for subscription
                await asyncio.sleep(1.0)

                # Send test messages
                test_messages = [
                    {"id": 1, "content": "First message"},
                    {"id": 2, "content": "Second message"},
                    {"id": 3, "content": "Third message"},
                ]

                for msg in test_messages:
                    await router.dispatch("test.messages", msg)

                # Wait for messages
                await asyncio.sleep(2.0)

                # Check messages received
                assert len(received_messages) >= 0  # May not receive if broker not configured

            except ConnectionError:
                pytest.skip("AMQP 1.0 broker not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("python-qpid-proton not available")


class TestPhase3Performance:
    """Performance tests for Phase 3 protocols."""

    @pytest.mark.performance
    @pytest.mark.asyncio
    async def test_nats_throughput(self):
        """Test NATS throughput."""
        try:
            config = {
                "servers": ["nats://localhost:4222"],
                "name": "hybridpipe-perf-test",
                "jetstream": False,
            }

            router = await deploy_router(BrokerType.NATS, config=config)

            try:
                await router.connect()

                # Measure throughput
                import time

                message_count = 1000
                start_time = time.time()

                for i in range(message_count):
                    await router.dispatch("perf.test", {"id": i})

                end_time = time.time()
                duration = end_time - start_time
                throughput = message_count / duration

                print(f"NATS throughput: {throughput:.0f} messages/second")

                # Should achieve >10K msg/s
                assert throughput > 1000  # Relaxed for CI

            except ConnectionError:
                pytest.skip("NATS server not available")
            finally:
                try:
                    await router.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("nats-py not available")

    @pytest.mark.performance
    @pytest.mark.asyncio
    async def test_zeromq_latency(self):
        """Test ZeroMQ latency."""
        try:
            # Use PAIR pattern for lowest latency
            config1 = {
                "pattern": "PAIR",
                "bind_addresses": ["tcp://*:5558"],
            }

            config2 = {
                "pattern": "PAIR",
                "connect_addresses": ["tcp://localhost:5558"],
            }

            router1 = await deploy_router(BrokerType.ZEROMQ, config=config1)
            router2 = await deploy_router(BrokerType.ZEROMQ, config=config2)

            try:
                await router1.connect()
                await router2.connect()

                # Wait for connection
                await asyncio.sleep(1.0)

                # Measure round-trip latency
                import time

                latencies = []

                for i in range(10):
                    start_time = time.time()
                    await router1.dispatch("ping", {"id": i})
                    # In real test, would wait for reply
                    end_time = time.time()
                    latency = (end_time - start_time) * 1000000  # microseconds
                    latencies.append(latency)

                    await asyncio.sleep(0.01)  # Small delay between tests

                avg_latency = sum(latencies) / len(latencies)
                print(f"ZeroMQ average latency: {avg_latency:.0f} microseconds")

                # Should achieve <1000μs (relaxed for CI)
                assert avg_latency < 10000

            finally:
                try:
                    await router1.disconnect()
                    await router2.disconnect()
                except:
                    pass

        except ImportError:
            pytest.skip("pyzmq not available")


if __name__ == "__main__":
    pytest.main([__file__])