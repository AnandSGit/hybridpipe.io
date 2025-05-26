"""
Configuration management for HybridPipe.

This module provides configuration classes and utilities for managing
protocol-specific settings, serialization options, and global configuration.
"""

from typing import Any, Dict, List, Optional, Union
from pathlib import Path
import os
import json
import toml
from pydantic import BaseModel, Field, field_validator, model_validator

from hybridpipe.core.types import BrokerType, SerializationFormat

__all__ = [
    "HybridPipeConfig",
    "ProtocolConfig",
    "SerializationConfig",
    "MonitoringConfig",
    "load_config",
    "save_config",
]


class SerializationConfig(BaseModel):
    """Configuration for serialization options."""

    default_format: SerializationFormat = SerializationFormat.JSON
    compression_enabled: bool = False
    compression_level: int = Field(default=6, ge=0, le=9)
    compression_threshold: int = Field(default=64, ge=0)
    cross_language_compatible: bool = True

    model_config = {"use_enum_values": True}


class MonitoringConfig(BaseModel):
    """Configuration for monitoring and metrics."""

    enabled: bool = True
    metrics_enabled: bool = True
    tracing_enabled: bool = True
    metrics_port: int = Field(default=8080, ge=1024, le=65535)
    metrics_path: str = "/metrics"
    trace_sample_rate: float = Field(default=0.1, ge=0.0, le=1.0)
    log_level: str = Field(default="INFO", pattern="^(DEBUG|INFO|WARNING|ERROR|CRITICAL)$")
    structured_logging: bool = True


class ProtocolConfig(BaseModel):
    """Base configuration for protocol implementations."""

    enabled: bool = True
    connection_timeout: float = Field(default=30.0, gt=0)
    reconnect_enabled: bool = True
    reconnect_delay: float = Field(default=1.0, gt=0)
    reconnect_max_attempts: int = Field(default=5, ge=0)
    reconnect_backoff_multiplier: float = Field(default=2.0, gt=1.0)
    reconnect_max_delay: float = Field(default=60.0, gt=0)
    health_check_interval: float = Field(default=30.0, gt=0)

    # Protocol-specific settings (override in subclasses)
    protocol_settings: Dict[str, Any] = Field(default_factory=dict)

    model_config = {"extra": "allow"}  # Allow additional fields for protocol-specific settings


class KafkaConfig(ProtocolConfig):
    """Configuration for Kafka protocol."""

    bootstrap_servers: List[str] = Field(default=["localhost:9092"])
    client_id: str = "hybridpipe-python"
    group_id: str = "hybridpipe-group"
    auto_offset_reset: str = Field(default="latest", pattern="^(earliest|latest)$")
    enable_auto_commit: bool = True
    auto_commit_interval_ms: int = Field(default=5000, gt=0)
    session_timeout_ms: int = Field(default=30000, gt=0)
    heartbeat_interval_ms: int = Field(default=3000, gt=0)
    max_poll_records: int = Field(default=500, gt=0)

    # Security settings
    security_protocol: str = Field(default="PLAINTEXT", pattern="^(PLAINTEXT|SSL|SASL_PLAINTEXT|SASL_SSL)$")
    ssl_ca_location: Optional[str] = None
    ssl_certificate_location: Optional[str] = None
    ssl_key_location: Optional[str] = None
    sasl_mechanism: Optional[str] = None
    sasl_username: Optional[str] = None
    sasl_password: Optional[str] = None


class RedisConfig(ProtocolConfig):
    """Configuration for Redis protocol."""

    host: str = "localhost"
    port: int = Field(default=6379, ge=1, le=65535)
    db: int = Field(default=0, ge=0)
    password: Optional[str] = None
    username: Optional[str] = None
    ssl: bool = False
    ssl_cert_reqs: str = "required"
    ssl_ca_certs: Optional[str] = None
    ssl_certfile: Optional[str] = None
    ssl_keyfile: Optional[str] = None
    max_connections: int = Field(default=10, gt=0)
    retry_on_timeout: bool = True
    socket_keepalive: bool = True
    socket_keepalive_options: Dict[str, int] = Field(default_factory=dict)


class RabbitMQConfig(ProtocolConfig):
    """Configuration for RabbitMQ protocol."""

    host: str = "localhost"
    port: int = Field(default=5672, ge=1, le=65535)
    virtual_host: str = "/"
    username: str = "guest"
    password: str = "guest"
    ssl: bool = False
    ssl_options: Dict[str, Any] = Field(default_factory=dict)
    heartbeat: int = Field(default=600, ge=0)
    blocked_connection_timeout: float = Field(default=300.0, gt=0)
    connection_attempts: int = Field(default=3, gt=0)
    retry_delay: float = Field(default=2.0, gt=0)


class MQTTConfig(ProtocolConfig):
    """Configuration for MQTT protocol."""

    host: str = "localhost"
    port: int = Field(default=1883, ge=1, le=65535)
    keepalive: int = Field(default=60, gt=0)
    username: Optional[str] = None
    password: Optional[str] = None
    client_id: Optional[str] = None
    clean_session: bool = True
    protocol: str = Field(default="MQTTv311", pattern="^(MQTTv31|MQTTv311|MQTTv5)$")
    transport: str = Field(default="tcp", pattern="^(tcp|websockets)$")

    # TLS settings
    tls_enabled: bool = False
    tls_ca_certs: Optional[str] = None
    tls_certfile: Optional[str] = None
    tls_keyfile: Optional[str] = None
    tls_insecure: bool = False

    # QoS settings
    default_qos: int = Field(default=0, ge=0, le=2)
    max_inflight_messages: int = Field(default=20, gt=0)
    max_queued_messages: int = Field(default=0, ge=0)


