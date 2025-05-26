#!/usr/bin/env python3
"""
Simple test script to verify HybridPipe functionality.
"""

import sys
import os
import asyncio

# Add current directory to path
sys.path.insert(0, '.')

def test_imports():
    """Test basic imports."""
    print("Testing imports...")
    
    try:
        from enum import IntEnum
        print("✅ enum imported")
        
        # Test enum creation
        class TestEnum(IntEnum):
            TEST = 1
        print("✅ IntEnum works")
        
        # Test our types
        from hybridpipe.core.types import BrokerType
        print("✅ BrokerType imported")
        print(f"BrokerType.MOCK = {BrokerType.MOCK}")
        
        return True
        
    except Exception as e:
        print(f"❌ Import error: {e}")
        import traceback
        traceback.print_exc()
        return False

def test_mock_protocol():
    """Test Mock protocol functionality."""
    print("\nTesting Mock protocol...")
    
    try:
        from hybridpipe.protocols.mock import MockHybridPipe
        print("✅ MockHybridPipe imported")
        
        # Create instance
        router = MockHybridPipe()
        print("✅ MockHybridPipe instance created")
        
        return True
        
    except Exception as e:
        print(f"❌ Mock protocol error: {e}")
        import traceback
        traceback.print_exc()
        return False

async def test_async_functionality():
    """Test async functionality."""
    print("\nTesting async functionality...")
    
    try:
        from hybridpipe.protocols.mock import MockHybridPipe
        
        router = MockHybridPipe()
        print("✅ Router created")
        
        await router.connect()
        print("✅ Router connected")
        
        await router.dispatch("test.channel", {"message": "hello"})
        print("✅ Message dispatched")
        
        await router.disconnect()
        print("✅ Router disconnected")
        
        return True
        
    except Exception as e:
        print(f"❌ Async test error: {e}")
        import traceback
        traceback.print_exc()
        return False

def main():
    """Run all tests."""
    print("HybridPipe Simple Test")
    print("=" * 30)
    
    # Test imports
    if not test_imports():
        return False
    
    # Test mock protocol
    if not test_mock_protocol():
        return False
    
    # Test async functionality
    try:
        result = asyncio.run(test_async_functionality())
        if not result:
            return False
    except Exception as e:
        print(f"❌ Async test failed: {e}")
        return False
    
    print("\n✅ All tests passed!")
    return True

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
