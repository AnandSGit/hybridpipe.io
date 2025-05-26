"""
Monitoring middleware for HybridPipe.

This module provides middleware for collecting metrics and monitoring
message processing performance, throughput, and error rates.
"""

import time
from typing import List, Optional, Dict, Any
from collections import defaultdict, deque
import threading

from hybridpipe.middleware.base import Middleware, MiddlewareContext, MiddlewareStage
from hybridpipe.core.types import BrokerType

__all__ = ["MonitoringMiddleware", "MetricsCollector"]


class MetricsCollector:
    """
    Thread-safe metrics collector for HybridPipe operations.
    
    Collects and aggregates metrics about message processing,
    including throughput, latency, error rates, and broker usage.
    """
    
    def __init__(self, max_history: int = 1000) -> None:
        """
        Initialize metrics collector.
        
        Args:
            max_history: Maximum number of historical data points to keep
        """
        self.max_history = max_history
        self._lock = threading.RLock()
        
        # Counters
        self._message_counts: Dict[str, int] = defaultdict(int)
        self._error_counts: Dict[str, int] = defaultdict(int)
        self._broker_counts: Dict[BrokerType, int] = defaultdict(int)
        
        # Timing data
        self._latencies: Dict[str, deque] = defaultdict(lambda: deque(maxlen=max_history))
        self._throughput_history: deque = deque(maxlen=max_history)
        
        # Current state
        self._active_operations: Dict[str, float] = {}
        self._start_time = time.time()
        
        # Prometheus metrics (if available)
        self._prometheus_metrics = self._initialize_prometheus_metrics()
    
    def _initialize_prometheus_metrics(self) -> Optional[Dict[str, Any]]:
        """Initialize Prometheus metrics if available."""
        try:
            from prometheus_client import Counter, Histogram, Gauge, Info
            
            return {
                "messages_total": Counter(
                    "hybridpipe_messages_total",
                    "Total number of messages processed",
                    ["broker_type", "pipe", "stage"]
                ),
                "message_duration_seconds": Histogram(
                    "hybridpipe_message_duration_seconds",
                    "Time spent processing messages",
                    ["broker_type", "pipe", "operation"]
                ),
                "errors_total": Counter(
                    "hybridpipe_errors_total",
                    "Total number of errors",
                    ["broker_type", "pipe", "error_type"]
                ),
                "active_connections": Gauge(
                    "hybridpipe_active_connections",
                    "Number of active connections",
                    ["broker_type"]
                ),
                "info": Info(
                    "hybridpipe_info",
                    "HybridPipe information"
                ),
            }
        except ImportError:
            return None
    
    def record_message(
        self,
        broker_type: BrokerType,
        pipe: str,
        stage: MiddlewareStage,
        duration_ms: Optional[float] = None,
    ) -> None:
        """
        Record a message processing event.
        
        Args:
            broker_type: The broker type
            pipe: The pipe name
            stage: The processing stage
            duration_ms: Optional duration in milliseconds
        """
        with self._lock:
            key = f"{broker_type.name}:{pipe}:{stage.value}"
            self._message_counts[key] += 1
            self._broker_counts[broker_type] += 1
            
            if duration_ms is not None:
                self._latencies[key].append(duration_ms)
            
            # Update Prometheus metrics
            if self._prometheus_metrics:
                self._prometheus_metrics["messages_total"].labels(
                    broker_type=broker_type.name,
                    pipe=pipe,
                    stage=stage.value
                ).inc()
                
                if duration_ms is not None:
                    self._prometheus_metrics["message_duration_seconds"].labels(
                        broker_type=broker_type.name,
                        pipe=pipe,
                        operation=stage.value
                    ).observe(duration_ms / 1000.0)
    
    def record_error(
        self,
        broker_type: BrokerType,
        pipe: str,
        error_type: str,
    ) -> None:
        """
        Record an error event.
        
        Args:
            broker_type: The broker type
            pipe: The pipe name
            error_type: The type of error
        """
        with self._lock:
            key = f"{broker_type.name}:{pipe}:{error_type}"
            self._error_counts[key] += 1
            
            # Update Prometheus metrics
            if self._prometheus_metrics:
                self._prometheus_metrics["errors_total"].labels(
                    broker_type=broker_type.name,
                    pipe=pipe,
                    error_type=error_type
                ).inc()
    
    def start_operation(self, operation_id: str) -> None:
        """Start timing an operation."""
        with self._lock:
            self._active_operations[operation_id] = time.time()
    
    def end_operation(self, operation_id: str) -> Optional[float]:
        """
        End timing an operation and return duration.
        
        Args:
            operation_id: The operation identifier
        
        Returns:
            Duration in milliseconds, or None if operation wasn't started
        """
        with self._lock:
            start_time = self._active_operations.pop(operation_id, None)
            if start_time:
                return (time.time() - start_time) * 1000
            return None
    
    def get_metrics_summary(self) -> Dict[str, Any]:
        """
        Get a summary of collected metrics.
        
        Returns:
            Dictionary containing metrics summary
        """
        with self._lock:
            total_messages = sum(self._message_counts.values())
            total_errors = sum(self._error_counts.values())
            uptime_seconds = time.time() - self._start_time
            
            # Calculate average latencies
            avg_latencies = {}
            for key, latencies in self._latencies.items():
                if latencies:
                    avg_latencies[key] = sum(latencies) / len(latencies)
            
            # Calculate throughput (messages per second)
            throughput = total_messages / uptime_seconds if uptime_seconds > 0 else 0
            
            return {
                "uptime_seconds": uptime_seconds,
                "total_messages": total_messages,
                "total_errors": total_errors,
                "error_rate": total_errors / total_messages if total_messages > 0 else 0,
                "throughput_mps": throughput,
                "active_operations": len(self._active_operations),
                "broker_usage": dict(self._broker_counts),
                "average_latencies_ms": avg_latencies,
                "message_counts": dict(self._message_counts),
                "error_counts": dict(self._error_counts),
            }
    
    def reset_metrics(self) -> None:
        """Reset all collected metrics."""
        with self._lock:
            self._message_counts.clear()
            self._error_counts.clear()
            self._broker_counts.clear()
            self._latencies.clear()
            self._throughput_history.clear()
            self._active_operations.clear()
            self._start_time = time.time()


