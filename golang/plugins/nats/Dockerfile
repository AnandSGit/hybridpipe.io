# ============================================================================
# NATS Server Dockerfile for HybridPipe.io
# ============================================================================
# This Dockerfile builds a lightweight, configurable NATS server container
# for use with the HybridPipe.io messaging system.
#
# Author: HybridPipe.io
# Date: April 21, 2025
# Version: 2.0
# ============================================================================

# Build stage
# -----------
# Using a recent Go version for better performance and security
FROM golang:1.24-alpine AS builder

# Build arguments with defaults
ARG NATS_VERSION=v2.10.5
ARG TARGETPLATFORM=linux/amd64

# Set build environment variables
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GO111MODULE=on

# Set the target architecture based on build argument
RUN case "${TARGETPLATFORM}" in \
    "linux/amd64") export GOARCH=amd64 ;; \
    "linux/arm64") export GOARCH=arm64 ;; \
    "linux/arm/v7") export GOARCH=arm ;; \
    *) export GOARCH=amd64 ;; \
    esac

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# Clone specific version of NATS server
WORKDIR /src
RUN git clone --branch ${NATS_VERSION} --depth 1 https://github.com/nats-io/nats-server.git

# Build NATS server with optimizations
WORKDIR /src/nats-server
RUN go build -trimpath -ldflags "-s -w -X github.com/nats-io/nats-server/server.gitCommit=$(git rev-parse --short HEAD)" -o /nats-server

# Create a minimal runtime image
# ------------------------------
# Using distroless for minimal attack surface and small image size
FROM gcr.io/distroless/static:nonroot

# Metadata
LABEL maintainer="HybridPipe.io" \
      description="NATS Server for HybridPipe.io messaging system" \
      version="2.0" \
      org.opencontainers.image.source="https://github.com/AnandSGit/hybridpipe.io"

# Copy the binary from the builder stage
COPY --from=builder /nats-server /nats-server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Create a directory for configuration and data
WORKDIR /nats

# Copy configuration files
COPY config/ /nats/config/

# Default configuration file
ENV CONFIG_FILE="/nats/config/nats.conf"

# Expose ports:
# - 4222: client connections
# - 6222: route connections (clustering)
# - 8222: HTTP monitoring
# - 7777: Leafnode connections
# - 7422: WebSocket connections
EXPOSE 4222 6222 8222 7777 7422

# Set health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD ["wget", "-q", "--spider", "http://localhost:8222/varz"]

# Run as non-root user for security
USER nonroot:nonroot

# Command to run
ENTRYPOINT ["/nats-server"]
CMD ["--config", "${CONFIG_FILE}"]
