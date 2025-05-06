// Package serialization provides functionality for encoding and decoding data
// in various formats for transmission through the HybridPipe messaging system.
package serialization

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"

	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

// SerializationFormat represents the format used for serialization.
type SerializationFormat byte

const (
	// FormatGob is Go's native gob encoding (default for backward compatibility)
	FormatGob SerializationFormat = iota
	// FormatJSON is standard JSON encoding (language agnostic but less efficient)
	FormatJSON
	// FormatProtobuf is Protocol Buffers encoding (efficient and language agnostic)
	FormatProtobuf
	// FormatMsgPack is MessagePack encoding (efficient JSON alternative)
	FormatMsgPack
)

// HeaderSize is the size of the serialization header in bytes.
const HeaderSize = 9

// SerializationOptions contains options for serialization.
type SerializationOptions struct {
	// Format is the serialization format to use.
	Format SerializationFormat
	// Compression indicates whether to compress the data.
	Compression bool
	// CompressionLevel is the compression level to use (0-9, where 0 is no compression
	// and 9 is maximum compression).
	CompressionLevel int
	// ProtobufMessageType is the fully qualified name of the Protocol Buffer message type.
	// This is only used when Format is FormatProtobuf.
	ProtobufMessageType string
	// TypeRegistry is the registry of type mappings for serialization.
	TypeRegistry *TypeRegistry
	// ProtobufRegistry is the registry of Protocol Buffer message types.
	ProtobufRegistry *ProtobufRegistry
	// CrossLanguageCompatible indicates whether to use cross-language compatible serialization.
	// This may limit the types that can be serialized.
	CrossLanguageCompatible bool
}

// DefaultSerializationOptions returns the default serialization options.
func DefaultSerializationOptions() SerializationOptions {
	return SerializationOptions{
		Format:                  FormatGob, // Default to gob for backward compatibility
		Compression:             false,
		CompressionLevel:        6, // Default compression level
		TypeRegistry:            globalTypeRegistry,
		ProtobufRegistry:        globalProtobufRegistry,
		CrossLanguageCompatible: false,
	}
}

// Encode converts user-defined data (to be transmitted) into a byte stream.
// It uses the default serialization format (gob for backward compatibility).
func Encode(data any) ([]byte, error) {
	// Use the default serialization options (gob for backward compatibility)
	return EncodeWithOptions(data, DefaultSerializationOptions())
}

// EncodeWithOptions converts user-defined data into a byte stream using the specified options.
func EncodeWithOptions(data any, options SerializationOptions) ([]byte, error) {
	// Apply type conversions if needed
	if options.TypeRegistry != nil {
		var err error
		data, err = options.TypeRegistry.ConvertForSerialization(data)
		if err != nil {
			return nil, fmt.Errorf("type conversion error: %w", err)
		}
	}

	// Serialize the data based on the format
	var payload []byte
	var err error
	switch options.Format {
	case FormatGob:
		payload, err = encodeGob(data)
	case FormatJSON:
		payload, err = encodeJSON(data)
	case FormatProtobuf:
		// Use the enhanced Protocol Buffers encoder if a message type is specified
		if options.ProtobufMessageType != "" {
			protobufOptions := &ProtobufSerializationOptions{
				MessageTypeName: options.ProtobufMessageType,
				DynamicMessage:  true,
			}
			payload, err = encodeProtobufEnhanced(data, protobufOptions)
		} else {
			payload, err = encodeProtobuf(data)
		}
	case FormatMsgPack:
		payload, err = encodeMsgPack(data)
	default:
		return nil, fmt.Errorf("unsupported serialization format: %d", options.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("serialization error: %w", err)
	}

	// Compress the payload if requested
	if options.Compression && len(payload) > 64 {
		compressed, err := compressData(payload, options.CompressionLevel)
		if err != nil {
			return nil, fmt.Errorf("compression error: %w", err)
		}

		// Only use compression if it actually reduces the size
		if len(compressed) < len(payload) {
			payload = compressed
			// Set the compression flag in the header
			options.Format |= 0x80
		}
	}

	// Create the header
	header := make([]byte, HeaderSize)
	// Magic number (HP)
	header[0] = 'H'
	header[1] = 'P'
	// Version
	header[2] = 1
	// Format (with compression flag)
	header[3] = byte(options.Format)
	// Payload length
	binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))
	// Checksum (simple for now)
	header[8] = calculateChecksum(payload)

	// Combine header and payload
	result := make([]byte, HeaderSize+len(payload))
	copy(result[:HeaderSize], header)
	copy(result[HeaderSize:], payload)

	return result, nil
}

