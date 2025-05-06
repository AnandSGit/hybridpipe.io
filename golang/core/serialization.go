package core

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
)

// SerializationFormat represents the format used for serialization.
type SerializationFormat byte

const (
	// FormatGOB is Go's native binary format (default for backward compatibility).
	FormatGOB SerializationFormat = iota
	// FormatJSON is a text-based, human-readable format.
	FormatJSON
	// FormatProtobuf is a binary format with schema.
	FormatProtobuf
	// FormatMessagePack is a binary JSON alternative.
	FormatMessagePack
)

// SerializationOptions contains options for serialization.
type SerializationOptions struct {
	// Format specifies the serialization format to use.
	Format SerializationFormat
	// Compression specifies whether to compress the data.
	Compression bool
	// ProtobufOptions contains Protocol Buffers specific options.
	ProtobufOptions interface{}
}

// DefaultSerializationOptions returns the default serialization options.
func DefaultSerializationOptions() *SerializationOptions {
	return &SerializationOptions{
		Format:      FormatGOB,
		Compression: false,
	}
}

// messageHeader is the header prepended to serialized messages.
type messageHeader struct {
	// Version of the message format.
	Version byte
	// Format used for serialization.
	Format SerializationFormat
	// Flags for additional options (e.g., compression).
	Flags byte
}

// Encode converts user-defined data into a byte stream using the default options.
func Encode(data any) ([]byte, error) {
	return EncodeWithOptions(data, DefaultSerializationOptions())
}

// EncodeWithOptions converts user-defined data into a byte stream using the specified options.
func EncodeWithOptions(data any, options *SerializationOptions) ([]byte, error) {
	// Create a buffer for the header
	headerBuf := new(bytes.Buffer)

	// Create the header
	header := messageHeader{
		Version: 1,
		Format:  options.Format,
		Flags:   0,
	}

	if options.Compression {
		header.Flags |= 1 // Set compression flag
	}

	// Write the header to the buffer
	if err := binary.Write(headerBuf, binary.LittleEndian, header); err != nil {
		return nil, fmt.Errorf("failed to write message header: %w", err)
	}

	// Serialize the data based on the format
	var dataBuf []byte
	var err error

	switch options.Format {
	case FormatGOB:
		dataBuf, err = encodeGob(data)
	case FormatJSON:
		dataBuf, err = encodeJSON(data)
	case FormatProtobuf:
		return nil, errors.New("protobuf serialization must be handled by the protobuf package")
	case FormatMessagePack:
		return nil, errors.New("messagepack serialization must be handled by the messagepack package")
	default:
		return nil, fmt.Errorf("unsupported serialization format: %d", options.Format)
	}

	if err != nil {
		return nil, err
	}

	// Combine the header and data
	result := append(headerBuf.Bytes(), dataBuf...)

	// Apply compression if requested
	if options.Compression {
		// Compression implementation would go here
		// For now, we'll just return an error
		return nil, errors.New("compression not yet implemented")
	}

	return result, nil
}

// Decode converts a byte stream back into user-defined data using automatic format detection.
func Decode(data []byte, target any) error {
	// Check if the data has a header
	if len(data) < 3 {
		// No header, assume it's legacy GOB data for backward compatibility
		return decodeGob(data, target)
	}

	// Read the header
	headerBuf := bytes.NewReader(data[:3])
	var header messageHeader
	if err := binary.Read(headerBuf, binary.LittleEndian, &header); err != nil {
		// Error reading header, assume it's legacy GOB data
		return decodeGob(data, target)
	}

	// Check the version
	if header.Version != 1 {
		return fmt.Errorf("unsupported message version: %d", header.Version)
	}

	// Check if compression is enabled
	if header.Flags&1 != 0 {
		// Decompression implementation would go here
		// For now, we'll just return an error
		return errors.New("decompression not yet implemented")
	}

	// Decode the data based on the format
	switch header.Format {
	case FormatGOB:
		return decodeGob(data[3:], target)
	case FormatJSON:
		return decodeJSON(data[3:], target)
	case FormatProtobuf:
		return errors.New("protobuf deserialization must be handled by the protobuf package")
	case FormatMessagePack:
		return errors.New("messagepack deserialization must be handled by the messagepack package")
	default:
		return fmt.Errorf("unsupported serialization format: %d", header.Format)
	}
}

// DecodeWithOptions converts a byte stream back into user-defined data using the specified options.
func DecodeWithOptions(data []byte, target any, options *SerializationOptions) error {
	// For now, we'll just call Decode which automatically detects the format
	return Decode(data, target)
}

// encodeGob encodes data using Go's gob encoder.
func encodeGob(data any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return nil, fmt.Errorf("gob encoding failed: %w", err)
	}
	return buf.Bytes(), nil
}

// decodeGob decodes data using Go's gob decoder.
func decodeGob(data []byte, target any) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(target); err != nil {
		return fmt.Errorf("gob decoding failed: %w", err)
	}
	return nil
}

// encodeJSON encodes data using JSON.
func encodeJSON(data any) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed: %w", err)
	}
	return jsonData, nil
}

// decodeJSON decodes data using JSON.
func decodeJSON(data []byte, target any) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("json decoding failed: %w", err)
	}
	return nil
}
