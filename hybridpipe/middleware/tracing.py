"""
Distributed tracing middleware for HybridPipe.

This module provides middleware for distributed tracing using OpenTelemetry,
allowing for end-to-end visibility of message flows across services.
"""

import time
import uuid
from typing import List, Optional, Dict, Any, Union
from contextlib import contextmanager

from hybridpipe.middleware.base import Middleware, MiddlewareContext, MiddlewareStage
from hybridpipe.core.types import BrokerType

__all__ = ["TracingMiddleware", "TraceContext", "get_tracer"]


class TraceContext:
    """
    Context for distributed tracing.
    
    Maintains trace and span information for correlating
    messages across service boundaries.
    """
    
    def __init__(
        self,
        trace_id: Optional[str] = None,
        span_id: Optional[str] = None,
        parent_span_id: Optional[str] = None,
        baggage: Optional[Dict[str, str]] = None,
    ) -> None:
        """
        Initialize trace context.
        
        Args:
            trace_id: Unique trace identifier
            span_id: Current span identifier
            parent_span_id: Parent span identifier
            baggage: Additional trace baggage
        """
        self.trace_id = trace_id or self._generate_id()
        self.span_id = span_id or self._generate_id()
        self.parent_span_id = parent_span_id
        self.baggage = baggage or {}
        self.start_time = time.time()
        self.end_time: Optional[float] = None
        self.status = "OK"
        self.error: Optional[str] = None
    
    @staticmethod
    def _generate_id() -> str:
        """Generate a unique identifier."""
        return str(uuid.uuid4()).replace("-", "")
    
    def create_child_span(self, operation_name: str) -> "TraceContext":
        """
        Create a child span context.
        
        Args:
            operation_name: Name of the operation for the child span
        
        Returns:
            New TraceContext for the child span
        """
        return TraceContext(
            trace_id=self.trace_id,
            span_id=self._generate_id(),
            parent_span_id=self.span_id,
            baggage=self.baggage.copy(),
        )
    
    def finish(self, status: str = "OK", error: Optional[str] = None) -> None:
        """
        Finish the span.
        
        Args:
            status: Final status of the span
            error: Error message if the span failed
        """
        self.end_time = time.time()
        self.status = status
        self.error = error
    
    def duration_ms(self) -> float:
        """Get the duration of the span in milliseconds."""
        end_time = self.end_time or time.time()
        return (end_time - self.start_time) * 1000
    
    def to_headers(self) -> Dict[str, str]:
        """Convert trace context to headers for propagation."""
        headers = {
            "x-trace-id": self.trace_id,
            "x-span-id": self.span_id,
        }
        
        if self.parent_span_id:
            headers["x-parent-span-id"] = self.parent_span_id
        
        # Add baggage
        for key, value in self.baggage.items():
            headers[f"x-baggage-{key}"] = value
        
        return headers
    
    @classmethod
    def from_headers(cls, headers: Dict[str, str]) -> Optional["TraceContext"]:
        """
        Create trace context from headers.
        
        Args:
            headers: Headers containing trace information
        
        Returns:
            TraceContext if trace headers are present, None otherwise
        """
        trace_id = headers.get("x-trace-id")
        if not trace_id:
            return None
        
        span_id = headers.get("x-span-id")
        parent_span_id = headers.get("x-parent-span-id")
        
        # Extract baggage
        baggage = {}
        for key, value in headers.items():
            if key.startswith("x-baggage-"):
                baggage_key = key[10:]  # Remove "x-baggage-" prefix
                baggage[baggage_key] = value
        
        return cls(
            trace_id=trace_id,
            span_id=span_id,
            parent_span_id=parent_span_id,
            baggage=baggage,
        )


class SimpleTracer:
    """
    Simple tracer implementation for HybridPipe.
    
    Provides basic tracing functionality without external dependencies.
    Can be replaced with OpenTelemetry tracer for production use.
    """
    
    def __init__(self, service_name: str = "hybridpipe") -> None:
        """
        Initialize the tracer.
        
        Args:
            service_name: Name of the service for tracing
        """
        self.service_name = service_name
        self._spans: List[TraceContext] = []
        self._active_spans: Dict[str, TraceContext] = {}
    
    def start_span(
        self,
        operation_name: str,
        parent_context: Optional[TraceContext] = None,
        tags: Optional[Dict[str, Any]] = None,
    ) -> TraceContext:
        """
        Start a new span.
        
        Args:
            operation_name: Name of the operation
            parent_context: Parent trace context
            tags: Additional tags for the span
        
        Returns:
            New TraceContext for the span
        """
        if parent_context:
            context = parent_context.create_child_span(operation_name)
        else:
            context = TraceContext()
        
        # Add tags to baggage
        if tags:
            context.baggage.update({k: str(v) for k, v in tags.items()})
        
        context.baggage["operation"] = operation_name
        context.baggage["service"] = self.service_name
        
        self._active_spans[context.span_id] = context
        return context
    
    def finish_span(
        self,
        context: TraceContext,
        status: str = "OK",
        error: Optional[str] = None,
    ) -> None:
        """
        Finish a span.
        
        Args:
            context: The trace context to finish
            status: Final status of the span
            error: Error message if the span failed
        """
        context.finish(status, error)
        self._spans.append(context)
        self._active_spans.pop(context.span_id, None)
    
    @contextmanager
    def span(
        self,
        operation_name: str,
        parent_context: Optional[TraceContext] = None,
        tags: Optional[Dict[str, Any]] = None,
    ):
        """
        Context manager for automatic span lifecycle management.
        
        Args:
            operation_name: Name of the operation
            parent_context: Parent trace context
            tags: Additional tags for the span
        
        Yields:
            TraceContext for the span
        """
        context = self.start_span(operation_name, parent_context, tags)
        try:
            yield context
        except Exception as e:
            self.finish_span(context, "ERROR", str(e))
            raise
        else:
            self.finish_span(context, "OK")
    
    def get_spans(self, trace_id: Optional[str] = None) -> List[TraceContext]:
        """
        Get completed spans.
        
        Args:
            trace_id: Optional trace ID to filter by
        
        Returns:
            List of completed spans
        """
        if trace_id:
            return [span for span in self._spans if span.trace_id == trace_id]
        return self._spans.copy()
    
    def clear_spans(self) -> None:
        """Clear all completed spans."""
        self._spans.clear()


