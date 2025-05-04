// Package testutils provides utilities for testing the hybridpipe.io codebase.
package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertEventually repeatedly calls the check function until it returns true or the timeout is reached.
// It's useful for testing asynchronous code.
func AssertEventually(t *testing.T, check func() bool, timeout time.Duration, interval time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(interval)
	}
	
	assert.Fail(t, "Condition not met in time", msgAndArgs...)
}

// WithTimeout runs the given function with a timeout.
func WithTimeout(t *testing.T, timeout time.Duration, f func(ctx context.Context)) {
	t.Helper()
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	done := make(chan struct{})
	go func() {
		defer close(done)
		f(ctx)
	}()
	
	select {
	case <-done:
		// Function completed before timeout
	case <-time.After(timeout):
		t.Fatalf("Test timed out after %v", timeout)
	}
}

// TempDir creates a temporary directory for testing.
func TempDir(t *testing.T) string {
	t.Helper()
	
	dir, err := os.MkdirTemp("", "hybridpipe-test-")
	require.NoError(t, err)
	
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	
	return dir
}

// LoadTestData loads test data from the testdata directory.
func LoadTestData(t *testing.T, name string) []byte {
	t.Helper()
	
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	
	testdataDir := filepath.Join(filepath.Dir(filename), "..", "testdata")
	data, err := os.ReadFile(filepath.Join(testdataDir, name))
	require.NoError(t, err)
	
	return data
}

// JSONEqual asserts that two JSON strings are equal, ignoring formatting differences.
func JSONEqual(t *testing.T, expected, actual string) {
	t.Helper()
	
	var expectedJSON, actualJSON interface{}
	
	err := json.Unmarshal([]byte(expected), &expectedJSON)
	require.NoError(t, err, "Expected value is not valid JSON")
	
	err = json.Unmarshal([]byte(actual), &actualJSON)
	require.NoError(t, err, "Actual value is not valid JSON")
	
	assert.Equal(t, expectedJSON, actualJSON)
}

// CaptureOutput captures stdout and stderr during the execution of f.
func CaptureOutput(f func()) (stdout, stderr string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	
	os.Stdout = wOut
	os.Stderr = wErr
	
	outC := make(chan string)
	errC := make(chan string)
	
	// Copy stdout
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rOut)
		outC <- buf.String()
	}()
	
	// Copy stderr
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rErr)
		errC <- buf.String()
	}()
	
	// Call the function
	f()
	
	// Restore original stdout and stderr
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	
	stdout = <-outC
	stderr = <-errC
	
	return
}
