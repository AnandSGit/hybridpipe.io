package netchan

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"reflect"
)

// Message represents a NetChan protocol message.
type Message struct {
	Type     MessageType
	Sequence uint32
	Flags    byte
	Payload  []byte
}

// NewMessage creates a new message.
func NewMessage(msgType MessageType, seq uint32, flags byte, payload []byte) *Message {
	return &Message{
		Type:     msgType,
		Sequence: seq,
		Flags:    flags,
		Payload:  payload,
	}
}

// NewDataMessage creates a new data message.
func NewDataMessage(seq uint32, flags byte, value interface{}) (*Message, error) {
	payload, err := encodeValue(value)
	if err != nil {
		return nil, err
	}

	return NewMessage(MessageData, seq, flags, payload), nil
}

// NewHandshakeMessage creates a new handshake message.
func NewHandshakeMessage(seq uint32, typeInfo TypeInfo) (*Message, error) {
	payload, err := encodeValue(typeInfo)
	if err != nil {
		return nil, err
	}

	return NewMessage(MessageHandshake, seq, FlagNone, payload), nil
}

// NewCloseMessage creates a new close message.
func NewCloseMessage(seq uint32) *Message {
	return NewMessage(MessageClose, seq, FlagNone, nil)
}

// NewAckMessage creates a new acknowledgment message.
func NewAckMessage(seq uint32) *Message {
	return NewMessage(MessageAck, seq, FlagNone, nil)
}

// NewPingMessage creates a new ping message.
func NewPingMessage(seq uint32) *Message {
	return NewMessage(MessagePing, seq, FlagNone, nil)
}

// NewPongMessage creates a new pong message.
func NewPongMessage(seq uint32) *Message {
	return NewMessage(MessagePong, seq, FlagNone, nil)
}

// Encode encodes the message to a byte slice.
func (m *Message) Encode() ([]byte, error) {
	payloadLen := len(m.Payload)
	if payloadLen > MaxMessageSize {
		return nil, ErrMessageTooLarge
	}

	// Calculate the total message size
	totalSize := HeaderSize + payloadLen

	// Create a buffer for the message
	buf := make([]byte, totalSize)

	// Write the header
	buf[0] = byte(m.Type)
	binary.BigEndian.PutUint32(buf[1:5], m.Sequence)
	buf[5] = m.Flags
	binary.BigEndian.PutUint32(buf[6:10], uint32(payloadLen))

	// Copy the payload
	if payloadLen > 0 {
		copy(buf[HeaderSize:], m.Payload)
	}

	return buf, nil
}

// DecodeMessage decodes a message from a byte slice.
func DecodeMessage(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("netchan: message too short")
	}

	// Read the header
	msgType := MessageType(data[0])
	sequence := binary.BigEndian.Uint32(data[1:5])
	flags := data[5]
	payloadLen := binary.BigEndian.Uint32(data[6:10])

	// Validate the payload length
	if payloadLen > MaxMessageSize {
		return nil, ErrMessageTooLarge
	}

	if uint32(len(data)-HeaderSize) < payloadLen {
		return nil, fmt.Errorf("netchan: incomplete message")
	}

	// Extract the payload
	var payload []byte
	if payloadLen > 0 {
		payload = make([]byte, payloadLen)
		copy(payload, data[HeaderSize:HeaderSize+payloadLen])
	}

	return &Message{
		Type:     msgType,
		Sequence: sequence,
		Flags:    flags,
		Payload:  payload,
	}, nil
}

// ReadMessage reads a message from a reader.
func ReadMessage(r io.Reader) (*Message, error) {
	// Read the header
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	// Parse the header
	msgType := MessageType(header[0])
	sequence := binary.BigEndian.Uint32(header[1:5])
	flags := header[5]
	payloadLen := binary.BigEndian.Uint32(header[6:10])

	// Validate the payload length
	if payloadLen > MaxMessageSize {
		return nil, ErrMessageTooLarge
	}

	// Read the payload
	var payload []byte
	if payloadLen > 0 {
		payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}

	return &Message{
		Type:     msgType,
		Sequence: sequence,
		Flags:    flags,
		Payload:  payload,
	}, nil
}

