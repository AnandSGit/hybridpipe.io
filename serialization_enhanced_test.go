package hybridpipe

import (
	"reflect"
	"testing"
	"time"
)

// TestTypeConversion tests the type conversion functionality.
func TestTypeConversion(t *testing.T) {
	// Test time.Time <-> int64 conversion
	t.Run("TimeToUnixMillis", func(t *testing.T) {
		// Create a test time
		testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		expectedMillis := testTime.UnixNano() / int64(time.Millisecond)

		// Convert to Unix milliseconds
		converted, err := TimeToUnixMillis(testTime)
		if err != nil {
			t.Fatalf("Failed to convert time to Unix milliseconds: %v", err)
		}

		// Check the result
		millis, ok := converted.(int64)
		if !ok {
			t.Fatalf("Converted value is not an int64: %T", converted)
		}

		if millis != expectedMillis {
			t.Errorf("Incorrect conversion. Expected: %d, Got: %d", expectedMillis, millis)
		}

		// Convert back to time.Time
		reconverted, err := UnixMillisToTime(millis)
		if err != nil {
			t.Fatalf("Failed to convert Unix milliseconds to time: %v", err)
		}

		// Check the result
		reconvertedTime, ok := reconverted.(time.Time)
		if !ok {
			t.Fatalf("Reconverted value is not a time.Time: %T", reconverted)
		}

		// Compare the times (truncate to milliseconds for accurate comparison)
		if !testTime.Truncate(time.Millisecond).Equal(reconvertedTime) {
			t.Errorf("Incorrect reconversion. Expected: %v, Got: %v", testTime, reconvertedTime)
		}
	})

	// Test time.Duration <-> int64 conversion
	t.Run("DurationToMillis", func(t *testing.T) {
		// Create a test duration
		testDuration := 5*time.Second + 500*time.Millisecond
		expectedMillis := testDuration.Milliseconds()

		// Convert to milliseconds
		converted, err := DurationToMillis(testDuration)
		if err != nil {
			t.Fatalf("Failed to convert duration to milliseconds: %v", err)
		}

		// Check the result
		millis, ok := converted.(int64)
		if !ok {
			t.Fatalf("Converted value is not an int64: %T", converted)
		}

		if millis != expectedMillis {
			t.Errorf("Incorrect conversion. Expected: %d, Got: %d", expectedMillis, millis)
		}

		// Convert back to time.Duration
		reconverted, err := MillisToDuration(millis)
		if err != nil {
			t.Fatalf("Failed to convert milliseconds to duration: %v", err)
		}

		// Check the result
		reconvertedDuration, ok := reconverted.(time.Duration)
		if !ok {
			t.Fatalf("Reconverted value is not a time.Duration: %T", reconverted)
		}

		if reconvertedDuration != testDuration {
			t.Errorf("Incorrect reconversion. Expected: %v, Got: %v", testDuration, reconvertedDuration)
		}
	})
}

// TestTypeRegistry tests the TypeRegistry functionality.
func TestTypeRegistry(t *testing.T) {
	// Create a new type registry
	registry := NewTypeRegistry()

	// Register a custom converter
	type CustomType struct {
		Value string
	}

	type WireType struct {
		EncodedValue string
	}

	// Register converters
	registry.RegisterSerializationConverter(
		reflect.TypeOf(CustomType{}),
		func(value interface{}) (interface{}, error) {
			customValue := value.(CustomType)
			return WireType{EncodedValue: "encoded:" + customValue.Value}, nil
		},
	)

	registry.RegisterDeserializationConverter(
		reflect.TypeOf(WireType{}),
		func(value interface{}) (interface{}, error) {
			wireValue := value.(WireType)
			return CustomType{Value: wireValue.EncodedValue[8:]}, nil
		},
	)

	// Test serialization conversion
	customValue := CustomType{Value: "test"}
	converted, err := registry.ConvertForSerialization(customValue)
	if err != nil {
		t.Fatalf("Failed to convert for serialization: %v", err)
	}

	wireValue, ok := converted.(WireType)
	if !ok {
		t.Fatalf("Converted value is not a WireType: %T", converted)
	}

	if wireValue.EncodedValue != "encoded:test" {
		t.Errorf("Incorrect conversion. Expected: %s, Got: %s", "encoded:test", wireValue.EncodedValue)
	}

	// Test deserialization conversion
	reconverted, err := registry.ConvertForDeserialization(wireValue)
	if err != nil {
		t.Fatalf("Failed to convert for deserialization: %v", err)
	}

	reconvertedValue, ok := reconverted.(CustomType)
	if !ok {
		t.Fatalf("Reconverted value is not a CustomType: %T", reconverted)
	}

	if reconvertedValue.Value != "test" {
		t.Errorf("Incorrect reconversion. Expected: %s, Got: %s", "test", reconvertedValue.Value)
	}
}

