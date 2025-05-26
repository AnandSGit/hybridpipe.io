#!/usr/bin/env python3
"""
Setup script for HybridPipe.io Python implementation.
"""

from setuptools import setup, find_packages
from pathlib import Path

# Read the README file
this_directory = Path(__file__).parent
long_description = (this_directory / "README_PYTHON.md").read_text()

# Read requirements
def read_requirements(filename):
    """Read requirements from a file."""
    try:
        with open(filename, 'r') as f:
            return [line.strip() for line in f if line.strip() and not line.startswith('#')]
    except FileNotFoundError:
        return []

# Core requirements
install_requires = [
    "pydantic>=2.0.0",
    "structlog>=23.0.0",
    "toml>=0.10.0",
]

# Optional dependencies for different protocols
extras_require = {
    # CLI dependencies
    "cli": [
        "click>=8.0.0",
        "rich>=13.0.0",
    ],
    
    # Development dependencies
    "dev": [
        "pytest>=7.0.0",
        "pytest-asyncio>=0.21.0",
        "pytest-cov>=4.0.0",
        "black>=23.0.0",
        "isort>=5.12.0",
        "mypy>=1.0.0",
        "ruff>=0.1.0",
        "pre-commit>=3.0.0",
    ],
    
    # Protocol-specific dependencies
    "redis": ["aioredis>=2.0.0"],
    "kafka": ["confluent-kafka>=2.0.0"],
    "rabbitmq": ["aio-pika>=9.0.0"],
    "mqtt": ["paho-mqtt>=1.6.0"],
    "nats": ["nats-py>=2.0.0"],
    "zeromq": ["pyzmq>=25.0.0"],
    "amqp": ["py-amqp>=5.0.0"],
    
    # Testing with external services
    "integration": [
        "docker>=6.0.0",
        "testcontainers>=3.7.0",
    ],
    
    # Performance testing
    "performance": [
        "memory-profiler>=0.60.0",
        "psutil>=5.9.0",
    ],
    
    # Documentation
    "docs": [
        "sphinx>=6.0.0",
        "sphinx-rtd-theme>=1.2.0",
        "myst-parser>=1.0.0",
    ],
}

# All optional dependencies
extras_require["all"] = list(set(
    dep for deps in extras_require.values() for dep in deps
))

setup(
    name="hybridpipe",
    version="2.0.0",
    author="Anand S",
    author_email="Anand.S.Main@gmail.com",
    description="A unified messaging interface for microservices and distributed systems",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/AnandSGit/HybridPipe.io",
    project_urls={
        "Bug Tracker": "https://github.com/AnandSGit/HybridPipe.io/issues",
        "Documentation": "https://hybridpipe.readthedocs.io",
        "Source Code": "https://github.com/AnandSGit/HybridPipe.io",
    },
    packages=find_packages(exclude=["tests*", "examples*", "docs*"]),
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.12",
        "Programming Language :: Python :: 3.13",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Topic :: System :: Distributed Computing",
        "Topic :: System :: Networking",
        "Topic :: Communications",
        "Framework :: AsyncIO",
    ],
    python_requires=">=3.12",
    install_requires=install_requires,
    extras_require=extras_require,
    entry_points={
        "console_scripts": [
            "hybridpipe=hybridpipe.cli:main",
        ],
    },
    include_package_data=True,
    package_data={
        "hybridpipe": ["py.typed"],
    },
    zip_safe=False,
    keywords=[
        "messaging",
        "microservices",
        "distributed-systems",
        "async",
        "kafka",
        "redis",
        "rabbitmq",
        "mqtt",
        "nats",
        "zeromq",
        "tcp",
        "pubsub",
        "message-broker",
        "event-driven",
        "asyncio",
    ],
)
