"""
HybridPipe - A unified messaging interface for microservices and distributed systems.

This package provides a consistent API for various messaging protocols including
Kafka, RabbitMQ, MQTT, NATS, Redis, ZeroMQ, and more.

Example:
    Basic usage with Redis:

    >>> import asyncio
    >>> from hybridpipe import deploy_router, BrokerType
    >>>
    >>> async def main():
    ...     router = await deploy_router(BrokerType.REDIS)
    ...     await router.dispatch("test.channel", {"message": "Hello World"})
    ...     await router.disconnect()
    >>>
    >>> asyncio.run(main())

Author: Anand S <Anand.S.Main@gmail.com>
License: MIT
Version: 2.0.0
"""

from hybridpipe.core.interface import HybridPipe
from hybridpipe.core.registry import HybridPipeRegistry, deploy_router, register_factory
from hybridpipe.core.types import BrokerType, MessageCallback, SerializationFormat, MessageMetadata
from hybridpipe.core.errors import (
    HybridPipeError,
    ConnectionError,
    SerializationError,
    ProtocolError,
    ConfigurationError,
)
from hybridpipe.serialization.engine import (
    encode,
    decode,
    encode_with_options,
    decode_with_options,
    SerializationOptions,
)
from hybridpipe.middleware.base import Middleware, MiddlewareStack

# Import middleware with optional dependencies
try:
    from hybridpipe.middleware.logging import LoggingMiddleware
except ImportError:
    LoggingMiddleware = None

try:
    from hybridpipe.middleware.monitoring import MonitoringMiddleware
except ImportError:
    MonitoringMiddleware = None

try:
    from hybridpipe.middleware.tracing import TracingMiddleware
except ImportError:
    TracingMiddleware = None
# Import monitoring with optional dependencies
try:
    from hybridpipe.monitoring.metrics import MetricsCollector, get_metrics_collector
except ImportError:
    MetricsCollector = None
    get_metrics_collector = None

try:
    from hybridpipe.monitoring.tracer import Tracer, get_tracer
except ImportError:
    Tracer = None
    get_tracer = None

# Import protocol implementations to register them
# Phase 1 protocols (always available)
from hybridpipe.protocols import mock, tcp, redis

# Phase 2 protocols (optional dependencies)
try:
    from hybridpipe.protocols import kafka, rabbitmq, mqtt
except ImportError:
    pass

# Phase 3 protocols (optional dependencies)
try:
    from hybridpipe.protocols import nats, zeromq, amqp1
except ImportError:
    pass

# Phase 4 protocols (not yet implemented)
# try:
#     from hybridpipe.protocols import nsq, netchan
# except ImportError:
#     pass

__version__ = "2.0.0"
__author__ = "Anand S"
__email__ = "Anand.S.Main@gmail.com"
__license__ = "MIT"

__all__ = [
    # Core interfaces
    "HybridPipe",
    "BrokerType",
    "MessageCallback",
    "SerializationFormat",
    "MessageMetadata",

    # Registry functions
    "deploy_router",
    "register_factory",
    "HybridPipeRegistry",

    # Serialization
    "encode",
    "decode",
    "encode_with_options",
    "decode_with_options",
    "SerializationOptions",

    # Middleware
    "Middleware",
    "MiddlewareStack",
    "LoggingMiddleware",
    "MonitoringMiddleware",
    "TracingMiddleware",

    # Monitoring
    "MetricsCollector",
    "get_metrics_collector",
    "Tracer",
    "get_tracer",

    # Exceptions
    "HybridPipeError",
    "ConnectionError",
    "SerializationError",
    "ProtocolError",
    "ConfigurationError",
]


def get_version() -> str:
    """Get the current version of HybridPipe."""
    return __version__


def get_supported_protocols() -> list[BrokerType]:
    """Get a list of currently supported messaging protocols."""
    registry = HybridPipeRegistry()
    return list(registry.get_registered_protocols())


def initialize_hybridpipe(
    enable_metrics: bool = True,
    enable_tracing: bool = True,
    log_level: str = "INFO"
) -> None:
    """
    Initialize HybridPipe with optional monitoring and logging.

    Args:
        enable_metrics: Whether to enable metrics collection
        enable_tracing: Whether to enable distributed tracing
        log_level: Logging level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
    """
    import logging
    import structlog

    # Configure structured logging
    structlog.configure(
        processors=[
            structlog.stdlib.filter_by_level,
            structlog.stdlib.add_logger_name,
            structlog.stdlib.add_log_level,
            structlog.stdlib.PositionalArgumentsFormatter(),
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.UnicodeDecoder(),
            structlog.processors.JSONRenderer()
        ],
        context_class=dict,
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )

    # Set log level
    logging.basicConfig(level=getattr(logging, log_level.upper()))

    # Initialize metrics if enabled
    if enable_metrics:
        metrics_collector = get_metrics_collector()
        # Metrics collector is always running, no need to start

    # Initialize tracing if enabled
    if enable_tracing:
        tracer = get_tracer()
        # Tracer is always available, no need to start


# Note: Auto-initialization removed to prevent import hangs
# Call initialize_hybridpipe() manually if needed
