"""
Metrics collection and reporting for HybridPipe.

This module provides comprehensive metrics collection for monitoring
message processing performance, throughput, and system health.
"""

import time
import threading
from typing import Dict, List, Optional, Any, Callable
from collections import defaultdict, deque
from dataclasses import dataclass, field
from datetime import datetime, timedelta

from hybridpipe.core.types import BrokerType

__all__ = ["MetricsCollector", "MetricPoint", "TimeSeriesMetric", "get_metrics_collector"]


@dataclass
class MetricPoint:
    """A single metric data point."""
    timestamp: datetime
    value: float
    labels: Dict[str, str] = field(default_factory=dict)


class TimeSeriesMetric:
    """Time series metric with automatic aggregation."""
    
    def __init__(self, name: str, max_points: int = 1000) -> None:
        """
        Initialize time series metric.
        
        Args:
            name: Name of the metric
            max_points: Maximum number of data points to retain
        """
        self.name = name
        self.max_points = max_points
        self._points: deque = deque(maxlen=max_points)
        self._lock = threading.RLock()
    
    def add_point(self, value: float, labels: Optional[Dict[str, str]] = None) -> None:
        """Add a data point to the time series."""
        with self._lock:
            point = MetricPoint(
                timestamp=datetime.now(),
                value=value,
                labels=labels or {}
            )
            self._points.append(point)
    
    def get_points(
        self,
        since: Optional[datetime] = None,
        labels: Optional[Dict[str, str]] = None,
    ) -> List[MetricPoint]:
        """
        Get data points matching criteria.
        
        Args:
            since: Only return points after this timestamp
            labels: Only return points with matching labels
        
        Returns:
            List of matching data points
        """
        with self._lock:
            points = list(self._points)
        
        if since:
            points = [p for p in points if p.timestamp >= since]
        
        if labels:
            points = [
                p for p in points
                if all(p.labels.get(k) == v for k, v in labels.items())
            ]
        
        return points
    
    def aggregate(
        self,
        aggregation: str = "avg",
        window: Optional[timedelta] = None,
        labels: Optional[Dict[str, str]] = None,
    ) -> Optional[float]:
        """
        Aggregate metric values.
        
        Args:
            aggregation: Type of aggregation (avg, sum, min, max, count)
            window: Time window for aggregation
            labels: Filter by labels
        
        Returns:
            Aggregated value or None if no data
        """
        since = datetime.now() - window if window else None
        points = self.get_points(since=since, labels=labels)
        
        if not points:
            return None
        
        values = [p.value for p in points]
        
        if aggregation == "avg":
            return sum(values) / len(values)
        elif aggregation == "sum":
            return sum(values)
        elif aggregation == "min":
            return min(values)
        elif aggregation == "max":
            return max(values)
        elif aggregation == "count":
            return len(values)
        else:
            raise ValueError(f"Unknown aggregation type: {aggregation}")


