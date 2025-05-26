"""
Distributed tracing implementation for HybridPipe.

This module provides tracing capabilities for tracking message flows
across distributed systems and service boundaries.
"""

import time
import uuid
import threading
from typing import Dict, List, Optional, Any, ContextManager
from dataclasses import dataclass, field
from datetime import datetime
from contextlib import contextmanager

__all__ = ["Tracer", "Span", "Trace", "TraceStatus", "get_tracer"]


class TraceStatus:
    """Constants for trace status."""
    OK = "OK"
    ERROR = "ERROR"
    TIMEOUT = "TIMEOUT"
    CANCELLED = "CANCELLED"


@dataclass
class Span:
    """
    Represents a single span in a distributed trace.
    
    A span represents a single operation within a trace, such as
    sending a message, processing a message, or calling a service.
    """
    span_id: str
    trace_id: str
    parent_span_id: Optional[str]
    operation_name: str
    start_time: float
    end_time: Optional[float] = None
    status: str = TraceStatus.OK
    tags: Dict[str, Any] = field(default_factory=dict)
    logs: List[Dict[str, Any]] = field(default_factory=list)
    baggage: Dict[str, str] = field(default_factory=dict)
    
    @property
    def duration_ms(self) -> Optional[float]:
        """Get the duration of the span in milliseconds."""
        if self.end_time is None:
            return None
        return (self.end_time - self.start_time) * 1000
    
    @property
    def is_finished(self) -> bool:
        """Check if the span is finished."""
        return self.end_time is not None
    
    def set_tag(self, key: str, value: Any) -> None:
        """Set a tag on the span."""
        self.tags[key] = value
    
    def set_baggage(self, key: str, value: str) -> None:
        """Set baggage on the span."""
        self.baggage[key] = value
    
    def log(self, message: str, **kwargs: Any) -> None:
        """Add a log entry to the span."""
        log_entry = {
            "timestamp": time.time(),
            "message": message,
            **kwargs
        }
        self.logs.append(log_entry)
    
    def finish(self, status: str = TraceStatus.OK) -> None:
        """Finish the span."""
        if self.end_time is None:
            self.end_time = time.time()
            self.status = status
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert span to dictionary representation."""
        return {
            "span_id": self.span_id,
            "trace_id": self.trace_id,
            "parent_span_id": self.parent_span_id,
            "operation_name": self.operation_name,
            "start_time": self.start_time,
            "end_time": self.end_time,
            "duration_ms": self.duration_ms,
            "status": self.status,
            "tags": self.tags,
            "logs": self.logs,
            "baggage": self.baggage,
            "is_finished": self.is_finished,
        }


@dataclass
class Trace:
    """
    Represents a complete distributed trace.
    
    A trace is a collection of spans that represent a single
    request or operation flow across multiple services.
    """
    trace_id: str
    spans: List[Span] = field(default_factory=list)
    start_time: Optional[float] = None
    end_time: Optional[float] = None
    
    @property
    def duration_ms(self) -> Optional[float]:
        """Get the total duration of the trace."""
        if self.start_time is None or self.end_time is None:
            return None
        return (self.end_time - self.start_time) * 1000
    
    @property
    def root_span(self) -> Optional[Span]:
        """Get the root span of the trace."""
        for span in self.spans:
            if span.parent_span_id is None:
                return span
        return None
    
    @property
    def is_complete(self) -> bool:
        """Check if all spans in the trace are finished."""
        return all(span.is_finished for span in self.spans)
    
    def add_span(self, span: Span) -> None:
        """Add a span to the trace."""
        self.spans.append(span)
        
        # Update trace timing
        if self.start_time is None or span.start_time < self.start_time:
            self.start_time = span.start_time
        
        if span.end_time is not None:
            if self.end_time is None or span.end_time > self.end_time:
                self.end_time = span.end_time
    
    def get_span_by_id(self, span_id: str) -> Optional[Span]:
        """Get a span by its ID."""
        for span in self.spans:
            if span.span_id == span_id:
                return span
        return None
    
    def get_child_spans(self, parent_span_id: str) -> List[Span]:
        """Get all child spans of a parent span."""
        return [span for span in self.spans if span.parent_span_id == parent_span_id]
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert trace to dictionary representation."""
        return {
            "trace_id": self.trace_id,
            "start_time": self.start_time,
            "end_time": self.end_time,
            "duration_ms": self.duration_ms,
            "span_count": len(self.spans),
            "is_complete": self.is_complete,
            "spans": [span.to_dict() for span in self.spans],
        }


