# Command Line Tools

This directory contains command-line tools and applications that use the HybridPipe library.

## Purpose

The `cmd` directory is intended for executable applications that demonstrate or utilize the HybridPipe system. Each subdirectory should contain a separate application with its own `main.go` file.

## Examples

- `cmd/accept` - A command-line tool for accepting messages from various messaging systems
- `cmd/dispatch` - A command-line tool for dispatching messages to various messaging systems
- `cmd/monitor` - A command-line tool for monitoring message flow through the HybridPipe system

## Usage

To build and run an application, navigate to its directory and use the standard Go build commands:

```bash
cd cmd/accept
go build
./accept
```

Last updated: April 19, 2025
