#!/usr/bin/env python3
"""
Phase 2 Protocol Demonstration

This script demonstrates the usage of all Phase 2 protocols:
- Kafka with transactions
- RabbitMQ with exchanges
- MQTT with QoS levels

Run with: python examples/phase2_demo.py
"""

import asyncio
import json
from datetime import datetime
from typing import Dict, Any

# Import HybridPipe components
try:
    from hybridpipe import deploy_router, BrokerType
    from hybridpipe.core.types import MessageMetadata
except ImportError as e:
    print(f"❌ HybridPipe import failed: {e}")
    print("Please install HybridPipe: pip install -e .")
    exit(1)


async def demo_kafka():
    """Demonstrate Kafka protocol with transactions."""
    print("\n🔥 Kafka Protocol Demo")
    print("-" * 30)
    
    try:
        # Configure Kafka
        config = {
            "bootstrap_servers": ["localhost:9092"],
            "group_id": "hybridpipe-demo",
            "auto_offset_reset": "latest",
            "transactional_id": "demo-transactions",  # Enable transactions
        }
        
        router = await deploy_router(BrokerType.KAFKA, config=config)
        
        async with router:
            print("✅ Connected to Kafka")
            
            # Demonstrate transactional messaging
            print("📤 Starting transaction...")
            await router.begin_transaction()
            
            # Send multiple messages in transaction
            messages = [
                {"event": "user_signup", "user_id": 123, "timestamp": datetime.now().isoformat()},
                {"event": "email_sent", "user_id": 123, "template": "welcome"},
                {"event": "analytics_track", "user_id": 123, "action": "signup_complete"},
            ]
            
            for msg in messages:
                await router.dispatch("user.events", msg)
                print(f"  📨 Sent: {msg['event']}")
            
            # Commit transaction
            await router.commit_transaction()
            print("✅ Transaction committed")
            
            # Demonstrate subscription
            received_messages = []
            
            async def kafka_handler(data: bytes, metadata: MessageMetadata):
                message = json.loads(data.decode())
                received_messages.append(message)
                print(f"  📥 Received: {message['event']} (ID: {metadata.message_id})")
            
            await router.subscribe("user.events", kafka_handler)
            print("👂 Subscribed to user.events")
            
            # Wait a moment for messages
            await asyncio.sleep(2)
            
            print(f"📊 Processed {len(received_messages)} messages")
            
    except ImportError:
        print("⚠️  Kafka dependencies not installed (pip install confluent-kafka)")
    except Exception as e:
        print(f"❌ Kafka demo failed: {e}")


async def demo_rabbitmq():
    """Demonstrate RabbitMQ protocol with exchanges."""
    print("\n🐰 RabbitMQ Protocol Demo")
    print("-" * 30)
    
    try:
        # Configure RabbitMQ
        config = {
            "host": "localhost",
            "port": 5672,
            "username": "guest",
            "password": "guest",
            "virtual_host": "/",
            "exchange_name": "hybridpipe.demo",
            "exchange_type": "topic",
        }
        
        router = await deploy_router(BrokerType.RABBITMQ, config=config)
        
        async with router:
            print("✅ Connected to RabbitMQ")
            
            # Create exchange
            await router.create_exchange("hybridpipe.demo", exchange_type="topic")
            print("🔄 Created topic exchange")
            
            # Demonstrate routing patterns
            routing_keys = [
                "orders.created",
                "orders.updated", 
                "orders.cancelled",
                "payments.processed",
                "inventory.updated",
            ]
            
            # Send messages with different routing keys
            for routing_key in routing_keys:
                message = {
                    "routing_key": routing_key,
                    "timestamp": datetime.now().isoformat(),
                    "data": f"Sample data for {routing_key}",
                }
                
                await router.dispatch(routing_key, message)
                print(f"  📨 Sent to {routing_key}")
            
            # Demonstrate wildcard subscriptions
            received_orders = []
            received_payments = []
            
            async def orders_handler(data: bytes, metadata: MessageMetadata):
                message = json.loads(data.decode())
                received_orders.append(message)
                print(f"  📦 Order event: {message['routing_key']}")
            
            async def payments_handler(data: bytes, metadata: MessageMetadata):
                message = json.loads(data.decode())
                received_payments.append(message)
                print(f"  💳 Payment event: {message['routing_key']}")
            
            # Subscribe with wildcards
            await router.subscribe("orders.*", orders_handler)
            await router.subscribe("payments.*", payments_handler)
            print("👂 Subscribed to orders.* and payments.*")
            
            # Wait for messages
            await asyncio.sleep(2)
            
            print(f"📊 Received {len(received_orders)} order events")
            print(f"📊 Received {len(received_payments)} payment events")
            
    except ImportError:
        print("⚠️  RabbitMQ dependencies not installed (pip install aio-pika)")
    except Exception as e:
        print(f"❌ RabbitMQ demo failed: {e}")


