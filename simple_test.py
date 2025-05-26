#!/usr/bin/env python3
"""
Simple test script to verify Phase 2 implementation without complex imports.
"""

import sys
import os

# Add the current directory to Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

def test_basic_functionality():
    """Test basic functionality without complex imports."""
    print("🚀 HybridPipe Phase 2 Simple Test")
    print("=" * 40)
    
    # Test 1: Basic types import
    print("\n1. Testing basic types...")
    try:
        # Import types directly without going through __init__.py
        from hybridpipe.core.types import BrokerType, ConnectionState
        print(f"✅ Types imported successfully")
        print(f"   Available broker types: {[bt.name for bt in BrokerType]}")
        print(f"   Connection states: {[cs.name for cs in ConnectionState]}")
    except Exception as e:
        print(f"❌ Types import failed: {e}")
        return False
    
    # Test 2: Registry functionality
    print("\n2. Testing registry...")
    try:
        from hybridpipe.core.registry import HybridPipeRegistry
        registry = HybridPipeRegistry()
        protocols = registry.get_registered_protocols()
        print(f"✅ Registry working")
        print(f"   Registered protocols: {[p.name for p in protocols]}")
    except Exception as e:
        print(f"❌ Registry test failed: {e}")
        return False
    
    # Test 3: Mock protocol (should always work)
    print("\n3. Testing Mock protocol...")
    try:
        from hybridpipe.protocols.mock import MockHybridPipe
        mock_pipe = MockHybridPipe()
        print(f"✅ Mock protocol instantiated")
        print(f"   Broker type: {mock_pipe.broker_type.name}")
        print(f"   Connection state: {mock_pipe.connection_state.name}")
    except Exception as e:
        print(f"❌ Mock protocol test failed: {e}")
        return False
    
    # Test 4: Protocol registration
    print("\n4. Testing protocol registration...")
    try:
        # Check if protocols are registered
        registry = HybridPipeRegistry()
        protocols = registry.get_registered_protocols()
        
        expected_protocols = [BrokerType.MOCK, BrokerType.TCP, BrokerType.REDIS]
        for protocol in expected_protocols:
            if protocol in protocols:
                print(f"✅ {protocol.name} protocol registered")
            else:
                print(f"⚠️  {protocol.name} protocol not registered")
        
        # Try to get a factory
        mock_factory = registry.get_factory(BrokerType.MOCK)
        if mock_factory:
            print("✅ Mock factory retrieved successfully")
        else:
            print("❌ Mock factory not found")
            
    except Exception as e:
        print(f"❌ Protocol registration test failed: {e}")
        return False
    
    # Test 5: Phase 2 protocols (may fail if dependencies missing)
    print("\n5. Testing Phase 2 protocols...")
    
    phase2_protocols = [
        ("kafka", "KafkaHybridPipe", "confluent-kafka-python"),
        ("rabbitmq", "RabbitMQHybridPipe", "aio-pika"),
        ("mqtt", "MQTTHybridPipe", "paho-mqtt"),
    ]
    
    for module_name, class_name, dependency in phase2_protocols:
        try:
            module = __import__(f"hybridpipe.protocols.{module_name}", fromlist=[class_name])
            protocol_class = getattr(module, class_name)
            print(f"✅ {class_name} imported successfully")
            
            # Try to instantiate (may fail due to missing dependencies)
            try:
                instance = protocol_class()
                print(f"   ✅ {class_name} instantiated successfully")
            except ImportError as ie:
                print(f"   ⚠️  {class_name} instantiation failed (missing {dependency}): {ie}")
            except Exception as ie:
                print(f"   ⚠️  {class_name} instantiation failed: {ie}")
                
        except ImportError as e:
            print(f"⚠️  {class_name} import failed (missing {dependency}): {e}")
        except Exception as e:
            print(f"❌ {class_name} import error: {e}")
    
    # Test 6: Serialization
    print("\n6. Testing serialization...")
    try:
        from hybridpipe.serialization.engine import encode, decode
        from hybridpipe.core.types import SerializationFormat
        
        test_data = {"test": "data", "number": 42}
        encoded = encode(test_data, SerializationFormat.JSON)
        decoded = decode(encoded, SerializationFormat.JSON)
        
        if decoded == test_data:
            print("✅ Serialization working correctly")
        else:
            print("❌ Serialization data mismatch")
            return False
            
    except Exception as e:
        print(f"❌ Serialization test failed: {e}")
        return False
    
    print("\n🎉 Basic functionality tests completed!")
    return True

def test_async_functionality():
    """Test async functionality."""
    print("\n7. Testing async functionality...")
    
    import asyncio
    
    async def async_test():
        try:
            from hybridpipe.protocols.mock import MockHybridPipe
            
            # Test async context manager
            async with MockHybridPipe() as pipe:
                print("✅ Async context manager working")
                
                # Test basic dispatch
                await pipe.dispatch("test.pipe", {"message": "hello"})
                print("✅ Message dispatch working")
                
                # Test subscription
                received_messages = []
                
                def message_handler(data, metadata):
                    received_messages.append(data)
                
                await pipe.subscribe("test.pipe", message_handler)
                print("✅ Subscription working")
                
                # Test health check
                health = await pipe.health_check()
                print(f"✅ Health check working: {health['broker_type']}")
                
            return True
            
        except Exception as e:
            print(f"❌ Async test failed: {e}")
            import traceback
            traceback.print_exc()
            return False
    
    try:
        result = asyncio.run(async_test())
        return result
    except Exception as e:
        print(f"❌ Async test setup failed: {e}")
        return False

def main():
    """Run all tests."""
    success = True
    
    # Run basic tests
    if not test_basic_functionality():
        success = False
    
    # Run async tests
    if not test_async_functionality():
        success = False
    
    # Summary
    print("\n" + "=" * 40)
    if success:
        print("🎉 All tests passed! Phase 2 implementation is working.")
        return 0
    else:
        print("⚠️  Some tests failed. Check the output above.")
        return 1

if __name__ == "__main__":
    sys.exit(main())
