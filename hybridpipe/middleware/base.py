"""
Base middleware classes and middleware stack implementation.

This module provides the foundation for the HybridPipe middleware system,
including the abstract base class for middleware and the middleware stack
that manages the execution order and flow.
"""

from abc import ABC, abstractmethod
from typing import Any, Dict, List, Optional, Union, Callable, Awaitable
import asyncio
from enum import Enum

from hybridpipe.core.types import MessageMetadata, BrokerType
from hybridpipe.core.errors import MiddlewareError

__all__ = ["Middleware", "MiddlewareStack", "MiddlewareStage", "MiddlewareContext"]


class MiddlewareStage(Enum):
    """Stages where middleware can be executed."""
    PRE_SERIALIZE = "pre_serialize"
    POST_SERIALIZE = "post_serialize"
    PRE_SEND = "pre_send"
    POST_SEND = "post_send"
    PRE_RECEIVE = "pre_receive"
    POST_RECEIVE = "post_receive"
    PRE_DESERIALIZE = "pre_deserialize"
    POST_DESERIALIZE = "post_deserialize"
    ERROR = "error"


class MiddlewareContext:
    """
    Context object passed to middleware containing request/response information.
    
    Attributes:
        pipe: The pipe/channel/topic name
        broker_type: The type of broker handling the message
        metadata: Message metadata
        stage: Current middleware stage
        data: The message data (may be modified by middleware)
        headers: Message headers (may be modified by middleware)
        error: Any error that occurred (for error stage)
        custom: Custom data that can be set by middleware
    """
    
    def __init__(
        self,
        pipe: str,
        broker_type: BrokerType,
        metadata: MessageMetadata,
        stage: MiddlewareStage,
        data: Any = None,
        headers: Optional[Dict[str, Any]] = None,
        error: Optional[Exception] = None,
    ) -> None:
        self.pipe = pipe
        self.broker_type = broker_type
        self.metadata = metadata
        self.stage = stage
        self.data = data
        self.headers = headers or {}
        self.error = error
        self.custom: Dict[str, Any] = {}
    
    def set_data(self, data: Any) -> None:
        """Set the message data."""
        self.data = data
    
    def get_data(self) -> Any:
        """Get the message data."""
        return self.data
    
    def set_header(self, key: str, value: Any) -> None:
        """Set a message header."""
        self.headers[key] = value
    
    def get_header(self, key: str, default: Any = None) -> Any:
        """Get a message header."""
        return self.headers.get(key, default)
    
    def set_custom(self, key: str, value: Any) -> None:
        """Set custom data."""
        self.custom[key] = value
    
    def get_custom(self, key: str, default: Any = None) -> Any:
        """Get custom data."""
        return self.custom.get(key, default)


class Middleware(ABC):
    """
    Abstract base class for middleware implementations.
    
    Middleware can intercept and modify messages at various stages
    of the messaging pipeline, enabling cross-cutting concerns like
    logging, monitoring, authentication, and data transformation.
    """
    
    def __init__(self, name: Optional[str] = None) -> None:
        """
        Initialize the middleware.
        
        Args:
            name: Optional name for the middleware (defaults to class name)
        """
        self.name = name or self.__class__.__name__
        self.enabled = True
    
    @property
    @abstractmethod
    def supported_stages(self) -> List[MiddlewareStage]:
        """Get the list of stages this middleware supports."""
        pass
    
    @abstractmethod
    async def process(self, context: MiddlewareContext) -> MiddlewareContext:
        """
        Process the middleware context.
        
        Args:
            context: The middleware context containing message data
        
        Returns:
            Modified context (can be the same instance)
        
        Raises:
            MiddlewareError: If middleware processing fails
        """
        pass
    
    async def on_error(self, context: MiddlewareContext, error: Exception) -> None:
        """
        Handle errors that occur during middleware processing.
        
        Args:
            context: The middleware context
            error: The error that occurred
        """
        pass
    
    def enable(self) -> None:
        """Enable this middleware."""
        self.enabled = True
    
    def disable(self) -> None:
        """Disable this middleware."""
        self.enabled = False
    
    def is_enabled(self) -> bool:
        """Check if this middleware is enabled."""
        return self.enabled
    
    def supports_stage(self, stage: MiddlewareStage) -> bool:
        """Check if this middleware supports the given stage."""
        return stage in self.supported_stages


