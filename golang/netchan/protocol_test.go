package netchan

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// TestMessageEncoding tests the encoding and decoding of messages.
func TestMessageEncoding(t *testing.T) {
	// Test cases
	testCases := []struct {
		name     string
		message  *Message
		wantSize int
	}{
		{
			name: "EmptyMessage",
			message: &Message{
				Type:     MessageData,
				Sequence: 1,
				Flags:    FlagNone,
				Payload:  nil,
			},
			wantSize: HeaderSize,
		},
		{
			name: "DataMessage",
			message: &Message{
				Type:     MessageData,
				Sequence: 42,
				Flags:    FlagNone,
				Payload:  []byte("Hello, Protocol!"),
			},
			wantSize: HeaderSize + len("Hello, Protocol!"),
		},
		{
			name: "HandshakeMessage",
			message: &Message{
				Type:     MessageHandshake,
				Sequence: 1,
				Flags:    FlagNone,
				Payload:  []byte{1, 2, 3, 4},
			},
			wantSize: HeaderSize + 4,
		},
		{
			name: "PingMessage",
			message: &Message{
				Type:     MessagePing,
				Sequence: 100,
				Flags:    FlagNone,
				Payload:  nil,
			},
			wantSize: HeaderSize,
		},
		{
			name: "PongMessage",
			message: &Message{
				Type:     MessagePong,
				Sequence: 100,
				Flags:    FlagNone,
				Payload:  nil,
			},
			wantSize: HeaderSize,
		},
		{
			name: "CloseMessage",
			message: &Message{
				Type:     MessageClose,
				Sequence: 999,
				Flags:    FlagNone,
				Payload:  nil,
			},
			wantSize: HeaderSize,
		},
		{
			name: "MessageWithFlags",
			message: &Message{
				Type:     MessageData,
				Sequence: 42,
				Flags:    FlagCompressed,
				Payload:  []byte("Compressed data"),
			},
			wantSize: HeaderSize + len("Compressed data"),
		},
		{
			name: "LargeMessage",
			message: &Message{
				Type:     MessageData,
				Sequence: 1000,
				Flags:    FlagNone,
				Payload:  bytes.Repeat([]byte("X"), 1000),
			},
			wantSize: HeaderSize + 1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode the message
			encoded, err := tc.message.Encode()
			if err != nil {
				t.Fatalf("Failed to encode message: %v", err)
			}

			// Check the size
			if len(encoded) != tc.wantSize {
				t.Errorf("Encoded message size = %d, want %d", len(encoded), tc.wantSize)
			}

			// Decode the message
			decoded, err := DecodeMessage(encoded)
			if err != nil {
				t.Fatalf("Failed to decode message: %v", err)
			}

			// Check the decoded message
			if decoded.Type != tc.message.Type {
				t.Errorf("Decoded message type = %d, want %d", decoded.Type, tc.message.Type)
			}
			if decoded.Sequence != tc.message.Sequence {
				t.Errorf("Decoded message sequence = %d, want %d", decoded.Sequence, tc.message.Sequence)
			}
			if decoded.Flags != tc.message.Flags {
				t.Errorf("Decoded message flags = %d, want %d", decoded.Flags, tc.message.Flags)
			}
			if !bytes.Equal(decoded.Payload, tc.message.Payload) {
				t.Errorf("Decoded message payload doesn't match original")
			}
		})
	}
}

// TestMessageReadWrite tests reading and writing messages to a stream.
func TestMessageReadWrite(t *testing.T) {
	// Create a buffer to simulate a stream
	var buffer bytes.Buffer

	// Create a test message
	message := &Message{
		Type:     MessageData,
		Sequence: 42,
		Flags:    FlagNone,
		Payload:  []byte("Hello, Stream!"),
	}

	// Write the message to the buffer
	encoded, err := message.Encode()
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}
	_, err = buffer.Write(encoded)
	if err != nil {
		t.Fatalf("Failed to write message to buffer: %v", err)
	}

	// Read the message from the buffer
	readMessage, err := ReadMessage(&buffer)
	if err != nil {
		t.Fatalf("Failed to read message from buffer: %v", err)
	}

	// Check the read message
	if readMessage.Type != message.Type {
		t.Errorf("Read message type = %d, want %d", readMessage.Type, message.Type)
	}
	if readMessage.Sequence != message.Sequence {
		t.Errorf("Read message sequence = %d, want %d", readMessage.Sequence, message.Sequence)
	}
	if readMessage.Flags != message.Flags {
		t.Errorf("Read message flags = %d, want %d", readMessage.Flags, message.Flags)
	}
	if !bytes.Equal(readMessage.Payload, message.Payload) {
		t.Errorf("Read message payload doesn't match original")
	}
}