// Decode converts a received byte stream back into user-defined interface data.
// It automatically detects the serialization format used.
func Decode(data []byte, target any) error {
	// Use DecodeWithOptions which will automatically detect the format
	return DecodeWithOptions(data, target, DefaultSerializationOptions())
}

// DecodeWithOptions converts a received byte stream back into user-defined interface data
// using the specified options.
func DecodeWithOptions(data []byte, target any, options SerializationOptions) error {
	// Check if the data is valid
	if len(data) < HeaderSize {
		return fmt.Errorf("data too short to contain header")
	}

	// Validate the header
	if data[0] != 'H' || data[1] != 'P' {
		return fmt.Errorf("invalid magic number in header")
	}

	// Get the format from the header
	format := SerializationFormat(data[3] & 0x7F) // Remove compression flag
	compressed := (data[3] & 0x80) != 0           // Check compression flag

	// Get the payload length from the header
	payloadLen := binary.BigEndian.Uint32(data[4:8])
	if uint32(len(data)-HeaderSize) != payloadLen {
		return fmt.Errorf("payload length mismatch")
	}

	// Validate the checksum
	checksum := calculateChecksum(data[HeaderSize:])
	if checksum != data[8] {
		return fmt.Errorf("checksum mismatch")
	}

	// Extract the payload
	payload := data[HeaderSize:]

	// Decompress the payload if needed
	if compressed {
		decompressed, err := decompressData(payload)
		if err != nil {
			return fmt.Errorf("decompression error: %w", err)
		}
		payload = decompressed
	}

	// Deserialize the payload based on the format
	var err error
	switch format {
	case FormatGob:
		err = decodeGob(payload, target)
	case FormatJSON:
		err = decodeJSON(payload, target)
	case FormatProtobuf:
		err = decodeProtobuf(payload, target)
	case FormatMsgPack:
		err = decodeMsgPack(payload, target)
	default:
		return fmt.Errorf("unsupported serialization format: %d", format)
	}

	if err != nil {
		return fmt.Errorf("deserialization error: %w", err)
	}

	// Apply type conversions if needed
	if options.TypeRegistry != nil {
		err = options.TypeRegistry.ConvertForDeserialization(target)
		if err != nil {
			return fmt.Errorf("type conversion error: %w", err)
		}
	}

	return nil
}

// Enable registers a specific user-defined data type to be passed via HybridPipe.
// This must be called before sending custom data types through the messaging system.
// Note: This is only needed for gob encoding, not for other formats.
func Enable(dataType any) {
	gob.Register(dataType)
}

// RegisterProtobufType registers a Protocol Buffer message type.
// This should be called for each Protocol Buffer message type used.
func RegisterProtobufType(message proto.Message) {
	// Protocol Buffers handles type registration internally
	// This function is provided for API consistency
}

// Helper functions for different serialization formats

func encodeGob(data any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("gob encoding error: %w", err)
	}

	return buffer.Bytes(), nil
}

func decodeGob(data []byte, target any) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("gob decoding error: %w", err)
	}

	return nil
}

func encodeJSON(data any) ([]byte, error) {
	return json.Marshal(data)
}

func decodeJSON(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

func encodeProtobuf(data any) ([]byte, error) {
	message, ok := data.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("data is not a Protocol Buffer message")
	}

	return proto.Marshal(message)
}

func decodeProtobuf(data []byte, target any) error {
	message, ok := target.(proto.Message)
	if !ok {
		return fmt.Errorf("target is not a Protocol Buffer message")
	}

	return proto.Unmarshal(data, message)
}

func encodeMsgPack(data any) ([]byte, error) {
	return msgpack.Marshal(data)
}

func decodeMsgPack(data []byte, target any) error {
	return msgpack.Unmarshal(data, target)
}

func compressData(data []byte, level int) ([]byte, error) {
	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, level)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func decompressData(data []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(data)
	reader, err := gzip.NewReader(buffer)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func calculateChecksum(data []byte) byte {
	var sum byte
	for _, b := range data {
		sum += b
	}
	return sum
}