// TestCrossLanguageCompatibility tests the cross-language compatibility features.
func TestCrossLanguageCompatibility(t *testing.T) {
	// Define a test struct with various types
	type TestStruct struct {
		String    string
		Int       int
		Float     float64
		Bool      bool
		Time      time.Time
		Duration  time.Duration
		StringMap map[string]string
		IntArray  []int
	}

	// Create a test instance
	testValue := TestStruct{
		String:    "test",
		Int:       42,
		Float:     3.14,
		Bool:      true,
		Time:      time.Now(),
		Duration:  5 * time.Second,
		StringMap: map[string]string{"key": "value"},
		IntArray:  []int{1, 2, 3},
	}

	// Test with JSON format (most cross-language compatible)
	t.Run("JSON", func(t *testing.T) {
		options := SerializationOptions{
			Format:                  FormatJSON,
			CrossLanguageCompatible: true,
		}

		// Serialize
		data, err := EncodeWithOptions(testValue, options)
		if err != nil {
			t.Fatalf("Failed to serialize with JSON: %v", err)
		}

		// Deserialize
		var result TestStruct
		err = DecodeWithOptions(data, &result, options)
		if err != nil {
			t.Fatalf("Failed to deserialize with JSON: %v", err)
		}

		// Verify basic fields
		if result.String != testValue.String {
			t.Errorf("String mismatch: got %q, want %q", result.String, testValue.String)
		}
		if result.Int != testValue.Int {
			t.Errorf("Int mismatch: got %d, want %d", result.Int, testValue.Int)
		}
		if result.Float != testValue.Float {
			t.Errorf("Float mismatch: got %f, want %f", result.Float, testValue.Float)
		}
		if result.Bool != testValue.Bool {
			t.Errorf("Bool mismatch: got %v, want %v", result.Bool, testValue.Bool)
		}
	})

	// Test with MessagePack format (efficient and cross-language compatible)
	t.Run("MessagePack", func(t *testing.T) {
		options := SerializationOptions{
			Format:                  FormatMsgPack,
			CrossLanguageCompatible: true,
		}

		// Serialize
		data, err := EncodeWithOptions(testValue, options)
		if err != nil {
			t.Fatalf("Failed to serialize with MessagePack: %v", err)
		}

		// Deserialize
		var result TestStruct
		err = DecodeWithOptions(data, &result, options)
		if err != nil {
			t.Fatalf("Failed to deserialize with MessagePack: %v", err)
		}

		// Verify basic fields
		if result.String != testValue.String {
			t.Errorf("String mismatch: got %q, want %q", result.String, testValue.String)
		}
		if result.Int != testValue.Int {
			t.Errorf("Int mismatch: got %d, want %d", result.Int, testValue.Int)
		}
		if result.Float != testValue.Float {
			t.Errorf("Float mismatch: got %f, want %f", result.Float, testValue.Float)
		}
		if result.Bool != testValue.Bool {
			t.Errorf("Bool mismatch: got %v, want %v", result.Bool, testValue.Bool)
		}
	})
}
