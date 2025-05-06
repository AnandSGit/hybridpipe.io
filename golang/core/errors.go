package core

import (
	"errors"
	"fmt"
)

// Common errors that can be returned by HybridPipe operations
var (
	// ErrNotConnected is returned when an operation is attempted on a disconnected HybridPipe
	ErrNotConnected = errors.New("not connected to messaging system")

	// ErrAlreadyConnected is returned when Connect is called on an already connected HybridPipe
	ErrAlreadyConnected = errors.New("already connected to messaging system")

	// ErrInvalidPipe is returned when an invalid pipe name is provided
	ErrInvalidPipe = errors.New("invalid pipe name")

	// ErrPipeNotFound is returned when a pipe is not found
	ErrPipeNotFound = errors.New("pipe not found")

	// ErrPipeAlreadyExists is returned when a pipe already exists
	ErrPipeAlreadyExists = errors.New("pipe already exists")

	// ErrSerializationFailed is returned when serialization fails
	ErrSerializationFailed = errors.New("serialization failed")

	// ErrDeserializationFailed is returned when deserialization fails
	ErrDeserializationFailed = errors.New("deserialization failed")

	// ErrInvalidData is returned when invalid data is provided
	ErrInvalidData = errors.New("invalid data")

	// ErrInvalidCallback is returned when a nil callback is provided
	ErrInvalidCallback = errors.New("invalid callback function")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timed out")

	// ErrClosed is returned when an operation is attempted on a closed HybridPipe
	ErrClosed = errors.New("hybridpipe is closed")
)

// WrapError wraps an error with a message
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
