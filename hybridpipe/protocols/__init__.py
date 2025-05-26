"""
Protocol implementations for HybridPipe.

This module contains implementations for various messaging protocols
including Kafka, RabbitMQ, MQTT, NATS, Redis, ZeroMQ, TCP, and more.
"""

# Phase 1 protocols (always available)
from hybridpipe.protocols import mock, tcp, redis

# Phase 2 protocols (optional dependencies)
try:
    from hybridpipe.protocols import kafka
except ImportError:
    pass

try:
    from hybridpipe.protocols import rabbitmq
except ImportError:
    pass

try:
    from hybridpipe.protocols import mqtt
except ImportError:
    pass

# Phase 3 protocols (optional dependencies)
try:
    from hybridpipe.protocols import nats
except ImportError:
    pass

try:
    from hybridpipe.protocols import zeromq
except ImportError:
    pass

try:
    from hybridpipe.protocols import amqp1
except ImportError:
    pass

# Phase 4 protocols (not yet implemented)
# try:
#     from hybridpipe.protocols import nsq
# except ImportError:
#     pass

# try:
#     from hybridpipe.protocols import netchan
# except ImportError:
#     pass

__all__ = [
    "mock",
    "tcp",
    "redis",
    # Optional protocols are not included in __all__
    # to avoid import errors
]
