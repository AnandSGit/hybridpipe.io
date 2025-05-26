"""
Middleware system for HybridPipe.

This module provides a flexible middleware system that allows for
cross-cutting concerns like logging, monitoring, tracing, and
custom message processing to be added to the messaging pipeline.
"""

from hybridpipe.middleware.base import Middleware, MiddlewareStack
from hybridpipe.middleware.logging import LoggingMiddleware
from hybridpipe.middleware.monitoring import MonitoringMiddleware
from hybridpipe.middleware.tracing import TracingMiddleware

__all__ = [
    "Middleware",
    "MiddlewareStack",
    "LoggingMiddleware",
    "MonitoringMiddleware",
    "TracingMiddleware",
]
