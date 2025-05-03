# Utilities

This directory contains utility functions and helpers for the HybridPipe system.

## Purpose

The `utils` directory is intended for common utility functions that are used across the HybridPipe system, such as:

- Logging utilities
- Error handling
- Configuration helpers
- Testing utilities
- Serialization helpers
- Network utilities

## Usage

Utility functions are organized into packages based on their functionality. Import the appropriate package to use the utilities:

```go
import (
    "hybridpipe.io/utils/logging"
    "hybridpipe.io/utils/config"
)

func main() {
    logger := logging.NewLogger("my-service")
    config := config.LoadConfig("config.yaml")
    
    logger.Info("Configuration loaded", "config", config)
}
```

## Available Utilities

- `logging` - Logging utilities
- `config` - Configuration helpers
- `errors` - Error handling utilities
- `network` - Network utilities
- `serialization` - Serialization helpers
- `testing` - Testing utilities

Last updated: April 19, 2025