class MetricsCollector:
    """
    Comprehensive metrics collector for HybridPipe.
    
    Collects and aggregates various metrics about message processing,
    system performance, and broker usage.
    """
    
    def __init__(self) -> None:
        """Initialize the metrics collector."""
        self._lock = threading.RLock()
        self._start_time = time.time()
        
        # Time series metrics
        self._time_series: Dict[str, TimeSeriesMetric] = {}
        
        # Counters
        self._counters: Dict[str, int] = defaultdict(int)
        
        # Gauges (current values)
        self._gauges: Dict[str, float] = {}
        
        # Histograms (for latency tracking)
        self._histograms: Dict[str, List[float]] = defaultdict(list)
        
        # Custom metrics
        self._custom_metrics: Dict[str, Any] = {}
        
        # Prometheus metrics (if available)
        self._prometheus_registry = self._setup_prometheus()
    
    def _setup_prometheus(self) -> Optional[Any]:
        """Setup Prometheus metrics if available."""
        try:
            from prometheus_client import CollectorRegistry, Counter, Histogram, Gauge
            
            registry = CollectorRegistry()
            
            # Core metrics
            self._prom_messages_total = Counter(
                "hybridpipe_messages_total",
                "Total messages processed",
                ["broker_type", "pipe", "operation"],
                registry=registry
            )
            
            self._prom_message_duration = Histogram(
                "hybridpipe_message_duration_seconds",
                "Message processing duration",
                ["broker_type", "pipe", "operation"],
                registry=registry
            )
            
            self._prom_errors_total = Counter(
                "hybridpipe_errors_total",
                "Total errors",
                ["broker_type", "pipe", "error_type"],
                registry=registry
            )
            
            self._prom_connections = Gauge(
                "hybridpipe_active_connections",
                "Active connections",
                ["broker_type"],
                registry=registry
            )
            
            return registry
            
        except ImportError:
            return None
    
    def increment_counter(
        self,
        name: str,
        value: int = 1,
        labels: Optional[Dict[str, str]] = None,
    ) -> None:
        """
        Increment a counter metric.
        
        Args:
            name: Counter name
            value: Value to increment by
            labels: Optional labels
        """
        with self._lock:
            key = self._make_key(name, labels)
            self._counters[key] += value
            
            # Add to time series
            if name not in self._time_series:
                self._time_series[name] = TimeSeriesMetric(name)
            self._time_series[name].add_point(self._counters[key], labels)
    
    def set_gauge(
        self,
        name: str,
        value: float,
        labels: Optional[Dict[str, str]] = None,
    ) -> None:
        """
        Set a gauge metric value.
        
        Args:
            name: Gauge name
            value: Current value
            labels: Optional labels
        """
        with self._lock:
            key = self._make_key(name, labels)
            self._gauges[key] = value
            
            # Add to time series
            if name not in self._time_series:
                self._time_series[name] = TimeSeriesMetric(name)
            self._time_series[name].add_point(value, labels)
    
    def record_histogram(
        self,
        name: str,
        value: float,
        labels: Optional[Dict[str, str]] = None,
    ) -> None:
        """
        Record a value in a histogram.
        
        Args:
            name: Histogram name
            value: Value to record
            labels: Optional labels
        """
        with self._lock:
            key = self._make_key(name, labels)
            self._histograms[key].append(value)
            
            # Keep only recent values (last 1000)
            if len(self._histograms[key]) > 1000:
                self._histograms[key] = self._histograms[key][-1000:]
            
            # Add to time series
            if name not in self._time_series:
                self._time_series[name] = TimeSeriesMetric(name)
            self._time_series[name].add_point(value, labels)
    
    def record_message_sent(
        self,
        broker_type: BrokerType,
        pipe: str,
        duration_ms: Optional[float] = None,
    ) -> None:
        """Record a message sent event."""
        labels = {"broker_type": broker_type.name, "pipe": pipe, "operation": "send"}
        
        self.increment_counter("messages_total", labels=labels)
        
        if duration_ms is not None:
            self.record_histogram("message_duration_ms", duration_ms, labels=labels)
        
        # Update Prometheus metrics
        if hasattr(self, "_prom_messages_total"):
            self._prom_messages_total.labels(**labels).inc()
            
            if duration_ms is not None:
                self._prom_message_duration.labels(**labels).observe(duration_ms / 1000.0)
    
    def record_message_received(
        self,
        broker_type: BrokerType,
        pipe: str,
        duration_ms: Optional[float] = None,
    ) -> None:
        """Record a message received event."""
        labels = {"broker_type": broker_type.name, "pipe": pipe, "operation": "receive"}
        
        self.increment_counter("messages_total", labels=labels)
        
        if duration_ms is not None:
            self.record_histogram("message_duration_ms", duration_ms, labels=labels)
        
        # Update Prometheus metrics
        if hasattr(self, "_prom_messages_total"):
            self._prom_messages_total.labels(**labels).inc()
            
            if duration_ms is not None:
                self._prom_message_duration.labels(**labels).observe(duration_ms / 1000.0)
    
    def record_error(
        self,
        broker_type: BrokerType,
        pipe: str,
        error_type: str,
    ) -> None:
        """Record an error event."""
        labels = {"broker_type": broker_type.name, "pipe": pipe, "error_type": error_type}
        
        self.increment_counter("errors_total", labels=labels)
        
        # Update Prometheus metrics
        if hasattr(self, "_prom_errors_total"):
            self._prom_errors_total.labels(**labels).inc()
    
    def set_active_connections(self, broker_type: BrokerType, count: int) -> None:
        """Set the number of active connections for a broker."""
        labels = {"broker_type": broker_type.name}
        
        self.set_gauge("active_connections", count, labels=labels)
        
        # Update Prometheus metrics
        if hasattr(self, "_prom_connections"):
            self._prom_connections.labels(**labels).set(count)
    
    def get_counter(self, name: str, labels: Optional[Dict[str, str]] = None) -> int:
        """Get current counter value."""
        with self._lock:
            key = self._make_key(name, labels)
            return self._counters.get(key, 0)
    
    def get_gauge(self, name: str, labels: Optional[Dict[str, str]] = None) -> Optional[float]:
        """Get current gauge value."""
        with self._lock:
            key = self._make_key(name, labels)
            return self._gauges.get(key)
    
    def get_histogram_stats(
        self,
        name: str,
        labels: Optional[Dict[str, str]] = None,
    ) -> Dict[str, float]:
        """Get histogram statistics."""
        with self._lock:
            key = self._make_key(name, labels)
            values = self._histograms.get(key, [])
            
            if not values:
                return {}
            
            sorted_values = sorted(values)
            count = len(sorted_values)
            
            return {
                "count": count,
                "min": min(sorted_values),
                "max": max(sorted_values),
                "avg": sum(sorted_values) / count,
                "p50": sorted_values[int(count * 0.5)],
                "p90": sorted_values[int(count * 0.9)],
                "p95": sorted_values[int(count * 0.95)],
                "p99": sorted_values[int(count * 0.99)],
            }
    
    def get_time_series(self, name: str) -> Optional[TimeSeriesMetric]:
        """Get time series metric by name."""
        return self._time_series.get(name)
    
    def get_summary(self) -> Dict[str, Any]:
        """Get a comprehensive metrics summary."""
        with self._lock:
            uptime = time.time() - self._start_time
            
            # Calculate totals
            total_messages = sum(
                v for k, v in self._counters.items()
                if k.startswith("messages_total")
            )
            total_errors = sum(
                v for k, v in self._counters.items()
                if k.startswith("errors_total")
            )
            
            # Calculate rates
            message_rate = total_messages / uptime if uptime > 0 else 0
            error_rate = total_errors / total_messages if total_messages > 0 else 0
            
            return {
                "uptime_seconds": uptime,
                "total_messages": total_messages,
                "total_errors": total_errors,
                "message_rate_per_second": message_rate,
                "error_rate": error_rate,
                "counters": dict(self._counters),
                "gauges": dict(self._gauges),
                "histogram_stats": {
                    name: self.get_histogram_stats(name.split(":")[0])
                    for name in self._histograms.keys()
                },
            }
    
    def reset(self) -> None:
        """Reset all metrics."""
        with self._lock:
            self._counters.clear()
            self._gauges.clear()
            self._histograms.clear()
            self._time_series.clear()
            self._custom_metrics.clear()
            self._start_time = time.time()
    
    def _make_key(self, name: str, labels: Optional[Dict[str, str]]) -> str:
        """Create a unique key for a metric with labels."""
        if not labels:
            return name
        
        label_str = ",".join(f"{k}={v}" for k, v in sorted(labels.items()))
        return f"{name}:{label_str}"
    
    def export_prometheus(self) -> Optional[str]:
        """Export metrics in Prometheus format."""
        if not self._prometheus_registry:
            return None
        
        try:
            from prometheus_client import generate_latest
            return generate_latest(self._prometheus_registry).decode('utf-8')
        except ImportError:
            return None


# Global metrics collector
_global_collector: Optional[MetricsCollector] = None
_collector_lock = threading.Lock()


def get_metrics_collector() -> MetricsCollector:
    """Get the global metrics collector instance."""
    global _global_collector
    if _global_collector is None:
        with _collector_lock:
            if _global_collector is None:
                _global_collector = MetricsCollector()
    return _global_collector