class Tracer:
    """
    Distributed tracer for HybridPipe.
    
    Manages the creation and lifecycle of traces and spans,
    providing visibility into message processing flows.
    """
    
    def __init__(self, service_name: str = "hybridpipe") -> None:
        """
        Initialize the tracer.
        
        Args:
            service_name: Name of the service for tracing
        """
        self.service_name = service_name
        self._traces: Dict[str, Trace] = {}
        self._active_spans: Dict[str, Span] = {}
        self._lock = threading.RLock()
        self._enabled = True
    
    def enable(self) -> None:
        """Enable tracing."""
        self._enabled = True
    
    def disable(self) -> None:
        """Disable tracing."""
        self._enabled = False
    
    def is_enabled(self) -> bool:
        """Check if tracing is enabled."""
        return self._enabled
    
    def start_trace(self, operation_name: str, trace_id: Optional[str] = None) -> Span:
        """
        Start a new trace with a root span.
        
        Args:
            operation_name: Name of the root operation
            trace_id: Optional trace ID (generated if not provided)
        
        Returns:
            Root span of the new trace
        """
        if not self._enabled:
            return self._create_noop_span()
        
        trace_id = trace_id or self._generate_id()
        span = self.start_span(operation_name, trace_id=trace_id)
        
        with self._lock:
            if trace_id not in self._traces:
                self._traces[trace_id] = Trace(trace_id=trace_id)
            self._traces[trace_id].add_span(span)
        
        return span
    
    def start_span(
        self,
        operation_name: str,
        parent_span: Optional[Span] = None,
        trace_id: Optional[str] = None,
        tags: Optional[Dict[str, Any]] = None,
    ) -> Span:
        """
        Start a new span.
        
        Args:
            operation_name: Name of the operation
            parent_span: Parent span (if any)
            trace_id: Trace ID (required if no parent span)
            tags: Initial tags for the span
        
        Returns:
            New span instance
        """
        if not self._enabled:
            return self._create_noop_span()
        
        if parent_span:
            trace_id = parent_span.trace_id
            parent_span_id = parent_span.span_id
        else:
            if trace_id is None:
                raise ValueError("trace_id is required when no parent span is provided")
            parent_span_id = None
        
        span = Span(
            span_id=self._generate_id(),
            trace_id=trace_id,
            parent_span_id=parent_span_id,
            operation_name=operation_name,
            start_time=time.time(),
            tags=tags or {},
        )
        
        # Add service tag
        span.set_tag("service.name", self.service_name)
        
        # Copy baggage from parent
        if parent_span:
            span.baggage.update(parent_span.baggage)
        
        with self._lock:
            self._active_spans[span.span_id] = span
            
            # Add to trace
            if trace_id not in self._traces:
                self._traces[trace_id] = Trace(trace_id=trace_id)
            self._traces[trace_id].add_span(span)
        
        return span
    
    def finish_span(self, span: Span, status: str = TraceStatus.OK) -> None:
        """
        Finish a span.
        
        Args:
            span: Span to finish
            status: Final status of the span
        """
        if not self._enabled or span is None:
            return
        
        span.finish(status)
        
        with self._lock:
            self._active_spans.pop(span.span_id, None)
            
            # Update trace
            if span.trace_id in self._traces:
                trace = self._traces[span.trace_id]
                # Update trace end time if this is the last span
                if trace.is_complete:
                    trace.end_time = max(s.end_time for s in trace.spans if s.end_time)
    
    @contextmanager
    def span(
        self,
        operation_name: str,
        parent_span: Optional[Span] = None,
        trace_id: Optional[str] = None,
        tags: Optional[Dict[str, Any]] = None,
    ) -> ContextManager[Span]:
        """
        Context manager for automatic span lifecycle management.
        
        Args:
            operation_name: Name of the operation
            parent_span: Parent span (if any)
            trace_id: Trace ID (required if no parent span)
            tags: Initial tags for the span
        
        Yields:
            Span instance
        """
        span = self.start_span(operation_name, parent_span, trace_id, tags)
        try:
            yield span
        except Exception as e:
            span.set_tag("error", True)
            span.set_tag("error.type", type(e).__name__)
            span.set_tag("error.message", str(e))
            span.log("Exception occurred", error=str(e))
            self.finish_span(span, TraceStatus.ERROR)
            raise
        else:
            self.finish_span(span, TraceStatus.OK)
    
    def get_trace(self, trace_id: str) -> Optional[Trace]:
        """Get a trace by ID."""
        with self._lock:
            return self._traces.get(trace_id)
    
    def get_active_spans(self) -> List[Span]:
        """Get all currently active spans."""
        with self._lock:
            return list(self._active_spans.values())
    
    def get_traces(self, limit: Optional[int] = None) -> List[Trace]:
        """
        Get all traces.
        
        Args:
            limit: Maximum number of traces to return
        
        Returns:
            List of traces
        """
        with self._lock:
            traces = list(self._traces.values())
            if limit:
                traces = traces[-limit:]
            return traces
    
    def clear_traces(self) -> None:
        """Clear all completed traces."""
        with self._lock:
            # Keep only traces with active spans
            active_trace_ids = {span.trace_id for span in self._active_spans.values()}
            self._traces = {
                trace_id: trace
                for trace_id, trace in self._traces.items()
                if trace_id in active_trace_ids
            }
    
    def get_statistics(self) -> Dict[str, Any]:
        """Get tracing statistics."""
        with self._lock:
            total_traces = len(self._traces)
            active_spans = len(self._active_spans)
            completed_traces = sum(1 for trace in self._traces.values() if trace.is_complete)
            
            # Calculate average trace duration
            completed_durations = [
                trace.duration_ms
                for trace in self._traces.values()
                if trace.is_complete and trace.duration_ms is not None
            ]
            avg_duration = sum(completed_durations) / len(completed_durations) if completed_durations else 0
            
            return {
                "enabled": self._enabled,
                "service_name": self.service_name,
                "total_traces": total_traces,
                "completed_traces": completed_traces,
                "active_spans": active_spans,
                "average_trace_duration_ms": avg_duration,
            }
    
    def _generate_id(self) -> str:
        """Generate a unique ID."""
        return str(uuid.uuid4()).replace("-", "")
    
    def _create_noop_span(self) -> Span:
        """Create a no-op span when tracing is disabled."""
        return Span(
            span_id="noop",
            trace_id="noop",
            parent_span_id=None,
            operation_name="noop",
            start_time=time.time(),
        )


# Global tracer instance
_global_tracer: Optional[Tracer] = None
_tracer_lock = threading.Lock()


def get_tracer(service_name: str = "hybridpipe") -> Tracer:
    """Get the global tracer instance."""
    global _global_tracer
    if _global_tracer is None:
        with _tracer_lock:
            if _global_tracer is None:
                _global_tracer = Tracer(service_name)
    return _global_tracer