class MiddlewareStack:
    """
    Manages a stack of middleware and executes them in order.
    
    The middleware stack maintains separate lists for each stage
    and executes middleware in the order they were added.
    """
    
    def __init__(self) -> None:
        """Initialize an empty middleware stack."""
        self._middleware: Dict[MiddlewareStage, List[Middleware]] = {
            stage: [] for stage in MiddlewareStage
        }
        self._global_middleware: List[Middleware] = []
    
    def add_middleware(
        self,
        middleware: Middleware,
        stages: Optional[List[MiddlewareStage]] = None,
    ) -> None:
        """
        Add middleware to the stack.
        
        Args:
            middleware: The middleware to add
            stages: Specific stages to add the middleware to (defaults to all supported)
        """
        if stages is None:
            stages = middleware.supported_stages
        
        for stage in stages:
            if middleware.supports_stage(stage):
                self._middleware[stage].append(middleware)
        
        # Keep track of all middleware for management
        if middleware not in self._global_middleware:
            self._global_middleware.append(middleware)
    
    def remove_middleware(self, middleware: Middleware) -> None:
        """
        Remove middleware from all stages.
        
        Args:
            middleware: The middleware to remove
        """
        for stage_middleware in self._middleware.values():
            while middleware in stage_middleware:
                stage_middleware.remove(middleware)
        
        if middleware in self._global_middleware:
            self._global_middleware.remove(middleware)
    
    def get_middleware_for_stage(self, stage: MiddlewareStage) -> List[Middleware]:
        """
        Get all middleware for a specific stage.
        
        Args:
            stage: The middleware stage
        
        Returns:
            List of middleware for the stage
        """
        return [mw for mw in self._middleware[stage] if mw.is_enabled()]
    
    def get_all_middleware(self) -> List[Middleware]:
        """Get all middleware in the stack."""
        return self._global_middleware.copy()
    
    def clear(self) -> None:
        """Remove all middleware from the stack."""
        for stage_middleware in self._middleware.values():
            stage_middleware.clear()
        self._global_middleware.clear()
    
    async def execute_stage(
        self,
        stage: MiddlewareStage,
        context: MiddlewareContext,
    ) -> MiddlewareContext:
        """
        Execute all middleware for a specific stage.
        
        Args:
            stage: The middleware stage to execute
            context: The middleware context
        
        Returns:
            Modified context after all middleware execution
        
        Raises:
            MiddlewareError: If any middleware fails
        """
        middleware_list = self.get_middleware_for_stage(stage)
        
        for middleware in middleware_list:
            try:
                context = await middleware.process(context)
                if context is None:
                    raise MiddlewareError(
                        f"Middleware {middleware.name} returned None context",
                        middleware_name=middleware.name,
                        middleware_stage=stage.value,
                    )
            except Exception as e:
                # Create error context for error handling
                error_context = MiddlewareContext(
                    pipe=context.pipe,
                    broker_type=context.broker_type,
                    metadata=context.metadata,
                    stage=MiddlewareStage.ERROR,
                    data=context.data,
                    headers=context.headers,
                    error=e,
                )
                
                # Call error handler
                try:
                    await middleware.on_error(error_context, e)
                except Exception:
                    pass  # Ignore errors in error handlers
                
                # Re-raise the original error
                if isinstance(e, MiddlewareError):
                    raise
                else:
                    raise MiddlewareError(
                        f"Middleware {middleware.name} failed: {str(e)}",
                        middleware_name=middleware.name,
                        middleware_stage=stage.value,
                        cause=e,
                    )
        
        return context
    
    def __len__(self) -> int:
        """Get the total number of middleware instances."""
        return len(self._global_middleware)
    
    def __contains__(self, middleware: Middleware) -> bool:
        """Check if middleware is in the stack."""
        return middleware in self._global_middleware
    
    def __iter__(self):
        """Iterate over all middleware."""
        return iter(self._global_middleware)


class ConditionalMiddleware(Middleware):
    """
    Wrapper middleware that conditionally executes based on a predicate.
    
    This allows for dynamic enabling/disabling of middleware based on
    runtime conditions like message content, headers, or context.
    """
    
    def __init__(
        self,
        wrapped_middleware: Middleware,
        condition: Callable[[MiddlewareContext], Union[bool, Awaitable[bool]]],
        name: Optional[str] = None,
    ) -> None:
        """
        Initialize conditional middleware.
        
        Args:
            wrapped_middleware: The middleware to wrap
            condition: Function that returns True if middleware should execute
            name: Optional name for the middleware
        """
        super().__init__(name or f"Conditional({wrapped_middleware.name})")
        self.wrapped_middleware = wrapped_middleware
        self.condition = condition
    
    @property
    def supported_stages(self) -> List[MiddlewareStage]:
        """Get supported stages from wrapped middleware."""
        return self.wrapped_middleware.supported_stages
    
    async def process(self, context: MiddlewareContext) -> MiddlewareContext:
        """Process context conditionally."""
        # Evaluate condition
        should_execute = self.condition(context)
        if asyncio.iscoroutine(should_execute):
            should_execute = await should_execute
        
        # Execute wrapped middleware if condition is met
        if should_execute:
            return await self.wrapped_middleware.process(context)
        else:
            return context
    
    async def on_error(self, context: MiddlewareContext, error: Exception) -> None:
        """Delegate error handling to wrapped middleware."""
        await self.wrapped_middleware.on_error(context, error)