# Global metrics collector instance
_global_metrics_collector: Optional[MetricsCollector] = None
_collector_lock = threading.Lock()


def get_metrics_collector() -> MetricsCollector:
    """Get the global metrics collector instance."""
    global _global_metrics_collector
    if _global_metrics_collector is None:
        with _collector_lock:
            if _global_metrics_collector is None:
                _global_metrics_collector = MetricsCollector()
    return _global_metrics_collector


class MonitoringMiddleware(Middleware):
    """
    Middleware for monitoring message processing metrics.
    
    Automatically collects metrics about message throughput,
    latency, error rates, and broker usage.
    """
    
    def __init__(
        self,
        name: str = "MonitoringMiddleware",
        metrics_collector: Optional[MetricsCollector] = None,
        track_latency: bool = True,
        track_throughput: bool = True,
    ) -> None:
        """
        Initialize monitoring middleware.
        
        Args:
            name: Name of the middleware
            metrics_collector: Custom metrics collector (uses global if None)
            track_latency: Whether to track operation latency
            track_throughput: Whether to track message throughput
        """
        super().__init__(name)
        self.metrics_collector = metrics_collector or get_metrics_collector()
        self.track_latency = track_latency
        self.track_throughput = track_throughput
    
    @property
    def supported_stages(self) -> List[MiddlewareStage]:
        """Support all stages for comprehensive monitoring."""
        return [
            MiddlewareStage.PRE_SERIALIZE,
            MiddlewareStage.POST_SERIALIZE,
            MiddlewareStage.PRE_SEND,
            MiddlewareStage.POST_SEND,
            MiddlewareStage.PRE_RECEIVE,
            MiddlewareStage.POST_RECEIVE,
            MiddlewareStage.PRE_DESERIALIZE,
            MiddlewareStage.POST_DESERIALIZE,
        ]
    
    def _get_operation_id(self, context: MiddlewareContext) -> str:
        """Generate operation ID for timing tracking."""
        return f"{context.metadata.message_id}:{context.stage.value}"
    
    async def process(self, context: MiddlewareContext) -> MiddlewareContext:
        """Process and record metrics for the context."""
        # Start timing for "pre" stages
        if self.track_latency and context.stage.value.startswith("pre_"):
            operation_id = self._get_operation_id(context)
            self.metrics_collector.start_operation(operation_id)
        
        # Record completion and timing for "post" stages
        elif context.stage.value.startswith("post_"):
            duration_ms = None
            
            if self.track_latency:
                # Calculate duration from corresponding "pre" stage
                pre_stage = context.stage.value.replace("post_", "pre_")
                pre_operation_id = f"{context.metadata.message_id}:{pre_stage}"
                duration_ms = self.metrics_collector.end_operation(pre_operation_id)
            
            # Record the message processing event
            self.metrics_collector.record_message(
                broker_type=context.broker_type,
                pipe=context.pipe,
                stage=context.stage,
                duration_ms=duration_ms,
            )
        
        # Record all stages if tracking throughput
        elif self.track_throughput:
            self.metrics_collector.record_message(
                broker_type=context.broker_type,
                pipe=context.pipe,
                stage=context.stage,
            )
        
        return context
    
    async def on_error(self, context: MiddlewareContext, error: Exception) -> None:
        """Record error metrics."""
        self.metrics_collector.record_error(
            broker_type=context.broker_type,
            pipe=context.pipe,
            error_type=type(error).__name__,
        )
        
        # End any active timing operations
        if self.track_latency:
            operation_id = self._get_operation_id(context)
            self.metrics_collector.end_operation(operation_id)
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get current metrics summary."""
        return self.metrics_collector.get_metrics_summary()
    
    def reset_metrics(self) -> None:
        """Reset all metrics."""
        self.metrics_collector.reset_metrics()
