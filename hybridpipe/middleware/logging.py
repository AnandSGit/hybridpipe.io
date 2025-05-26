"""
Logging middleware for HybridPipe.

This module provides middleware for logging message flows, errors,
and performance metrics throughout the messaging pipeline.
"""

import time
from typing import List, Optional, Dict, Any
import structlog

from hybridpipe.middleware.base import Middleware, MiddlewareContext, MiddlewareStage
from hybridpipe.core.errors import MiddlewareError

__all__ = ["LoggingMiddleware", "PerformanceLoggingMiddleware"]


class LoggingMiddleware(Middleware):
    """
    Middleware for logging message processing events.
    
    Logs message dispatch, receipt, and processing events with
    configurable detail levels and filtering options.
    """
    
    def __init__(
        self,
        name: str = "LoggingMiddleware",
        log_level: str = "INFO",
        log_data: bool = False,
        log_headers: bool = True,
        max_data_length: int = 1000,
        exclude_pipes: Optional[List[str]] = None,
        include_pipes: Optional[List[str]] = None,
    ) -> None:
        """
        Initialize logging middleware.
        
        Args:
            name: Name of the middleware
            log_level: Logging level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
            log_data: Whether to log message data
            log_headers: Whether to log message headers
            max_data_length: Maximum length of data to log
            exclude_pipes: Pipes to exclude from logging
            include_pipes: Pipes to include in logging (if set, only these are logged)
        """
        super().__init__(name)
        self.logger = structlog.get_logger(name)
        self.log_level = log_level.upper()
        self.log_data = log_data
        self.log_headers = log_headers
        self.max_data_length = max_data_length
        self.exclude_pipes = set(exclude_pipes or [])
        self.include_pipes = set(include_pipes or []) if include_pipes else None
    
    @property
    def supported_stages(self) -> List[MiddlewareStage]:
        """Support all stages except error (handled separately)."""
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
    
    def _should_log_pipe(self, pipe: str) -> bool:
        """Check if we should log events for this pipe."""
        if self.include_pipes and pipe not in self.include_pipes:
            return False
        if pipe in self.exclude_pipes:
            return False
        return True
    
    def _format_data(self, data: Any) -> str:
        """Format data for logging."""
        if not self.log_data or data is None:
            return "<data not logged>"
        
        data_str = str(data)
        if len(data_str) > self.max_data_length:
            data_str = data_str[:self.max_data_length] + "..."
        
        return data_str
    
    def _get_log_method(self):
        """Get the appropriate log method based on log level."""
        level_map = {
            "DEBUG": self.logger.debug,
            "INFO": self.logger.info,
            "WARNING": self.logger.warning,
            "ERROR": self.logger.error,
            "CRITICAL": self.logger.critical,
        }
        return level_map.get(self.log_level, self.logger.info)
    
    async def process(self, context: MiddlewareContext) -> MiddlewareContext:
        """Process and log the middleware context."""
        if not self._should_log_pipe(context.pipe):
            return context
        
        log_method = self._get_log_method()
        
        # Prepare log data
        log_data = {
            "stage": context.stage.value,
            "pipe": context.pipe,
            "broker_type": context.broker_type.name,
            "message_id": context.metadata.message_id,
            "timestamp": context.metadata.timestamp.isoformat(),
        }
        
        # Add headers if enabled
        if self.log_headers and context.headers:
            log_data["headers"] = context.headers
        
        # Add data if enabled
        if self.log_data:
            log_data["data"] = self._format_data(context.data)
        
        # Add custom context data
        if context.custom:
            log_data["custom"] = context.custom
        
        # Log the event
        log_method(
            f"Message processing: {context.stage.value}",
            **log_data
        )
        
        return context
    
    async def on_error(self, context: MiddlewareContext, error: Exception) -> None:
        """Log errors that occur during processing."""
        if not self._should_log_pipe(context.pipe):
            return
        
        self.logger.error(
            "Error in message processing",
            stage=context.stage.value,
            pipe=context.pipe,
            broker_type=context.broker_type.name,
            message_id=context.metadata.message_id,
            error_type=type(error).__name__,
            error_message=str(error),
            headers=context.headers if self.log_headers else None,
            data=self._format_data(context.data) if self.log_data else None,
        )


class PerformanceLoggingMiddleware(Middleware):
    """
    Middleware for logging performance metrics.
    
    Tracks timing information for message processing stages
    and logs performance statistics.
    """
    
    def __init__(
        self,
        name: str = "PerformanceLoggingMiddleware",
        log_threshold_ms: float = 100.0,
        track_all_stages: bool = False,
    ) -> None:
        """
        Initialize performance logging middleware.
        
        Args:
            name: Name of the middleware
            log_threshold_ms: Only log operations slower than this threshold
            track_all_stages: Whether to track timing for all stages
        """
        super().__init__(name)
        self.logger = structlog.get_logger(name)
        self.log_threshold_ms = log_threshold_ms
        self.track_all_stages = track_all_stages
        self._timings: Dict[str, float] = {}
    
    @property
    def supported_stages(self) -> List[MiddlewareStage]:
        """Support timing-relevant stages."""
        if self.track_all_stages:
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
        else:
            return [
                MiddlewareStage.PRE_SEND,
                MiddlewareStage.POST_SEND,
                MiddlewareStage.PRE_RECEIVE,
                MiddlewareStage.POST_RECEIVE,
            ]
    
    def _get_timing_key(self, context: MiddlewareContext) -> str:
        """Generate a unique key for timing tracking."""
        return f"{context.metadata.message_id}:{context.stage.value}"
    
    async def process(self, context: MiddlewareContext) -> MiddlewareContext:
        """Track timing for message processing."""
        timing_key = self._get_timing_key(context)
        
        # Record start time for "pre" stages
        if context.stage.value.startswith("pre_"):
            context.set_custom(f"start_time_{context.stage.value}", time.time())
        
        # Calculate and log duration for "post" stages
        elif context.stage.value.startswith("post_"):
            pre_stage = context.stage.value.replace("post_", "pre_")
            start_time = context.get_custom(f"start_time_{pre_stage}")
            
            if start_time:
                duration_ms = (time.time() - start_time) * 1000
                
                if duration_ms >= self.log_threshold_ms:
                    operation = pre_stage.replace("pre_", "")
                    
                    self.logger.info(
                        f"Performance: {operation} operation",
                        operation=operation,
                        duration_ms=round(duration_ms, 2),
                        pipe=context.pipe,
                        broker_type=context.broker_type.name,
                        message_id=context.metadata.message_id,
                        threshold_exceeded=True,
                    )
        
        return context
    
    async def on_error(self, context: MiddlewareContext, error: Exception) -> None:
        """Log performance data when errors occur."""
        # Calculate duration if we have start time
        for stage in ["serialize", "send", "receive", "deserialize"]:
            start_time = context.get_custom(f"start_time_pre_{stage}")
            if start_time:
                duration_ms = (time.time() - start_time) * 1000
                
                self.logger.warning(
                    f"Performance: {stage} operation failed",
                    operation=stage,
                    duration_ms=round(duration_ms, 2),
                    pipe=context.pipe,
                    broker_type=context.broker_type.name,
                    message_id=context.metadata.message_id,
                    error_type=type(error).__name__,
                    error_message=str(error),
                )
