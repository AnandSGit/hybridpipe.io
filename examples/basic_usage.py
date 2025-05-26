#!/usr/bin/env python3
"""
Basic usage examples for HybridPipe.

This script demonstrates the core functionality of HybridPipe
including message dispatch, subscription, and different protocols.
"""

import asyncio
import json
import time
import sys
import os
from typing import Any
from datetime import datetime

# Add the parent directory to the path so we can import hybridpipe
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from hybridpipe import (
    deploy_router,
    BrokerType,
    MessageMetadata,
    LoggingMiddleware,
    MonitoringMiddleware,
    MiddlewareStack,
)


async def basic_mock_example():
    """Basic example using the Mock protocol."""
    print("=== Basic Mock Protocol Example ===")

    # Deploy a mock router
    router = await deploy_router(BrokerType.MOCK)

    # Message storage for demonstration
    received_messages = []

    async def message_handler(data: bytes, metadata: MessageMetadata):
        """Handle received messages."""
        from hybridpipe.serialization.engine import decode
        message = decode(data)
        received_messages.append({
            "data": message,
            "metadata": {
                "message_id": metadata.message_id,
                "timestamp": metadata.timestamp.isoformat(),
                "pipe": metadata.pipe,
            }
        })
        print(f"Received message: {message}")

    try:
        # Subscribe to a channel
        await router.subscribe("user.events", message_handler)
        print("Subscribed to 'user.events' channel")

        # Send some messages
        messages = [
            {"event": "user_login", "user_id": 123, "timestamp": datetime.now().isoformat()},
            {"event": "user_logout", "user_id": 123, "timestamp": datetime.now().isoformat()},
            {"event": "user_purchase", "user_id": 456, "amount": 99.99},
        ]

        for msg in messages:
            await router.dispatch("user.events", msg)
            print(f"Sent message: {msg}")

        # Wait for message delivery
        await asyncio.sleep(0.1)

        print(f"Total messages received: {len(received_messages)}")

        # Health check
        health = await router.health_check()
        print(f"Router health: {health['connection_state']}")

    finally:
        await router.disconnect()
        print("Disconnected from router")


async def tcp_server_client_example():
    """Example using TCP protocol in server/client mode."""
    print("\n=== TCP Server/Client Example ===")

    # Start TCP server
    server_config = {
        "host": "localhost",
        "port": 0,  # Let OS choose port
        "server_mode": True,
    }

    server = await deploy_router(BrokerType.TCP, config=server_config, auto_connect=True)

    # Get the assigned port
    server_port = server._server.sockets[0].getsockname()[1]
    print(f"TCP server started on port {server_port}")

    # Start TCP client
    client_config = {
        "host": "localhost",
        "port": server_port,
        "server_mode": False,
    }

    client = await deploy_router(BrokerType.TCP, config=client_config, auto_connect=True)
    print("TCP client connected")

    # Message handling
    server_messages = []
    client_messages = []

    async def server_handler(data: bytes, metadata: MessageMetadata):
        from hybridpipe.serialization.engine import decode
        message = decode(data)
        server_messages.append(message)
        print(f"Server received: {message}")

    async def client_handler(data: bytes, metadata: MessageMetadata):
        from hybridpipe.serialization.engine import decode
        message = decode(data)
        client_messages.append(message)
        print(f"Client received: {message}")

    try:
        # Subscribe on both ends
        await server.subscribe("server.channel", server_handler)
        await client.subscribe("client.channel", client_handler)

        # Send messages
        await client.dispatch("server.channel", {"from": "client", "message": "Hello server!"})
        await server.dispatch("client.channel", {"from": "server", "message": "Hello client!"})

        # Wait for delivery
        await asyncio.sleep(0.2)

        print(f"Server received {len(server_messages)} messages")
        print(f"Client received {len(client_messages)} messages")

    finally:
        await client.disconnect()
        await server.disconnect()
        print("TCP connections closed")