// WriteMessage writes a message to a writer.
func WriteMessage(w io.Writer, msg *Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

// TypeInfo contains type information for a channel.
type TypeInfo struct {
	// TypeName is the name of the type.
	TypeName string
	// TypeKind is the kind of the type.
	TypeKind reflect.Kind
	// IsPointer indicates if the type is a pointer.
	IsPointer bool
	// ElementInfo contains type information for element types (for slices, maps, etc.).
	ElementInfo *TypeInfo
	// KeyInfo contains type information for key types (for maps).
	KeyInfo *TypeInfo
	// FieldInfo contains type information for struct fields.
	FieldInfo map[string]TypeInfo
}

// TypeInfoFromType creates type information from a reflect.Type.
func TypeInfoFromType(t reflect.Type) TypeInfo {
	info := TypeInfo{
		TypeName:  t.String(),
		TypeKind:  t.Kind(),
		IsPointer: t.Kind() == reflect.Ptr,
	}

	// Handle pointer types
	if info.IsPointer {
		elemInfo := TypeInfoFromType(t.Elem())
		info.ElementInfo = &elemInfo
		return info
	}

	// Handle container types
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		elemInfo := TypeInfoFromType(t.Elem())
		info.ElementInfo = &elemInfo
	case reflect.Map:
		keyInfo := TypeInfoFromType(t.Key())
		elemInfo := TypeInfoFromType(t.Elem())
		info.KeyInfo = &keyInfo
		info.ElementInfo = &elemInfo
	case reflect.Struct:
		info.FieldInfo = make(map[string]TypeInfo)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath == "" { // Exported field
				info.FieldInfo[field.Name] = TypeInfoFromType(field.Type)
			}
		}
	}

	return info
}

// Compatible checks if this type is compatible with another type.
func (ti TypeInfo) Compatible(other TypeInfo) bool {
	// Check basic compatibility
	if ti.TypeKind != other.TypeKind {
		return false
	}

	// Handle pointer types
	if ti.IsPointer {
		if ti.ElementInfo == nil || other.ElementInfo == nil {
			return false
		}
		return ti.ElementInfo.Compatible(*other.ElementInfo)
	}

	// Handle container types
	switch ti.TypeKind {
	case reflect.Slice, reflect.Array:
		if ti.ElementInfo == nil || other.ElementInfo == nil {
			return false
		}
		return ti.ElementInfo.Compatible(*other.ElementInfo)
	case reflect.Map:
		if ti.KeyInfo == nil || other.KeyInfo == nil || ti.ElementInfo == nil || other.ElementInfo == nil {
			return false
		}
		return ti.KeyInfo.Compatible(*other.KeyInfo) && ti.ElementInfo.Compatible(*other.ElementInfo)
	case reflect.Struct:
		// For structs, we need to check both ways to ensure full compatibility
		// First, check that all fields in this type exist in the other type
		for name, field := range ti.FieldInfo {
			otherField, ok := other.FieldInfo[name]
			if !ok || !field.Compatible(otherField) {
				return false
			}
		}
		// Then, check that all fields in the other type exist in this type
		for name, field := range other.FieldInfo {
			thisField, ok := ti.FieldInfo[name]
			if !ok || !field.Compatible(thisField) {
				return false
			}
		}
		return true
	}

	// For basic types, just check the kind
	return true
}

// encodeValue encodes a value using gob encoding.
func encodeValue(value interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, fmt.Errorf("netchan: encode error: %w", err)
	}
	return buf.Bytes(), nil
}

// decodeValue decodes a value using gob encoding.
func decodeValue(data []byte, value interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(value); err != nil {
		return fmt.Errorf("netchan: decode error: %w", err)
	}
	return nil
}
