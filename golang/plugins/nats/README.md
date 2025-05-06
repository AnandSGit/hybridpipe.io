# NATS Server for HybridPipe.io

This directory contains the Docker configuration for running NATS server as part of the HybridPipe.io messaging system.

## Features

- **Optimized Docker Image**: Multi-stage build for minimal image size
- **Configurable**: Multiple configuration options via environment variables
- **Secure**: TLS support, authentication, and authorization
- **Scalable**: Clustering support for high availability
- **Persistent Messaging**: JetStream support for message persistence
- **Cross-Platform**: Multi-architecture support (amd64, arm64, arm/v7)

## Quick Start

### Basic Usage

To start a basic NATS server:

```bash
docker-compose up nats
```

This will start a NATS server with the default configuration, listening on port 4222 for client connections and 8222 for HTTP monitoring.

### TLS-Enabled Server

To start a NATS server with TLS enabled:

```bash
# First, create the necessary certificates
mkdir -p certs
# ... generate certificates ...

# Then start the TLS-enabled server
docker-compose --profile tls up nats-tls
```

This will start a NATS server with TLS enabled, listening on port 4223 for client connections and 8223 for HTTP monitoring.

### Clustered Deployment

To start a 3-node NATS cluster:

```bash
docker-compose --profile cluster up
```

This will start a 3-node NATS cluster with JetStream enabled for persistent messaging.

## Configuration

### Environment Variables

The following environment variables can be used to configure the NATS server:

- `CONFIG_FILE`: Path to the configuration file (default: `/nats/config/nats.conf`)
- `SERVER_ID`: Server ID for cluster mode
- `CLUSTER_PASSWORD`: Password for cluster authentication

### Configuration Files

The following configuration files are available:

- `nats.conf`: Basic configuration
- `tls.conf`: Configuration with TLS enabled
- `cluster.conf`: Configuration for clustered deployment

You can customize these files or create your own configuration files.

## Building the Image

To build the Docker image:

```bash
docker build -t hybridpipe/nats:latest .
```

You can specify the NATS version to build:

```bash
docker build --build-arg NATS_VERSION=v2.10.5 -t hybridpipe/nats:2.10.5 .
```

## Advanced Usage

### Custom Configuration

You can mount your own configuration file:

```bash
docker run -v /path/to/your/config.conf:/nats/config/custom.conf -e CONFIG_FILE=/nats/config/custom.conf hybridpipe/nats:latest
```

### JetStream

To enable JetStream (persistent messaging):

```bash
docker run -v /path/to/data:/nats/data hybridpipe/nats:latest --config /nats/config/nats.conf --jetstream
```

### Monitoring

The NATS server exposes a monitoring endpoint on port 8222. You can access it at:

```
http://localhost:8222/
```

## Integration with HybridPipe.io

This NATS server is designed to work seamlessly with the HybridPipe.io messaging system. To use it with HybridPipe.io:

1. Start the NATS server using one of the methods above
2. Configure HybridPipe.io to connect to the NATS server at `localhost:4222` (or the appropriate port)
3. If using TLS, ensure that HybridPipe.io is configured with the correct certificates

## Security Considerations

- Change default passwords in production
- Use TLS for secure communication
- Restrict access to the monitoring port (8222)
- Use authentication and authorization for production deployments

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.
