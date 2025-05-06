# Scripts

This directory contains utility scripts for the HybridPipe system.

## Purpose

The `scripts` directory is intended for scripts that help with:

- Setting up development environments
- Installing dependencies
- Building and packaging the system
- Running tests
- Generating documentation
- Deploying the system

## Available Scripts

### Installation Scripts

- `install_kafka.sh` - Installs Apache Kafka on Linux/macOS
- `install_kafka.ps1` - Installs Apache Kafka on Windows
- `install_nats.sh` - Installs NATS on Linux/macOS
- `install_nats.ps1` - Installs NATS on Windows
- `install_rabbitmq.sh` - Installs RabbitMQ on Linux/macOS
- `install_rabbitmq.ps1` - Installs RabbitMQ on Windows

### Build Scripts

- `build.sh` - Builds the HybridPipe system on Linux/macOS
- `build.ps1` - Builds the HybridPipe system on Windows

### Test Scripts

- `run_tests.sh` - Runs all tests on Linux/macOS
- `run_tests.ps1` - Runs all tests on Windows

## Usage

To run a script, navigate to the scripts directory and execute it:

```bash
cd scripts
./install_kafka.sh
```

On Windows:

```powershell
cd scripts
.\install_kafka.ps1
```

Last updated: April 19, 2025
