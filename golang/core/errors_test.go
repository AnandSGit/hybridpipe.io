package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	// Test error constants
	assert.NotNil(t, ErrNotConnected, "ErrNotConnected should not be nil")
	assert.NotNil(t, ErrAlreadyConnected, "ErrAlreadyConnected should not be nil")
	assert.NotNil(t, ErrPipeNotFound, "ErrPipeNotFound should not be nil")
	assert.NotNil(t, ErrPipeAlreadyExists, "ErrPipeAlreadyExists should not be nil")
	assert.NotNil(t, ErrInvalidData, "ErrInvalidData should not be nil")
	assert.NotNil(t, ErrTimeout, "ErrTimeout should not be nil")
	assert.NotNil(t, ErrSerializationFailed, "ErrSerializationFailed should not be nil")
	assert.NotNil(t, ErrDeserializationFailed, "ErrDeserializationFailed should not be nil")
}

func TestWrapError(t *testing.T) {
	// Test wrapping a nil error
	wrapped := WrapError(nil, "test message")
	assert.Nil(t, wrapped, "WrapError should return nil when error is nil")

	// Test wrapping a non-nil error
	originalErr := errors.New("original error")
	wrapped = WrapError(originalErr, "test message")
	assert.NotNil(t, wrapped, "WrapError should not return nil when error is not nil")
	assert.Contains(t, wrapped.Error(), "test message", "Wrapped error should contain the message")
	assert.Contains(t, wrapped.Error(), "original error", "Wrapped error should contain the original error")

	// Test wrapping with a different message
	wrapped = WrapError(originalErr, "another test message")
	assert.NotNil(t, wrapped, "WrapError should not return nil when error is not nil")
	assert.Contains(t, wrapped.Error(), "another test message", "Wrapped error should contain the message")
	assert.Contains(t, wrapped.Error(), "original error", "Wrapped error should contain the original error")

	// Test error unwrapping
	unwrapped := errors.Unwrap(wrapped)
	assert.Equal(t, originalErr, unwrapped, "Unwrapped error should be the original error")

	// Test error is
	assert.True(t, errors.Is(wrapped, originalErr), "errors.Is should return true for the original error")
}

func TestErrorTypes(t *testing.T) {
	// Test that errors can be compared with errors.Is
	testCases := []struct {
		err    error
		target error
	}{
		{ErrNotConnected, ErrNotConnected},
		{WrapError(ErrNotConnected, "test"), ErrNotConnected},
		{ErrAlreadyConnected, ErrAlreadyConnected},
		{WrapError(ErrAlreadyConnected, "test"), ErrAlreadyConnected},
		{ErrPipeNotFound, ErrPipeNotFound},
		{WrapError(ErrPipeNotFound, "test"), ErrPipeNotFound},
		{ErrPipeAlreadyExists, ErrPipeAlreadyExists},
		{WrapError(ErrPipeAlreadyExists, "test"), ErrPipeAlreadyExists},
		{ErrInvalidData, ErrInvalidData},
		{WrapError(ErrInvalidData, "test"), ErrInvalidData},
		{ErrTimeout, ErrTimeout},
		{WrapError(ErrTimeout, "test"), ErrTimeout},
		{ErrSerializationFailed, ErrSerializationFailed},
		{WrapError(ErrSerializationFailed, "test"), ErrSerializationFailed},
		{ErrDeserializationFailed, ErrDeserializationFailed},
		{WrapError(ErrDeserializationFailed, "test"), ErrDeserializationFailed},
	}

	for _, tc := range testCases {
		assert.True(t, errors.Is(tc.err, tc.target), "errors.Is should return true for %v and %v", tc.err, tc.target)
	}
}