// TestMessageReadWriteMultiple tests reading and writing multiple messages to a stream.
func TestMessageReadWriteMultiple(t *testing.T) {
	// Create a buffer to simulate a stream
	var buffer bytes.Buffer

	// Create test messages
	messages := []*Message{
		{
			Type:     MessageHandshake,
			Sequence: 1,
			Flags:    FlagNone,
			Payload:  []byte{1, 2, 3, 4},
		},
		{
			Type:     MessageData,
			Sequence: 2,
			Flags:    FlagNone,
			Payload:  []byte("First message"),
		},
		{
			Type:     MessageData,
			Sequence: 3,
			Flags:    FlagCompressed,
			Payload:  []byte("Second message"),
		},
		{
			Type:     MessagePing,
			Sequence: 4,
			Flags:    FlagNone,
			Payload:  nil,
		},
		{
			Type:     MessagePong,
			Sequence: 5,
			Flags:    FlagNone,
			Payload:  nil,
		},
	}

	// Write all messages to the buffer
	for _, msg := range messages {
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Failed to encode message: %v", err)
		}
		_, err = buffer.Write(encoded)
		if err != nil {
			t.Fatalf("Failed to write message to buffer: %v", err)
		}
	}

	// Read all messages from the buffer
	for i, expected := range messages {
		readMessage, err := ReadMessage(&buffer)
		if err != nil {
			t.Fatalf("Failed to read message %d from buffer: %v", i, err)
		}

		// Check the read message
		if readMessage.Type != expected.Type {
			t.Errorf("Message %d: type = %d, want %d", i, readMessage.Type, expected.Type)
		}
		if readMessage.Sequence != expected.Sequence {
			t.Errorf("Message %d: sequence = %d, want %d", i, readMessage.Sequence, expected.Sequence)
		}
		if readMessage.Flags != expected.Flags {
			t.Errorf("Message %d: flags = %d, want %d", i, readMessage.Flags, expected.Flags)
		}
		if !bytes.Equal(readMessage.Payload, expected.Payload) {
			t.Errorf("Message %d: payload doesn't match original", i)
		}
	}

	// Verify that the buffer is empty
	if buffer.Len() > 0 {
		t.Errorf("Buffer is not empty after reading all messages: %d bytes remaining", buffer.Len())
	}
}

// TestMessageReadIncomplete tests reading an incomplete message.
func TestMessageReadIncomplete(t *testing.T) {
	// Create a buffer with an incomplete message
	var buffer bytes.Buffer

	// Write just the header
	header := make([]byte, HeaderSize)
	header[0] = byte(MessageData)
	binary.BigEndian.PutUint32(header[1:5], 42)
	header[5] = FlagNone
	binary.BigEndian.PutUint32(header[6:10], 10) // Payload length = 10
	buffer.Write(header)

	// Write only part of the payload
	buffer.Write([]byte("Hello")) // Only 5 bytes, but header says 10

	// Try to read the message
	_, err := ReadMessage(&buffer)
	if err != io.ErrUnexpectedEOF && err != io.EOF {
		t.Errorf("Expected EOF error, got %v", err)
	}
}

// TestMessageTooLarge tests encoding a message that exceeds the maximum size.
func TestMessageTooLarge(t *testing.T) {
	// Create a message with a payload that exceeds the maximum size
	message := &Message{
		Type:     MessageData,
		Sequence: 1,
		Flags:    FlagNone,
		Payload:  make([]byte, MaxMessageSize+1),
	}

	// Try to encode the message
	_, err := message.Encode()
	if err != ErrMessageTooLarge {
		t.Errorf("Expected ErrMessageTooLarge, got %v", err)
	}
}

