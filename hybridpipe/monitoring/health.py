"""
Health monitoring for HybridPipe.

This module provides health checking capabilities for monitoring
the status of messaging connections and system components.
"""

import time
import asyncio
from typing import Dict, List, Optional, Any, Callable, Awaitable
from dataclasses import dataclass
from enum import Enum
from datetime import datetime, timedelta

from hybridpipe.core.types import BrokerType
from hybridpipe.core.interface import HybridPipe

__all__ = ["HealthStatus", "HealthCheck", "HealthChecker"]


class HealthStatus(Enum):
    """Health status enumeration."""
    HEALTHY = "healthy"
    DEGRADED = "degraded"
    UNHEALTHY = "unhealthy"
    UNKNOWN = "unknown"


@dataclass
class HealthCheck:
    """
    Represents a health check result.
    
    Attributes:
        name: Name of the health check
        status: Current health status
        message: Human-readable status message
        timestamp: When the check was performed
        duration_ms: How long the check took
        metadata: Additional metadata about the check
    """
    name: str
    status: HealthStatus
    message: str
    timestamp: datetime
    duration_ms: float
    metadata: Dict[str, Any]


class HealthChecker:
    """
    Health checker for HybridPipe components.
    
    Monitors the health of messaging connections, brokers,
    and other system components.
    """
    
    def __init__(self, check_interval: float = 30.0) -> None:
        """
        Initialize health checker.
        
        Args:
            check_interval: Interval between health checks in seconds
        """
        self.check_interval = check_interval
        self._checks: Dict[str, Callable[[], Awaitable[HealthCheck]]] = {}
        self._results: Dict[str, HealthCheck] = {}
        self._running = False
        self._task: Optional[asyncio.Task] = None
    
    def register_check(
        self,
        name: str,
        check_func: Callable[[], Awaitable[HealthCheck]],
    ) -> None:
        """
        Register a health check function.
        
        Args:
            name: Name of the health check
            check_func: Async function that performs the check
        """
        self._checks[name] = check_func
    
    def register_router_check(self, name: str, router: HybridPipe) -> None:
        """
        Register a health check for a HybridPipe router.
        
        Args:
            name: Name of the health check
            router: HybridPipe router to check
        """
        async def check_router() -> HealthCheck:
            start_time = time.time()
            
            try:
                # Check if router is connected
                if not router.is_connected:
                    return HealthCheck(
                        name=name,
                        status=HealthStatus.UNHEALTHY,
                        message="Router is not connected",
                        timestamp=datetime.now(),
                        duration_ms=(time.time() - start_time) * 1000,
                        metadata={
                            "broker_type": router.broker_type.name,
                            "connection_state": router.connection_state.value,
                        },
                    )
                
                # Perform health check
                health_info = await router.health_check()
                
                # Determine status based on health info
                if health_info.get("is_connected", False):
                    status = HealthStatus.HEALTHY
                    message = "Router is healthy"
                else:
                    status = HealthStatus.DEGRADED
                    message = "Router connection issues detected"
                
                return HealthCheck(
                    name=name,
                    status=status,
                    message=message,
                    timestamp=datetime.now(),
                    duration_ms=(time.time() - start_time) * 1000,
                    metadata=health_info,
                )
                
            except Exception as e:
                return HealthCheck(
                    name=name,
                    status=HealthStatus.UNHEALTHY,
                    message=f"Health check failed: {str(e)}",
                    timestamp=datetime.now(),
                    duration_ms=(time.time() - start_time) * 1000,
                    metadata={
                        "error": str(e),
                        "error_type": type(e).__name__,
                    },
                )
        
        self.register_check(name, check_router)
    
    async def run_check(self, name: str) -> HealthCheck:
        """
        Run a specific health check.
        
        Args:
            name: Name of the health check to run
        
        Returns:
            Health check result
        
        Raises:
            KeyError: If health check is not registered
        """
        if name not in self._checks:
            raise KeyError(f"Health check '{name}' is not registered")
        
        try:
            result = await self._checks[name]()
            self._results[name] = result
            return result
        except Exception as e:
            result = HealthCheck(
                name=name,
                status=HealthStatus.UNHEALTHY,
                message=f"Health check execution failed: {str(e)}",
                timestamp=datetime.now(),
                duration_ms=0,
                metadata={
                    "error": str(e),
                    "error_type": type(e).__name__,
                },
            )
            self._results[name] = result
            return result
    
    async def run_all_checks(self) -> Dict[str, HealthCheck]:
        """
        Run all registered health checks.
        
        Returns:
            Dictionary of health check results
        """
        results = {}
        
        # Run all checks concurrently
        tasks = [
            self.run_check(name)
            for name in self._checks.keys()
        ]
        
        if tasks:
            check_results = await asyncio.gather(*tasks, return_exceptions=True)
            
            for i, (name, result) in enumerate(zip(self._checks.keys(), check_results)):
                if isinstance(result, Exception):
                    results[name] = HealthCheck(
                        name=name,
                        status=HealthStatus.UNHEALTHY,
                        message=f"Health check failed: {str(result)}",
                        timestamp=datetime.now(),
                        duration_ms=0,
                        metadata={
                            "error": str(result),
                            "error_type": type(result).__name__,
                        },
                    )
                else:
                    results[name] = result
        
        self._results.update(results)
        return results
    
    def get_latest_results(self) -> Dict[str, HealthCheck]:
        """Get the latest health check results."""
        return self._results.copy()
    
    def get_overall_status(self) -> HealthStatus:
        """
        Get the overall system health status.
        
        Returns:
            Overall health status based on all checks
        """
        if not self._results:
            return HealthStatus.UNKNOWN
        
        statuses = [check.status for check in self._results.values()]
        
        # If any check is unhealthy, system is unhealthy
        if HealthStatus.UNHEALTHY in statuses:
            return HealthStatus.UNHEALTHY
        
        # If any check is degraded, system is degraded
        if HealthStatus.DEGRADED in statuses:
            return HealthStatus.DEGRADED
        
        # If all checks are healthy, system is healthy
        if all(status == HealthStatus.HEALTHY for status in statuses):
            return HealthStatus.HEALTHY
        
        return HealthStatus.UNKNOWN
    
    def get_health_summary(self) -> Dict[str, Any]:
        """
        Get a comprehensive health summary.
        
        Returns:
            Dictionary containing health summary
        """
        overall_status = self.get_overall_status()
        results = self.get_latest_results()
        
        # Count statuses
        status_counts = {status.value: 0 for status in HealthStatus}
        for check in results.values():
            status_counts[check.status.value] += 1
        
        # Calculate average check duration
        durations = [check.duration_ms for check in results.values()]
        avg_duration = sum(durations) / len(durations) if durations else 0
        
        return {
            "overall_status": overall_status.value,
            "check_count": len(results),
            "status_counts": status_counts,
            "average_check_duration_ms": avg_duration,
            "last_check_time": max(
                (check.timestamp for check in results.values()),
                default=None
            ),
            "checks": {
                name: {
                    "status": check.status.value,
                    "message": check.message,
                    "timestamp": check.timestamp.isoformat(),
                    "duration_ms": check.duration_ms,
                    "metadata": check.metadata,
                }
                for name, check in results.items()
            },
        }
    
    async def start_monitoring(self) -> None:
        """Start continuous health monitoring."""
        if self._running:
            return
        
        self._running = True
        self._task = asyncio.create_task(self._monitoring_loop())
    
    async def stop_monitoring(self) -> None:
        """Stop continuous health monitoring."""
        self._running = False
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
            self._task = None
    
    async def _monitoring_loop(self) -> None:
        """Main monitoring loop."""
        while self._running:
            try:
                await self.run_all_checks()
                await asyncio.sleep(self.check_interval)
            except asyncio.CancelledError:
                break
            except Exception as e:
                # Log error but continue monitoring
                import structlog
                logger = structlog.get_logger()
                logger.error(
                    "Error in health monitoring loop",
                    error=str(e),
                    error_type=type(e).__name__,
                )
                await asyncio.sleep(self.check_interval)
    
    def is_monitoring(self) -> bool:
        """Check if continuous monitoring is active."""
        return self._running and self._task is not None and not self._task.done()
    
    def remove_check(self, name: str) -> None:
        """Remove a health check."""
        self._checks.pop(name, None)
        self._results.pop(name, None)
    
    def clear_results(self) -> None:
        """Clear all health check results."""
        self._results.clear()
    
    def __len__(self) -> int:
        """Get the number of registered health checks."""
        return len(self._checks)
    
    def __contains__(self, name: str) -> bool:
        """Check if a health check is registered."""
        return name in self._checks
