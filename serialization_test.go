package hybridpipe

import (
	"testing"
	"time"
)

// TestMessage is a test message type
type TestMessage struct {
	Text      string
	Timestamp time.Time
	Count     int
	Tags      []string
}

func TestSerializationFormats(t *testing.T) {
	// Register the message type for gob encoding
	Enable(TestMessage{})

	// Create a test message
	original := TestMessage{
		Text:      "Hello, world!",
		Timestamp: time.Now(),
		Count:     42,
		Tags:      []string{"test", "serialization", "hybridpipe"},
	}

	// Test GOB serialization
	t.Run("GOB", func(t *testing.T) {
		options := SerializationOptions{
			Format: FormatGob,
		}

		// Serialize
		data, err := EncodeWithOptions(original, options)
		if err != nil {
			t.Fatalf("Failed to serialize with GOB: %v", err)
		}

		// Deserialize
		var result TestMessage
		err = DecodeWithOptions(data, &result)
		if err != nil {
			t.Fatalf("Failed to deserialize with GOB: %v", err)
		}

		// Verify
		if result.Text != original.Text {
			t.Errorf("Text mismatch: got %q, want %q", result.Text, original.Text)
		}
		if result.Count != original.Count {
			t.Errorf("Count mismatch: got %d, want %d", result.Count, original.Count)
		}
		if len(result.Tags) != len(original.Tags) {
			t.Errorf("Tags length mismatch: got %d, want %d", len(result.Tags), len(original.Tags))
		}
	})

	// Test JSON serialization
	t.Run("JSON", func(t *testing.T) {
		options := SerializationOptions{
			Format: FormatJSON,
		}

		// Serialize
		data, err := EncodeWithOptions(original, options)
		if err != nil {
			t.Fatalf("Failed to serialize with JSON: %v", err)
		}

		// Deserialize
		var result TestMessage
		err = DecodeWithOptions(data, &result)
		if err != nil {
			t.Fatalf("Failed to deserialize with JSON: %v", err)
		}

		// Verify
		if result.Text != original.Text {
			t.Errorf("Text mismatch: got %q, want %q", result.Text, original.Text)
		}
		if result.Count != original.Count {
			t.Errorf("Count mismatch: got %d, want %d", result.Count, original.Count)
		}
		if len(result.Tags) != len(original.Tags) {
			t.Errorf("Tags length mismatch: got %d, want %d", len(result.Tags), len(original.Tags))
		}
	})

	// Test MessagePack serialization
	t.Run("MessagePack", func(t *testing.T) {
		options := SerializationOptions{
			Format: FormatMsgPack,
		}

		// Serialize
		data, err := EncodeWithOptions(original, options)
		if err != nil {
			t.Fatalf("Failed to serialize with MessagePack: %v", err)
		}

		// Deserialize
		var result TestMessage
		err = DecodeWithOptions(data, &result)
		if err != nil {
			t.Fatalf("Failed to deserialize with MessagePack: %v", err)
		}

		// Verify
		if result.Text != original.Text {
			t.Errorf("Text mismatch: got %q, want %q", result.Text, original.Text)
		}
		if result.Count != original.Count {
			t.Errorf("Count mismatch: got %d, want %d", result.Count, original.Count)
		}
		if len(result.Tags) != len(original.Tags) {
			t.Errorf("Tags length mismatch: got %d, want %d", len(result.Tags), len(original.Tags))
		}
	})

	// Test compression
	t.Run("Compression", func(t *testing.T) {
		options := SerializationOptions{
			Format:      FormatJSON,
			Compression: true,
		}

		// Serialize
		data, err := EncodeWithOptions(original, options)
		if err != nil {
			t.Fatalf("Failed to serialize with compression: %v", err)
		}

		// Deserialize
		var result TestMessage
		err = DecodeWithOptions(data, &result)
		if err != nil {
			t.Fatalf("Failed to deserialize with compression: %v", err)
		}

		// Verify
		if result.Text != original.Text {
			t.Errorf("Text mismatch: got %q, want %q", result.Text, original.Text)
		}
		if result.Count != original.Count {
			t.Errorf("Count mismatch: got %d, want %d", result.Count, original.Count)
		}
		if len(result.Tags) != len(original.Tags) {
			t.Errorf("Tags length mismatch: got %d, want %d", len(result.Tags), len(original.Tags))
		}
	})

	// Test backward compatibility
	t.Run("BackwardCompatibility", func(t *testing.T) {
		// Serialize using the old Encode function
		data, err := Encode(original)
		if err != nil {
			t.Fatalf("Failed to serialize with Encode: %v", err)
		}

		// Deserialize using the old Decode function
		var result TestMessage
		err = Decode(data, &result)
		if err != nil {
			t.Fatalf("Failed to deserialize with Decode: %v", err)
		}

		// Verify
		if result.Text != original.Text {
			t.Errorf("Text mismatch: got %q, want %q", result.Text, original.Text)
		}
		if result.Count != original.Count {
			t.Errorf("Count mismatch: got %d, want %d", result.Count, original.Count)
		}
		if len(result.Tags) != len(original.Tags) {
			t.Errorf("Tags length mismatch: got %d, want %d", len(result.Tags), len(original.Tags))
		}
	})
}