async def demo_mqtt():
    """Demonstrate MQTT protocol with QoS levels."""
    print("\n📡 MQTT Protocol Demo")
    print("-" * 30)
    
    try:
        # Configure MQTT
        config = {
            "host": "localhost",
            "port": 1883,
            "client_id": "hybridpipe-demo",
            "clean_session": True,
            "default_qos": 1,  # At least once delivery
        }
        
        router = await deploy_router(BrokerType.MQTT, config=config)
        
        async with router:
            print("✅ Connected to MQTT broker")
            
            # Demonstrate different QoS levels
            qos_levels = [0, 1, 2]  # At most once, at least once, exactly once
            
            for qos in qos_levels:
                message = {
                    "sensor": "temperature",
                    "value": 23.5 + qos,  # Different values for each QoS
                    "qos": qos,
                    "timestamp": datetime.now().isoformat(),
                }
                
                await router.dispatch(
                    f"sensors/temperature/qos{qos}",
                    message,
                    headers={"qos": qos}
                )
                print(f"  📨 Sent with QoS {qos}")
            
            # Demonstrate topic wildcards
            received_sensors = []
            received_devices = []
            
            async def sensor_handler(data: bytes, metadata: MessageMetadata):
                message = json.loads(data.decode())
                received_sensors.append(message)
                print(f"  🌡️  Sensor reading: {message['value']}°C (QoS {message['qos']})")
            
            async def device_handler(data: bytes, metadata: MessageMetadata):
                message = json.loads(data.decode())
                received_devices.append(message)
                print(f"  📱 Device status: {message['status']}")
            
            # Subscribe with MQTT wildcards
            await router.subscribe("sensors/+/+", sensor_handler)  # Single level wildcard
            await router.subscribe("devices/#", device_handler)    # Multi level wildcard
            print("👂 Subscribed to sensors/+/+ and devices/#")
            
            # Send some device messages
            device_messages = [
                {"topic": "devices/gateway/status", "status": "online"},
                {"topic": "devices/sensor1/status", "status": "active"},
                {"topic": "devices/sensor2/battery", "level": 85},
            ]
            
            for msg in device_messages:
                await router.dispatch(msg["topic"], msg)
                print(f"  📨 Sent device message to {msg['topic']}")
            
            # Wait for messages
            await asyncio.sleep(2)
            
            print(f"📊 Received {len(received_sensors)} sensor readings")
            print(f"📊 Received {len(received_devices)} device messages")
            
    except ImportError:
        print("⚠️  MQTT dependencies not installed (pip install paho-mqtt)")
    except Exception as e:
        print(f"❌ MQTT demo failed: {e}")


async def demo_performance():
    """Demonstrate performance capabilities."""
    print("\n⚡ Performance Demo")
    print("-" * 30)
    
    try:
        # Use Mock protocol for performance testing
        router = await deploy_router(BrokerType.MOCK)
        
        async with router:
            print("✅ Connected to Mock broker")
            
            # Measure throughput
            import time
            
            message_count = 10000
            start_time = time.time()
            
            for i in range(message_count):
                await router.dispatch("perf.test", {"id": i, "data": f"message_{i}"})
            
            end_time = time.time()
            duration = end_time - start_time
            throughput = message_count / duration
            
            print(f"📊 Sent {message_count} messages in {duration:.2f} seconds")
            print(f"🚀 Throughput: {throughput:.0f} messages/second")
            
            # Test subscription performance
            received_count = 0
            
            async def perf_handler(data: bytes, metadata: MessageMetadata):
                nonlocal received_count
                received_count += 1
            
            await router.subscribe("perf.test", perf_handler)
            
            # Send more messages
            start_time = time.time()
            for i in range(1000):
                await router.dispatch("perf.test", {"id": i})
            
            # Wait for processing
            await asyncio.sleep(0.1)
            
            end_time = time.time()
            duration = end_time - start_time
            
            print(f"📥 Processed {received_count} messages")
            print(f"⚡ Processing rate: {received_count/duration:.0f} messages/second")
            
    except Exception as e:
        print(f"❌ Performance demo failed: {e}")


async def main():
    """Run all Phase 2 protocol demonstrations."""
    print("🎉 HybridPipe Phase 2 Protocol Demonstration")
    print("=" * 50)
    print("This demo showcases Kafka, RabbitMQ, and MQTT protocols")
    print("Make sure the respective brokers are running:")
    print("  - Kafka: localhost:9092")
    print("  - RabbitMQ: localhost:5672")
    print("  - MQTT: localhost:1883")
    print("\nOr run: docker-compose -f docker-compose.test.yml up -d")
    print("=" * 50)
    
    # Run all demos
    await demo_kafka()
    await demo_rabbitmq()
    await demo_mqtt()
    await demo_performance()
    
    print("\n🎉 Phase 2 Protocol Demonstration Complete!")
    print("\nKey Features Demonstrated:")
    print("✅ Kafka transactions and consumer groups")
    print("✅ RabbitMQ exchanges and routing patterns")
    print("✅ MQTT QoS levels and topic wildcards")
    print("✅ High-performance async messaging")
    print("✅ Comprehensive error handling")
    print("✅ Production-ready features")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\n👋 Demo interrupted by user")
    except Exception as e:
        print(f"\n❌ Demo failed: {e}")
        import traceback
        traceback.print_exc()
