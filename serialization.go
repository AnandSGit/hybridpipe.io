package hybridpipe

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

// SerializationOptions contains options for serialization.
type SerializationOptions struct {
	// Format is the serialization format to use.
	Format SerializationFormat
	// Compression enables gzip compression for large messages.
	Compression bool
	// CompressionLevel sets the compression level (1-9, higher = better compression but slower).
	CompressionLevel int
	// ProtobufMessageType is the fully qualified name of the protobuf message type.
	// Only needed when using FormatProtobuf.
	ProtobufMessageType string
	// TypeRegistry is the registry to use for type conversions.
	// If nil, the global registry will be used.
	TypeRegistry *TypeRegistry
	// ProtobufRegistry is the registry to use for Protocol Buffer messages.
	// If nil, the global registry will be used.
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

// Message is the wire format for HybridPipe messages.
// It includes a header with format information and the serialized payload.
type Message struct {
	// Format is the serialization format used.
	Format SerializationFormat
	// Flags contains additional information about the message.
	Flags byte
	// Payload is the serialized data.
	Payload []byte
}

const (
	// FlagCompressed indicates the payload is compressed.
	FlagCompressed byte = 1 << 0
)

// MessageHeader is the fixed-size header for serialized messages.
// Format: [magic (2 bytes)][version (1 byte)][format (1 byte)][flags (1 byte)][payload length (4 bytes)]
type MessageHeader struct {
	// Magic is a fixed value to identify HybridPipe messages.
	Magic uint16
	// Version is the protocol version.
	Version byte
	// Format is the serialization format.
	Format SerializationFormat
	// Flags contains additional information about the message.
	Flags byte
	// PayloadLength is the length of the payload in bytes.
	PayloadLength uint32
}

const (
	// MessageMagic is the magic number for HybridPipe messages.
	MessageMagic uint16 = 0x4850 // "HP" in ASCII
	// CurrentVersion is the current protocol version.
	CurrentVersion byte = 1
	// HeaderSize is the size of the message header in bytes.
	HeaderSize = 9
)

// EncodeWithOptions serializes data using the specified options.
func EncodeWithOptions(data any, options SerializationOptions) ([]byte, error) {
	var payload []byte
	var err error

	// Use the specified type registry or the global one
	typeRegistry := options.TypeRegistry
	if typeRegistry == nil {
		typeRegistry = globalTypeRegistry
	}

	// Apply type conversion if needed
	if typeRegistry != nil {
		data, err = typeRegistry.ConvertForSerialization(data)
		if err != nil {
			return nil, fmt.Errorf("type conversion error: %w", err)
		}
	}

	// Serialize the data based on the format
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

	// Apply compression if enabled and beneficial
	flags := byte(0)
	if options.Compression && len(payload) > 64 {
		compressed, err := compressData(payload, options.CompressionLevel)
		if err == nil && len(compressed) < len(payload) {
			payload = compressed
			flags |= FlagCompressed
		}
	}

	// Create the message header
	header := MessageHeader{
		Magic:         MessageMagic,
		Version:       CurrentVersion,
		Format:        options.Format,
		Flags:         flags,
		PayloadLength: uint32(len(payload)),
	}

	// Allocate buffer for the complete message
	message := make([]byte, HeaderSize+len(payload))

	// Write the header
	binary.BigEndian.PutUint16(message[0:2], header.Magic)
	message[2] = header.Version
	message[3] = byte(header.Format)
	message[4] = header.Flags
	binary.BigEndian.PutUint32(message[5:9], header.PayloadLength)

	// Copy the payload
	copy(message[HeaderSize:], payload)

	return message, nil
}

// DecodeWithOptions deserializes data using information from the message.
// If options are provided, they will be used; otherwise, default options will be used.
func DecodeWithOptions(data []byte, target any, options ...SerializationOptions) error {
	// Use default options if none provided
	var opts SerializationOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = DefaultSerializationOptions()
	}

	// Use the specified type registry or the global one
	typeRegistry := opts.TypeRegistry
	if typeRegistry == nil {
		typeRegistry = globalTypeRegistry
	}

	// Check if the data has our header format
	if len(data) < HeaderSize {
		// Try legacy format (pure gob) for backward compatibility
		return legacyDecode(data, target)
	}

	// Read the header
	magic := binary.BigEndian.Uint16(data[0:2])
	version := data[2]
	format := SerializationFormat(data[3])
	flags := data[4]
	payloadLength := binary.BigEndian.Uint32(data[5:9])

	// Validate the header
	if magic != MessageMagic {
		// Try legacy format (pure gob) for backward compatibility
		return legacyDecode(data, target)
	}

	// Check version compatibility
	if version > CurrentVersion {
		return fmt.Errorf("unsupported protocol version: %d", version)
	}

	// Extract the payload
	if HeaderSize+int(payloadLength) > len(data) {
		return fmt.Errorf("invalid payload length: %d", payloadLength)
	}
	payload := data[HeaderSize : HeaderSize+int(payloadLength)]

	// Decompress if needed
	if flags&FlagCompressed != 0 {
		var err error
		payload, err = decompressData(payload)
		if err != nil {
			return fmt.Errorf("decompression error: %w", err)
		}
	}

	// Deserialize based on the format
	var result interface{}
	var err error

	switch format {
	case FormatGob:
		err = decodeGob(payload, target)
	case FormatJSON:
		err = decodeJSON(payload, target)
	case FormatProtobuf:
		// Use the enhanced Protocol Buffers decoder if a message type is specified
		if opts.ProtobufMessageType != "" {
			protobufOptions := &ProtobufSerializationOptions{
				MessageTypeName: opts.ProtobufMessageType,
				DynamicMessage:  true,
			}
			err = decodeProtobufEnhanced(payload, target, protobufOptions)
		} else {
			err = decodeProtobuf(payload, target)
		}
	case FormatMsgPack:
		err = decodeMsgPack(payload, target)
	default:
		return fmt.Errorf("unsupported serialization format: %d", format)
	}

	if err != nil {
		return fmt.Errorf("deserialization error: %w", err)
	}

	// Apply type conversion if needed
	if typeRegistry != nil {
		converted, err := typeRegistry.ConvertForDeserialization(result)
		if err != nil {
			return fmt.Errorf("type conversion error: %w", err)
		}

		// Set the converted value to the target
		if converted != nil {
			// TODO: Set the converted value to the target
			// This requires reflection and is complex
		}
	}

	return nil
}

// legacyDecode handles decoding of legacy format (pure gob) for backward compatibility.
func legacyDecode(data []byte, target any) error {
	return decodeGob(data, target)
}

// These functions are defined in hybridpipe.go:
// - Encode(data any) ([]byte, error)
// - Decode(data []byte, target any) error
// - Enable(dataType any)

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
	reader := bytes.NewReader(data)
	decoder := gob.NewDecoder(reader)

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
	if msg, ok := data.(proto.Message); ok {
		return proto.Marshal(msg)
	}
	return nil, fmt.Errorf("data is not a Protocol Buffer message")
}

func decodeProtobuf(data []byte, target any) error {
	if msg, ok := target.(proto.Message); ok {
		return proto.Unmarshal(data, msg)
	}
	return fmt.Errorf("target is not a Protocol Buffer message")
}

func encodeMsgPack(data any) ([]byte, error) {
	return msgpack.Marshal(data)
}

func decodeMsgPack(data []byte, target any) error {
	return msgpack.Unmarshal(data, target)
}

// compressData compresses data using gzip.
func compressData(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressData decompresses gzip-compressed data.
func decompressData(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var result bytes.Buffer
	if _, err := io.Copy(&result, r); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

