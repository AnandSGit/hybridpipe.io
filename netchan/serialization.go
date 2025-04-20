package netchan

import (
	"bytes"
	"compress/gzip"
	"encoding"
	"encoding/gob"
	"fmt"
	"io"
	"reflect"
)

// init registers common types with gob.
func init() {
	// Register common types
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register(TypeInfo{})
}

// RegisterType registers a type with the serialization system.
func RegisterType(value interface{}) {
	gob.Register(value)
}

// Serializer handles serialization and deserialization of values.
type Serializer struct {
	// Compression settings
	compression      bool
	compressionLevel int
}

// NewSerializer creates a new serializer.
func NewSerializer(compression bool, compressionLevel int) *Serializer {
	return &Serializer{
		compression:      compression,
		compressionLevel: compressionLevel,
	}
}

// Serialize serializes a value.
func (s *Serializer) Serialize(value interface{}) ([]byte, byte, error) {
	// Check for nil
	if value == nil {
		return nil, FlagNone, nil
	}

	// Check for BinaryMarshaler interface
	if marshaler, ok := value.(encoding.BinaryMarshaler); ok {
		data, err := marshaler.MarshalBinary()
		if err != nil {
			return nil, FlagNone, fmt.Errorf("netchan: binary marshal error: %w", err)
		}
		return data, FlagNone, nil
	}

	// Use gob encoding
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, FlagNone, fmt.Errorf("netchan: gob encode error: %w", err)
	}

	data := buf.Bytes()
	flags := FlagNone

	// Apply compression if enabled and beneficial
	if s.compression && len(data) > 64 {
		compressed, err := s.compress(data)
		if err != nil {
			// Just use uncompressed data
			return data, flags, nil
		}

		// Only use compression if it actually reduces the size
		if len(compressed) < len(data) {
			data = compressed
			flags |= FlagCompressed
		}
	}

	return data, flags, nil
}

// Deserialize deserializes a value.
func (s *Serializer) Deserialize(data []byte, flags byte, valueType reflect.Type) (interface{}, error) {
	// Handle nil
	if len(data) == 0 {
		return nil, nil
	}

	// Decompress if needed
	if flags&FlagCompressed != 0 {
		var err error
		data, err = s.decompress(data)
		if err != nil {
			return nil, fmt.Errorf("netchan: decompress error: %w", err)
		}
	}

	// Create a new value of the target type
	value := reflect.New(valueType).Interface()

	// Check for BinaryUnmarshaler interface
	if unmarshaler, ok := value.(encoding.BinaryUnmarshaler); ok {
		if err := unmarshaler.UnmarshalBinary(data); err != nil {
			return nil, fmt.Errorf("netchan: binary unmarshal error: %w", err)
		}
		return reflect.ValueOf(value).Elem().Interface(), nil
	}

	// Use gob decoding
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(value); err != nil {
		return nil, fmt.Errorf("netchan: gob decode error: %w", err)
	}

	return reflect.ValueOf(value).Elem().Interface(), nil
}

// compress compresses data using gzip.
func (s *Serializer) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, s.compressionLevel)
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

// decompress decompresses gzip-compressed data.
func (s *Serializer) decompress(data []byte) ([]byte, error) {
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