# Global tracer instance
_global_tracer: Optional[SimpleTracer] = None


def get_tracer(service_name: str = "hybridpipe") -> SimpleTracer:
    """Get the global tracer instance."""
    global _global_tracer
    if _global_tracer is None:
        _global_tracer = SimpleTracer(service_name)
    return _global_tracer


class TracingMiddleware(Middleware):
    """
    Middleware for distributed tracing of message processing.
    
    Automatically creates spans for message operations and
    propagates trace context across service boundaries.
    """
    
    def __init__(
        self,
        name: str = "TracingMiddleware",
        tracer: Optional[SimpleTracer] = None,
        service_name: str = "hybridpipe",
        sample_rate: float = 1.0,
    ) -> None:
        """
        Initialize tracing middleware.
        
        Args:
            name: Name of the middleware
            tracer: Custom tracer instance
            service_name: Name of the service for tracing
            sample_rate: Sampling rate (0.0 to 1.0)
        """
        super().__init__(name)
        self.tracer = tracer or get_tracer(service_name)
        self.service_name = service_name
        self.sample_rate = sample_rate
    
    @property
    def supported_stages(self) -> List[MiddlewareStage]:
        """Support key stages for tracing."""
        return [
            MiddlewareStage.PRE_SEND,
            MiddlewareStage.POST_SEND,
            MiddlewareStage.PRE_RECEIVE,
            MiddlewareStage.POST_RECEIVE,
        ]
    
    def _should_trace(self) -> bool:
        """Determine if this operation should be traced based on sample rate."""
        import random
        return random.random() < self.sample_rate
    
    def _get_operation_name(self, context: MiddlewareContext) -> str:
        """Generate operation name for the span."""
        operation = context.stage.value.replace("_", " ").title()
        return f"{operation} {context.broker_type.name}"
    
    async def process(self, context: MiddlewareContext) -> MiddlewareContext:
        """Process tracing for the middleware context."""
        if not self._should_trace():
            return context
        
        operation_name = self._get_operation_name(context)
        
        # Extract or create trace context
        trace_context = None
        if context.headers:
            trace_context = TraceContext.from_headers(context.headers)
        
        # Start span for "pre" stages
        if context.stage.value.startswith("pre_"):
            span_context = self.tracer.start_span(
                operation_name=operation_name,
                parent_context=trace_context,
                tags={
                    "pipe": context.pipe,
                    "broker_type": context.broker_type.name,
                    "message_id": context.metadata.message_id,
                    "stage": context.stage.value,
                },
            )
            
            # Store span context for later use
            context.set_custom("trace_context", span_context)
            
            # Add trace headers for propagation
            trace_headers = span_context.to_headers()
            for key, value in trace_headers.items():
                context.set_header(key, value)
        
        # Finish span for "post" stages
        elif context.stage.value.startswith("post_"):
            span_context = context.get_custom("trace_context")
            if span_context:
                self.tracer.finish_span(span_context, "OK")
        
        return context
    
    async def on_error(self, context: MiddlewareContext, error: Exception) -> None:
        """Handle tracing for errors."""
        span_context = context.get_custom("trace_context")
        if span_context:
            self.tracer.finish_span(
                span_context,
                status="ERROR",
                error=f"{type(error).__name__}: {str(error)}",
            )
    
    def get_trace_summary(self, trace_id: str) -> Dict[str, Any]:
        """
        Get a summary of a trace.
        
        Args:
            trace_id: The trace ID to summarize
        
        Returns:
            Dictionary containing trace summary
        """
        spans = self.tracer.get_spans(trace_id)
        if not spans:
            return {}
        
        total_duration = sum(span.duration_ms() for span in spans)
        error_count = sum(1 for span in spans if span.status == "ERROR")
        
        return {
            "trace_id": trace_id,
            "span_count": len(spans),
            "total_duration_ms": total_duration,
            "error_count": error_count,
            "success_rate": (len(spans) - error_count) / len(spans) if spans else 0,
            "spans": [
                {
                    "span_id": span.span_id,
                    "parent_span_id": span.parent_span_id,
                    "operation": span.baggage.get("operation", "unknown"),
                    "duration_ms": span.duration_ms(),
                    "status": span.status,
                    "error": span.error,
                }
                for span in spans
            ],
        }