// TestMessageHelpers tests the helper functions for creating messages.
func TestMessageHelpers(t *testing.T) {
	// Test NewMessage
	t.Run("NewMessage", func(t *testing.T) {
		msg := NewMessage(MessageData, 42, FlagNone, []byte("Payload"))
		if msg.Type != MessageData {
			t.Errorf("Message type = %d, want %d", msg.Type, MessageData)
		}
		if msg.Sequence != 42 {
			t.Errorf("Message sequence = %d, want %d", msg.Sequence, 42)
		}
		if msg.Flags != FlagNone {
			t.Errorf("Message flags = %d, want %d", msg.Flags, FlagNone)
		}
		if !bytes.Equal(msg.Payload, []byte("Payload")) {
			t.Errorf("Message payload doesn't match")
		}
	})

	// Test NewDataMessage
	t.Run("NewDataMessage", func(t *testing.T) {
		msg := NewDataMessage(42, FlagNone, []byte("Data"))
		if msg.Type != MessageData {
			t.Errorf("Message type = %d, want %d", msg.Type, MessageData)
		}
		if msg.Sequence != 42 {
			t.Errorf("Message sequence = %d, want %d", msg.Sequence, 42)
		}
		if msg.Flags != FlagNone {
			t.Errorf("Message flags = %d, want %d", msg.Flags, FlagNone)
		}
		if !bytes.Equal(msg.Payload, []byte("Data")) {
			t.Errorf("Message payload doesn't match")
		}
	})

	// Test NewHandshakeMessage
	t.Run("NewHandshakeMessage", func(t *testing.T) {
		msg := NewHandshakeMessage([]byte{1, 2, 3, 4})
		if msg.Type != MessageHandshake {
			t.Errorf("Message type = %d, want %d", msg.Type, MessageHandshake)
		}
		if msg.Sequence != 0 {
			t.Errorf("Message sequence = %d, want %d", msg.Sequence, 0)
		}
		if msg.Flags != FlagNone {
			t.Errorf("Message flags = %d, want %d", msg.Flags, FlagNone)
		}
		if !bytes.Equal(msg.Payload, []byte{1, 2, 3, 4}) {
			t.Errorf("Message payload doesn't match")
		}
	})

	// Test NewPingMessage
	t.Run("NewPingMessage", func(t *testing.T) {
		msg := NewPingMessage(100)
		if msg.Type != MessagePing {
			t.Errorf("Message type = %d, want %d", msg.Type, MessagePing)
		}
		if msg.Sequence != 100 {
			t.Errorf("Message sequence = %d, want %d", msg.Sequence, 100)
		}
		if msg.Flags != FlagNone {
			t.Errorf("Message flags = %d, want %d", msg.Flags, FlagNone)
		}
		if len(msg.Payload) != 0 {
			t.Errorf("Message payload should be empty")
		}
	})

	// Test NewPongMessage
	t.Run("NewPongMessage", func(t *testing.T) {
		msg := NewPongMessage(100)
		if msg.Type != MessagePong {
			t.Errorf("Message type = %d, want %d", msg.Type, MessagePong)
		}
		if msg.Sequence != 100 {
			t.Errorf("Message sequence = %d, want %d", msg.Sequence, 100)
		}
		if msg.Flags != FlagNone {
			t.Errorf("Message flags = %d, want %d", msg.Flags, FlagNone)
		}
		if len(msg.Payload) != 0 {
			t.Errorf("Message payload should be empty")
		}
	})

	// Test NewCloseMessage
	t.Run("NewCloseMessage", func(t *testing.T) {
		msg := NewCloseMessage()
		if msg.Type != MessageClose {
			t.Errorf("Message type = %d, want %d", msg.Type, MessageClose)
		}
		if msg.Sequence != 0 {
			t.Errorf("Message sequence = %d, want %d", msg.Sequence, 0)
		}
		if msg.Flags != FlagNone {
			t.Errorf("Message flags = %d, want %d", msg.Flags, FlagNone)
		}
		if len(msg.Payload) != 0 {
			t.Errorf("Message payload should be empty")
		}
	})
}
