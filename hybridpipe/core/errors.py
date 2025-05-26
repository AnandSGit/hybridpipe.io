"""
Exception hierarchy for HybridPipe.

This module defines all custom exceptions used throughout the HybridPipe
messaging system, providing detailed error information and proper error
handling capabilities.
"""

from typing import Any, Dict, Optional
from hybridpipe.core.types import BrokerType

__all__ = [
    "HybridPipeError",
    "ConnectionError", 
    "SerializationError",
    "ProtocolError",
    "ConfigurationError",
    "TimeoutError",
    "AuthenticationError",
    "AuthorizationError",
    "MessageTooLargeError",
    "PipeNotFoundError",
    "BrokerNotSupportedError",
    "MiddlewareError",
    "MonitoringError",
]


class HybridPipeError(Exception):
    """
    Base exception class for all HybridPipe errors.
    
    Attributes:
        message: Human-readable error message
        error_code: Machine-readable error code
        context: Additional context information
        broker_type: The broker type where the error occurred (if applicable)
        pipe: The pipe/channel where the error occurred (if applicable)
    """
    
    def __init__(
        self,
        message: str,
        error_code: Optional[str] = None,
        context: Optional[Dict[str, Any]] = None,
        broker_type: Optional[BrokerType] = None,
        pipe: Optional[str] = None,
        cause: Optional[Exception] = None,
    ) -> None:
        super().__init__(message)
        self.message = message
        self.error_code = error_code or self.__class__.__name__
        self.context = context or {}
        self.broker_type = broker_type
        self.pipe = pipe
        self.cause = cause
    
    def __str__(self) -> str:
        parts = [self.message]
        
        if self.broker_type:
            parts.append(f"broker={self.broker_type.name}")
        
        if self.pipe:
            parts.append(f"pipe={self.pipe}")
        
        if self.error_code and self.error_code != self.__class__.__name__:
            parts.append(f"code={self.error_code}")
        
        if self.context:
            context_str = ", ".join(f"{k}={v}" for k, v in self.context.items())
            parts.append(f"context=({context_str})")
        
        return " | ".join(parts)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert the exception to a dictionary for serialization."""
        return {
            "error_type": self.__class__.__name__,
            "message": self.message,
            "error_code": self.error_code,
            "context": self.context,
            "broker_type": self.broker_type.name if self.broker_type else None,
            "pipe": self.pipe,
            "cause": str(self.cause) if self.cause else None,
        }


class ConnectionError(HybridPipeError):
    """
    Raised when connection-related errors occur.
    
    This includes connection failures, timeouts, authentication issues,
    and unexpected disconnections.
    """
    pass


class SerializationError(HybridPipeError):
    """
    Raised when serialization or deserialization fails.
    
    This includes format errors, encoding issues, and schema validation
    failures.
    """
    
    def __init__(
        self,
        message: str,
        serialization_format: Optional[str] = None,
        data_type: Optional[str] = None,
        **kwargs: Any,
    ) -> None:
        context = kwargs.get("context", {})
        if serialization_format:
            context["serialization_format"] = serialization_format
        if data_type:
            context["data_type"] = data_type
        kwargs["context"] = context
        super().__init__(message, **kwargs)


class ProtocolError(HybridPipeError):
    """
    Raised when protocol-specific errors occur.
    
    This includes protocol violations, unsupported operations,
    and broker-specific errors.
    """
    pass


class ConfigurationError(HybridPipeError):
    """
    Raised when configuration errors occur.
    
    This includes missing required configuration, invalid values,
    and incompatible settings.
    """
    
    def __init__(
        self,
        message: str,
        config_key: Optional[str] = None,
        config_value: Optional[Any] = None,
        **kwargs: Any,
    ) -> None:
        context = kwargs.get("context", {})
        if config_key:
            context["config_key"] = config_key
        if config_value is not None:
            context["config_value"] = str(config_value)
        kwargs["context"] = context
        super().__init__(message, **kwargs)


class TimeoutError(HybridPipeError):
    """
    Raised when operations timeout.
    
    This includes connection timeouts, message send timeouts,
    and subscription timeouts.
    """
    
    def __init__(
        self,
        message: str,
        timeout_seconds: Optional[float] = None,
        operation: Optional[str] = None,
        **kwargs: Any,
    ) -> None:
        context = kwargs.get("context", {})
        if timeout_seconds:
            context["timeout_seconds"] = timeout_seconds
        if operation:
            context["operation"] = operation
        kwargs["context"] = context
        super().__init__(message, **kwargs)


class AuthenticationError(HybridPipeError):
    """
    Raised when authentication fails.
    
    This includes invalid credentials, expired tokens,
    and authentication protocol errors.
    """
    pass


class AuthorizationError(HybridPipeError):
    """
    Raised when authorization fails.
    
    This includes insufficient permissions, access denied,
    and authorization policy violations.
    """
    pass


class MessageTooLargeError(HybridPipeError):
    """
    Raised when a message exceeds size limits.
    
    This includes broker-specific size limits and protocol
    message size restrictions.
    """
    
    def __init__(
        self,
        message: str,
        message_size: Optional[int] = None,
        max_size: Optional[int] = None,
        **kwargs: Any,
    ) -> None:
        context = kwargs.get("context", {})
        if message_size:
            context["message_size"] = message_size
        if max_size:
            context["max_size"] = max_size
        kwargs["context"] = context
        super().__init__(message, **kwargs)


class PipeNotFoundError(HybridPipeError):
    """
    Raised when a pipe/channel/topic is not found.
    
    This includes non-existent topics, deleted channels,
    and access to unavailable pipes.
    """
    pass


class BrokerNotSupportedError(HybridPipeError):
    """
    Raised when an unsupported broker type is requested.
    
    This includes unregistered broker types and disabled
    protocol implementations.
    """
    
    def __init__(
        self,
        message: str,
        requested_broker: Optional[BrokerType] = None,
        available_brokers: Optional[list[BrokerType]] = None,
        **kwargs: Any,
    ) -> None:
        context = kwargs.get("context", {})
        if requested_broker:
            context["requested_broker"] = requested_broker.name
        if available_brokers:
            context["available_brokers"] = [b.name for b in available_brokers]
        kwargs["context"] = context
        super().__init__(message, **kwargs)


class MiddlewareError(HybridPipeError):
    """
    Raised when middleware processing fails.
    
    This includes middleware execution errors, configuration
    issues, and middleware chain failures.
    """
    
    def __init__(
        self,
        message: str,
        middleware_name: Optional[str] = None,
        middleware_stage: Optional[str] = None,
        **kwargs: Any,
    ) -> None:
        context = kwargs.get("context", {})
        if middleware_name:
            context["middleware_name"] = middleware_name
        if middleware_stage:
            context["middleware_stage"] = middleware_stage
        kwargs["context"] = context
        super().__init__(message, **kwargs)


class MonitoringError(HybridPipeError):
    """
    Raised when monitoring or metrics collection fails.
    
    This includes metrics collection errors, tracing failures,
    and monitoring system unavailability.
    """
    pass