class NATSConfig(ProtocolConfig):
    """Configuration for NATS protocol."""

    servers: List[str] = Field(default=["nats://localhost:4222"])
    name: str = "hybridpipe-python"
    user: Optional[str] = None
    password: Optional[str] = None
    token: Optional[str] = None
    max_reconnect_attempts: int = Field(default=60, ge=-1)
    reconnect_time_wait: float = Field(default=2.0, gt=0)
    ping_interval: float = Field(default=120.0, gt=0)
    max_outstanding_pings: int = Field(default=2, gt=0)

    # TLS settings
    tls_enabled: bool = False
    tls_ca_file: Optional[str] = None
    tls_cert_file: Optional[str] = None
    tls_key_file: Optional[str] = None

    # JetStream settings
    jetstream_enabled: bool = False
    jetstream_domain: Optional[str] = None


class TCPConfig(ProtocolConfig):
    """Configuration for TCP protocol."""

    host: str = "localhost"
    port: int = Field(default=8080, ge=1, le=65535)
    buffer_size: int = Field(default=8192, gt=0)
    nodelay: bool = True
    keepalive: bool = True
    keepalive_idle: int = Field(default=1, gt=0)
    keepalive_interval: int = Field(default=3, gt=0)
    keepalive_count: int = Field(default=5, gt=0)

    # Server mode settings
    server_mode: bool = False
    max_connections: int = Field(default=100, gt=0)
    backlog: int = Field(default=128, gt=0)


class HybridPipeConfig(BaseModel):
    """Main configuration class for HybridPipe."""

    # Global settings
    default_broker: BrokerType = BrokerType.MOCK
    serialization: SerializationConfig = Field(default_factory=SerializationConfig)
    monitoring: MonitoringConfig = Field(default_factory=MonitoringConfig)

    # Protocol configurations
    kafka: KafkaConfig = Field(default_factory=KafkaConfig)
    redis: RedisConfig = Field(default_factory=RedisConfig)
    rabbitmq: RabbitMQConfig = Field(default_factory=RabbitMQConfig)
    mqtt: MQTTConfig = Field(default_factory=MQTTConfig)
    nats: NATSConfig = Field(default_factory=NATSConfig)
    tcp: TCPConfig = Field(default_factory=TCPConfig)

    # Environment variable overrides
    @model_validator(mode='before')
    @classmethod
    def env_override(cls, values):
        """Override configuration values with environment variables."""
        if isinstance(values, dict):
            # Override default_broker
            env_value = os.getenv("HYBRIDPIPE_DEFAULT_BROKER")
            if env_value is not None:
                try:
                    # Try to parse as integer for enum
                    broker_id = int(env_value)
                    values['default_broker'] = BrokerType(broker_id)
                except (ValueError, TypeError):
                    # Try to parse as string name
                    try:
                        values['default_broker'] = BrokerType[env_value.upper()]
                    except KeyError:
                        pass
        return values

    model_config = {"use_enum_values": True, "extra": "allow"}


def load_config(
    config_path: Optional[Union[str, Path]] = None,
    config_dict: Optional[Dict[str, Any]] = None,
) -> HybridPipeConfig:
    """
    Load configuration from file or dictionary.

    Args:
        config_path: Path to configuration file (JSON or TOML)
        config_dict: Configuration dictionary

    Returns:
        HybridPipeConfig instance

    Raises:
        FileNotFoundError: If config file doesn't exist
        ValueError: If config format is invalid
    """
    if config_dict:
        return HybridPipeConfig(**config_dict)

    if config_path:
        config_path = Path(config_path)
        if not config_path.exists():
            raise FileNotFoundError(f"Configuration file not found: {config_path}")

        if config_path.suffix.lower() == '.json':
            with open(config_path, 'r') as f:
                config_data = json.load(f)
        elif config_path.suffix.lower() in ['.toml', '.tml']:
            with open(config_path, 'r') as f:
                config_data = toml.load(f)
        else:
            raise ValueError(f"Unsupported config file format: {config_path.suffix}")

        return HybridPipeConfig(**config_data)

    # Load from environment variables or use defaults
    return HybridPipeConfig()


def save_config(
    config: HybridPipeConfig,
    config_path: Union[str, Path],
    format: str = "auto",
) -> None:
    """
    Save configuration to file.

    Args:
        config: Configuration to save
        config_path: Path to save configuration file
        format: File format ('json', 'toml', or 'auto' to detect from extension)

    Raises:
        ValueError: If format is unsupported
    """
    config_path = Path(config_path)

    if format == "auto":
        if config_path.suffix.lower() == '.json':
            format = "json"
        elif config_path.suffix.lower() in ['.toml', '.tml']:
            format = "toml"
        else:
            format = "json"  # Default to JSON

    config_dict = config.model_dump()

    if format == "json":
        with open(config_path, 'w') as f:
            json.dump(config_dict, f, indent=2, default=str)
    elif format == "toml":
        with open(config_path, 'w') as f:
            toml.dump(config_dict, f)
    else:
        raise ValueError(f"Unsupported format: {format}")


# Default configuration instance
default_config = HybridPipeConfig()
