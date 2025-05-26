#!/usr/bin/env python3
"""
Test script for Phase 2 HybridPipe implementation.

This script verifies that all Phase 2 protocols are properly implemented
and can be imported and instantiated.
"""

import sys
import asyncio
from typing import List, Dict, Any

# Add current directory to path
sys.path.insert(0, '.')

def test_imports() -> Dict[str, bool]:
    """Test that all Phase 2 protocols can be imported."""
    results = {}
    
    # Test Kafka
    try:
        from hybridpipe.protocols.kafka import KafkaHybridPipe
        results['Kafka'] = True
        print("✅ Kafka protocol imported successfully")
    except ImportError as e:
        results['Kafka'] = False
        print(f"⚠️  Kafka protocol import failed: {e}")
    
    # Test RabbitMQ
    try:
        from hybridpipe.protocols.rabbitmq import RabbitMQHybridPipe
        results['RabbitMQ'] = True
        print("✅ RabbitMQ protocol imported successfully")
    except ImportError as e:
        results['RabbitMQ'] = False
        print(f"⚠️  RabbitMQ protocol import failed: {e}")
    
    # Test MQTT
    try:
        from hybridpipe.protocols.mqtt import MQTTHybridPipe
        results['MQTT'] = True
        print("✅ MQTT protocol imported successfully")
    except ImportError as e:
        results['MQTT'] = False
        print(f"⚠️  MQTT protocol import failed: {e}")
    
    return results

def test_protocol_registration() -> Dict[str, bool]:
    """Test that protocols are properly registered."""
    results = {}
    
    try:
        from hybridpipe.core.registry import HybridPipeRegistry
        from hybridpipe.core.types import BrokerType
        
        registry = HybridPipeRegistry()
        protocols = registry.get_registered_protocols()
        
        print(f"\nRegistered protocols: {[p.name for p in protocols]}")
        
        # Test core protocols
        core_protocols = [BrokerType.MOCK, BrokerType.TCP, BrokerType.REDIS]
        for protocol in core_protocols:
            if protocol in protocols:
                results[protocol.name] = True
                print(f"✅ {protocol.name} protocol registered")
            else:
                results[protocol.name] = False
                print(f"❌ {protocol.name} protocol not registered")
        
        # Test Phase 2 protocols (may not be registered if dependencies missing)
        phase2_protocols = [BrokerType.KAFKA, BrokerType.RABBITMQ, BrokerType.MQTT]
        for protocol in phase2_protocols:
            if protocol in protocols:
                results[protocol.name] = True
                print(f"✅ {protocol.name} protocol registered")
            else:
                results[protocol.name] = False
                print(f"⚠️  {protocol.name} protocol not registered (dependencies may be missing)")
        
    except Exception as e:
        print(f"❌ Error testing protocol registration: {e}")
        results['registration'] = False
    
    return results

async def test_protocol_instantiation() -> Dict[str, bool]:
    """Test that protocols can be instantiated."""
    results = {}
    
    try:
        from hybridpipe import deploy_router, BrokerType
        
        # Test Mock protocol (should always work)
        try:
            router = await deploy_router(BrokerType.MOCK, auto_connect=False)
            results['Mock_instantiation'] = True
            print("✅ Mock protocol instantiated successfully")
        except Exception as e:
            results['Mock_instantiation'] = False
            print(f"❌ Mock protocol instantiation failed: {e}")
        
        # Test Phase 2 protocols (may fail if dependencies missing)
        phase2_protocols = [
            (BrokerType.KAFKA, "Kafka"),
            (BrokerType.RABBITMQ, "RabbitMQ"),
            (BrokerType.MQTT, "MQTT"),
        ]
        
        for broker_type, name in phase2_protocols:
            try:
                router = await deploy_router(broker_type, auto_connect=False)
                results[f'{name}_instantiation'] = True
                print(f"✅ {name} protocol instantiated successfully")
            except ImportError as e:
                results[f'{name}_instantiation'] = False
                print(f"⚠️  {name} protocol instantiation failed (dependency missing): {e}")
            except Exception as e:
                results[f'{name}_instantiation'] = False
                print(f"❌ {name} protocol instantiation failed: {e}")
        
    except Exception as e:
        print(f"❌ Error testing protocol instantiation: {e}")
        results['instantiation'] = False
    
    return results

def test_cli_functionality() -> bool:
    """Test CLI functionality."""
    try:
        from hybridpipe.cli import cli
        print("✅ CLI module imported successfully")
        return True
    except Exception as e:
        print(f"❌ CLI import failed: {e}")
        return False

def test_configuration() -> bool:
    """Test configuration system."""
    try:
        from hybridpipe.core.config import HybridPipeConfig
        
        config = HybridPipeConfig()
        print(f"✅ Configuration system working, default broker: {config.default_broker.name}")
        return True
    except Exception as e:
        print(f"❌ Configuration test failed: {e}")
        return False

def test_serialization() -> bool:
    """Test serialization system."""
    try:
        from hybridpipe.serialization.engine import encode, decode
        from hybridpipe.core.types import SerializationFormat
        
        test_data = {"test": "data", "number": 42}
        encoded = encode(test_data, SerializationFormat.JSON)
        decoded = decode(encoded, SerializationFormat.JSON)
        
        if decoded == test_data:
            print("✅ Serialization system working")
            return True
        else:
            print("❌ Serialization data mismatch")
            return False
    except Exception as e:
        print(f"❌ Serialization test failed: {e}")
        return False

async def main():
    """Run all tests."""
    print("🚀 HybridPipe Phase 2 Implementation Test")
    print("=" * 50)
    
    # Test imports
    print("\n📦 Testing Protocol Imports")
    import_results = test_imports()
    
    # Test registration
    print("\n🔧 Testing Protocol Registration")
    registration_results = test_protocol_registration()
    
    # Test instantiation
    print("\n⚡ Testing Protocol Instantiation")
    instantiation_results = await test_protocol_instantiation()
    
    # Test CLI
    print("\n🖥️  Testing CLI Functionality")
    cli_result = test_cli_functionality()
    
    # Test configuration
    print("\n⚙️  Testing Configuration System")
    config_result = test_configuration()
    
    # Test serialization
    print("\n📝 Testing Serialization System")
    serialization_result = test_serialization()
    
    # Summary
    print("\n📊 Test Summary")
    print("=" * 30)
    
    all_results = {
        **import_results,
        **registration_results,
        **instantiation_results,
        'CLI': cli_result,
        'Configuration': config_result,
        'Serialization': serialization_result,
    }
    
    passed = sum(1 for result in all_results.values() if result)
    total = len(all_results)
    
    print(f"Tests passed: {passed}/{total}")
    
    if passed == total:
        print("🎉 All tests passed! Phase 2 implementation is complete.")
        return True
    else:
        print("⚠️  Some tests failed. Check dependencies and implementation.")
        return False

if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
