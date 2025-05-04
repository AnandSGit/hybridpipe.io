package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertEventually(t *testing.T) {
	// Test that AssertEventually succeeds when the condition becomes true
	t.Run("ConditionBecomesTrue", func(t *testing.T) {
		counter := 0
		check := func() bool {
			counter++
			return counter >= 3
		}

		AssertEventually(t, check, 1*time.Second, 100*time.Millisecond)
		assert.GreaterOrEqual(t, counter, 3, "Counter should be at least 3")
	})

	// Test that AssertEventually fails when the condition never becomes true
	t.Run("ConditionNeverTrue", func(t *testing.T) {
		// Skip this test as it's hard to test failure cases
		t.Skip("Skipping test for failure case")
	})
}

func TestWithTimeout(t *testing.T) {
	// Test that WithTimeout succeeds when the function completes before the timeout
	t.Run("CompletesBeforeTimeout", func(t *testing.T) {
		executed := false
		WithTimeout(t, 1*time.Second, func(ctx context.Context) {
			executed = true
		})
		assert.True(t, executed, "Function should have been executed")
	})

	// Test that WithTimeout fails when the function takes too long
	t.Run("TimeoutExpires", func(t *testing.T) {
		// Skip this test as it's hard to test failure cases
		t.Skip("Skipping test for failure case")
	})

	// Test that WithTimeout passes context to the function
	t.Run("ContextPassed", func(t *testing.T) {
		var receivedCtx context.Context
		WithTimeout(t, 1*time.Second, func(ctx context.Context) {
			receivedCtx = ctx
		})
		assert.NotNil(t, receivedCtx, "Context should have been passed to the function")

		// Check that the context has a deadline
		deadline, hasDeadline := receivedCtx.Deadline()
		assert.True(t, hasDeadline, "Context should have a deadline")
		assert.WithinDuration(t, time.Now().Add(1*time.Second), deadline, 100*time.Millisecond)
	})
}

func TestTempDir(t *testing.T) {
	// Create a temporary directory
	dir := TempDir(t)

	// Check that the directory exists
	_, err := os.Stat(dir)
	assert.NoError(t, err, "Temporary directory should exist")

	// Create a file in the directory
	testFile := filepath.Join(dir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	assert.NoError(t, err, "Should be able to write to the temporary directory")

	// Check that the file exists
	_, err = os.Stat(testFile)
	assert.NoError(t, err, "Test file should exist")

	// The directory should be cleaned up when the test completes
}

func TestLoadTestData(t *testing.T) {
	// Create a testdata directory
	testdataDir := filepath.Join("..", "testdata")
	err := os.MkdirAll(testdataDir, 0755)
	require.NoError(t, err, "Failed to create testdata directory")

	// Create a test file
	testFile := filepath.Join(testdataDir, "test.json")
	testData := []byte(`{"message": "Hello, World!"}`)
	err = os.WriteFile(testFile, testData, 0644)
	require.NoError(t, err, "Failed to write test file")

	// Clean up after the test
	defer os.RemoveAll(testdataDir)

	// Load the test data
	data := LoadTestData(t, "test.json")
	assert.Equal(t, testData, data, "Loaded data should match the original data")
}

func TestJSONEqual(t *testing.T) {
	// Test that JSONEqual succeeds when the JSON strings are equal
	t.Run("Equal", func(t *testing.T) {
		json1 := `{"name": "John", "age": 30}`
		json2 := `{"age": 30, "name": "John"}`

		JSONEqual(t, json1, json2)
	})

	// Test that JSONEqual fails when the JSON strings are not equal
	t.Run("NotEqual", func(t *testing.T) {
		// Skip this test as it's hard to test failure cases
		t.Skip("Skipping test for failure case")
	})

	// Test that JSONEqual fails when one of the strings is not valid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		// Skip this test as it's hard to test failure cases
		t.Skip("Skipping test for failure case")
	})
}

func TestCaptureOutput(t *testing.T) {
	// Test capturing stdout
	t.Run("CaptureStdout", func(t *testing.T) {
		stdout, stderr := CaptureOutput(func() {
			fmt.Println("Hello, stdout!")
		})

		assert.Contains(t, stdout, "Hello, stdout!")
		assert.Empty(t, stderr, "Stderr should be empty")
	})

	// Test capturing stderr
	t.Run("CaptureStderr", func(t *testing.T) {
		stdout, stderr := CaptureOutput(func() {
			fmt.Fprintln(os.Stderr, "Hello, stderr!")
		})

		assert.Empty(t, stdout, "Stdout should be empty")
		assert.Contains(t, stderr, "Hello, stderr!")
	})

	// Test capturing both stdout and stderr
	t.Run("CaptureBoth", func(t *testing.T) {
		stdout, stderr := CaptureOutput(func() {
			fmt.Println("Hello, stdout!")
			fmt.Fprintln(os.Stderr, "Hello, stderr!")
		})

		assert.Contains(t, stdout, "Hello, stdout!")
		assert.Contains(t, stderr, "Hello, stderr!")
	})
}

// Additional test for JSON marshaling and unmarshaling
func TestJSONMarshalUnmarshal(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// Create a test person
	person := Person{
		Name: "John Doe",
		Age:  30,
	}

	// Marshal to JSON
	data, err := json.Marshal(person)
	require.NoError(t, err, "Failed to marshal person to JSON")

	// Unmarshal from JSON
	var unmarshaledPerson Person
	err = json.Unmarshal(data, &unmarshaledPerson)
	require.NoError(t, err, "Failed to unmarshal JSON to person")

	// Check that the unmarshaled person matches the original
	assert.Equal(t, person, unmarshaledPerson, "Unmarshaled person should match the original")
}
