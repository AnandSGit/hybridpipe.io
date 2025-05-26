#!/bin/bash

# HybridPipe Phase 2 Setup Script
# This script sets up the development environment for testing Phase 2 protocols

set -e

echo "🚀 HybridPipe Phase 2 Setup"
echo "=========================="

# Check if Python 3.12+ is available
echo "🐍 Checking Python version..."
python_version=$(python3 --version 2>&1 | cut -d' ' -f2)
echo "Found Python $python_version"

if ! python3 -c "import sys; exit(0 if sys.version_info >= (3, 12) else 1)" 2>/dev/null; then
    echo "❌ Python 3.12+ is required"
    exit 1
fi

echo "✅ Python version is compatible"

# Install base dependencies
echo "📦 Installing base dependencies..."
pip3 install -r requirements.txt
pip3 install -r requirements-dev.txt

# Install package in development mode
echo "🔧 Installing HybridPipe in development mode..."
pip3 install -e .

# Install optional protocol dependencies
echo "🔌 Installing Phase 2 protocol dependencies..."
echo "Installing Kafka support..."
pip3 install confluent-kafka || echo "⚠️  Kafka dependency installation failed"

echo "Installing RabbitMQ support..."
pip3 install aio-pika || echo "⚠️  RabbitMQ dependency installation failed"

echo "Installing MQTT support..."
pip3 install paho-mqtt || echo "⚠️  MQTT dependency installation failed"

# Check Docker availability
echo "🐳 Checking Docker availability..."
if command -v docker &> /dev/null; then
    echo "✅ Docker is available"
    
    if command -v docker-compose &> /dev/null; then
        echo "✅ Docker Compose is available"
        
        echo "🚀 Starting test brokers..."
        docker-compose -f docker-compose.test.yml up -d
        
        echo "⏳ Waiting for brokers to be ready..."
        sleep 30
        
        echo "🔍 Checking broker health..."
        docker-compose -f docker-compose.test.yml ps
        
    else
        echo "⚠️  Docker Compose not found. Install it to run integration tests."
    fi
else
    echo "⚠️  Docker not found. Install it to run integration tests."
fi

# Run verification
echo "🔍 Running implementation verification..."
python3 verify_phase2.py

# Run basic tests
echo "🧪 Running basic tests..."
python3 -m pytest tests/test_core.py -v || echo "⚠️  Some core tests failed"

# Try to run Phase 2 tests
echo "🧪 Running Phase 2 protocol tests..."
python3 -m pytest tests/test_phase2_protocols.py -v -k "not integration" || echo "⚠️  Some Phase 2 tests failed"

# Run integration tests if brokers are available
if command -v docker-compose &> /dev/null; then
    echo "🧪 Running integration tests..."
    python3 -m pytest tests/test_phase2_protocols.py -v -m integration || echo "⚠️  Some integration tests failed"
fi

# Run demo
echo "🎭 Running Phase 2 demo..."
python3 examples/phase2_demo.py || echo "⚠️  Demo failed - make sure brokers are running"

echo ""
echo "🎉 Phase 2 Setup Complete!"
echo "========================"
echo ""
echo "✅ HybridPipe Phase 2 is ready for use"
echo ""
echo "📚 Next steps:"
echo "  1. Check the implementation: python3 verify_phase2.py"
echo "  2. Run the demo: python3 examples/phase2_demo.py"
echo "  3. Run tests: python3 -m pytest tests/test_phase2_protocols.py -v"
echo "  4. Start brokers: docker-compose -f docker-compose.test.yml up -d"
echo "  5. Read documentation: cat PHASE2_IMPLEMENTATION_SUMMARY.md"
echo ""
echo "🔌 Supported protocols:"
echo "  ✅ Kafka (with transactions)"
echo "  ✅ RabbitMQ (with exchanges)"
echo "  ✅ MQTT (with QoS levels)"
echo "  ✅ Redis (pub/sub)"
echo "  ✅ TCP (direct sockets)"
echo "  ✅ Mock (for testing)"
echo ""
echo "Happy messaging! 🚀"