async def middleware_example():
    """Example demonstrating middleware usage."""
    print("\n=== Middleware Example ===")

    # Create router with middleware
    router = await deploy_router(BrokerType.MOCK, auto_connect=False)

    # Create middleware stack
    middleware_stack = MiddlewareStack()

    # Add logging middleware
    logging_middleware = LoggingMiddleware(
        log_level="INFO",
        log_data=True,
        max_data_length=100,
    )
    middleware_stack.add_middleware(logging_middleware)

    # Add monitoring middleware
    monitoring_middleware = MonitoringMiddleware(
        track_latency=True,
        track_throughput=True,
    )
    middleware_stack.add_middleware(monitoring_middleware)

    # Set middleware on router
    router.set_middleware_stack(middleware_stack)

    # Connect and use router
    await router.connect()

    received_messages = []

    async def message_handler(data: bytes, metadata: MessageMetadata):
        from hybridpipe.serialization.engine import decode
        message = decode(data)
        received_messages.append(message)

    try:
        await router.subscribe("monitored.channel", message_handler)

        # Send multiple messages to generate metrics
        for i in range(5):
            await router.dispatch("monitored.channel", {
                "message_number": i,
                "content": f"This is message {i}",
                "timestamp": datetime.now().isoformat(),
            })

        # Wait for processing
        await asyncio.sleep(0.2)

        # Get metrics
        metrics = monitoring_middleware.get_metrics()
        print(f"Messages processed: {metrics.get('total_messages', 0)}")
        print(f"Error rate: {metrics.get('error_rate', 0):.2%}")
        print(f"Throughput: {metrics.get('throughput_mps', 0):.2f} msgs/sec")

    finally:
        await router.disconnect()


async def performance_benchmark():
    """Simple performance benchmark."""
    print("\n=== Performance Benchmark ===")

    router = await deploy_router(BrokerType.MOCK)

    message_count = 1000
    received_count = 0

    async def counter_handler(data: bytes, metadata: MessageMetadata):
        nonlocal received_count
        received_count += 1

    try:
        await router.subscribe("benchmark.channel", counter_handler)

        # Benchmark message sending
        start_time = time.time()

        for i in range(message_count):
            await router.dispatch("benchmark.channel", {
                "id": i,
                "data": f"benchmark message {i}",
            })

        send_time = time.time() - start_time

        # Wait for all messages to be delivered
        await asyncio.sleep(0.5)

        total_time = time.time() - start_time

        print(f"Sent {message_count} messages in {send_time:.3f} seconds")
        print(f"Send rate: {message_count / send_time:.0f} msgs/sec")
        print(f"Received {received_count} messages in {total_time:.3f} seconds")
        print(f"Total rate: {received_count / total_time:.0f} msgs/sec")

    finally:
        await router.disconnect()


async def error_handling_example():
    """Example demonstrating error handling."""
    print("\n=== Error Handling Example ===")

    # Try to connect to non-existent Redis
    try:
        router = await deploy_router(
            BrokerType.REDIS,
            config={"host": "localhost", "port": 9999},
            auto_connect=True
        )
    except Exception as e:
        print(f"Expected connection error: {type(e).__name__}: {e}")

    # Try to use disconnected router
    router = await deploy_router(BrokerType.MOCK, auto_connect=False)

    try:
        await router.dispatch("test.channel", {"message": "test"})
    except Exception as e:
        print(f"Expected disconnection error: {type(e).__name__}: {e}")

    # Test timeout
    await router.connect()
    try:
        # This should work fine
        await router.dispatch_with_timeout(
            "test.channel",
            {"message": "test"},
            timeout_seconds=1.0
        )
        print("Timeout test passed")
    finally:
        await router.disconnect()


async def main():
    """Run all examples."""
    print("HybridPipe Basic Usage Examples")
    print("=" * 40)

    try:
        await basic_mock_example()
        await tcp_server_client_example()
        await middleware_example()
        await performance_benchmark()
        await error_handling_example()

        print("\n=== All Examples Completed Successfully ===")

    except Exception as e:
        print(f"Error running examples: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    # Run the examples
    asyncio.run(main())
