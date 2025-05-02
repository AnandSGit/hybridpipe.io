package hybridpipe

import (
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ProtobufRegistry manages Protocol Buffer message types.
type ProtobufRegistry struct {
	// messageTypes maps from type name to message type
	messageTypes map[string]protoreflect.MessageType
	// messageFactories maps from type name to a function that creates a new message
	messageFactories map[string]func() proto.Message
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
}

// globalProtobufRegistry is the global instance of ProtobufRegistry
var globalProtobufRegistry = NewProtobufRegistry()

// NewProtobufRegistry creates a new Protocol Buffer registry.
func NewProtobufRegistry() *ProtobufRegistry {
	return &ProtobufRegistry{
		messageTypes:     make(map[string]protoreflect.MessageType),
		messageFactories: make(map[string]func() proto.Message),
	}
}

// RegisterMessageType registers a Protocol Buffer message type.
func (r *ProtobufRegistry) RegisterMessageType(message proto.Message) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Get the message descriptor
	descriptor := message.ProtoReflect().Descriptor()
	fullName := string(descriptor.FullName())

	// Store the message type
	r.messageTypes[fullName] = dynamicpb.NewMessageType(descriptor)

	// Create a factory function
	r.messageFactories[fullName] = func() proto.Message {
		return proto.Clone(message)
	}
}

// GetMessageType returns the message type for the given name.
func (r *ProtobufRegistry) GetMessageType(name string) (protoreflect.MessageType, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	messageType, exists := r.messageTypes[name]
	return messageType, exists
}

// CreateMessage creates a new instance of the message with the given name.
func (r *ProtobufRegistry) CreateMessage(name string) (proto.Message, error) {
	r.mutex.RLock()
	factory, exists := r.messageFactories[name]
	r.mutex.RUnlock()

	if !exists {
		// Try to find the message type in the global registry
		messageType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(name))
		if err != nil {
			return nil, fmt.Errorf("unknown message type: %s", name)
		}

		// Create a new message
		message := dynamicpb.NewMessage(messageType.Descriptor())
		return message, nil
	}

	return factory(), nil
}

// RegisterMessageType registers a Protocol Buffer message type in the global registry.
func RegisterMessageType(message proto.Message) {
	globalProtobufRegistry.RegisterMessageType(message)
}

// GetMessageType returns the message type for the given name from the global registry.
func GetMessageType(name string) (protoreflect.MessageType, bool) {
	return globalProtobufRegistry.GetMessageType(name)
}

// CreateMessage creates a new instance of the message with the given name from the global registry.
func CreateMessage(name string) (proto.Message, error) {
	return globalProtobufRegistry.CreateMessage(name)
}

// ProtobufSerializationOptions contains options specific to Protocol Buffers serialization.
type ProtobufSerializationOptions struct {
	// MessageTypeName is the fully qualified name of the Protocol Buffer message type.
	MessageTypeName string
	// DynamicMessage indicates whether to use dynamic message handling.
	DynamicMessage bool
}

// encodeProtobufEnhanced is an enhanced version of encodeProtobuf that supports dynamic messages.
func encodeProtobufEnhanced(data interface{}, options *ProtobufSerializationOptions) ([]byte, error) {
	// If data is already a Protocol Buffer message, use it directly
	if msg, ok := data.(proto.Message); ok {
		return proto.Marshal(msg)
	}

	// If options specify a message type, try to convert the data
	if options != nil && options.MessageTypeName != "" {
		// Create a new message of the specified type
		msg, err := CreateMessage(options.MessageTypeName)
		if err != nil {
			return nil, fmt.Errorf("failed to create message of type %s: %w", options.MessageTypeName, err)
		}

		// Convert the data to the message
		if err := convertToProtobuf(data, msg); err != nil {
			return nil, fmt.Errorf("failed to convert data to Protocol Buffer message: %w", err)
		}

		// Marshal the message
		return proto.Marshal(msg)
	}

	return nil, fmt.Errorf("data is not a Protocol Buffer message and no message type specified")
}

// decodeProtobufEnhanced is an enhanced version of decodeProtobuf that supports dynamic messages.
func decodeProtobufEnhanced(data []byte, target interface{}, options *ProtobufSerializationOptions) error {
	// If target is already a Protocol Buffer message, use it directly
	if msg, ok := target.(proto.Message); ok {
		return proto.Unmarshal(data, msg)
	}

	// If options specify a message type, try to convert the data
	if options != nil && options.MessageTypeName != "" {
		// Create a new message of the specified type
		msg, err := CreateMessage(options.MessageTypeName)
		if err != nil {
			return fmt.Errorf("failed to create message of type %s: %w", options.MessageTypeName, err)
		}

		// Unmarshal the data into the message
		if err := proto.Unmarshal(data, msg); err != nil {
			return fmt.Errorf("failed to unmarshal Protocol Buffer message: %w", err)
		}

		// Convert the message to the target
		if err := convertFromProtobuf(msg, target); err != nil {
			return fmt.Errorf("failed to convert Protocol Buffer message to target: %w", err)
		}

		return nil
	}

	return fmt.Errorf("target is not a Protocol Buffer message and no message type specified")
}

// convertToProtobuf converts a Go struct to a Protocol Buffer message.
func convertToProtobuf(source interface{}, target proto.Message) error {
	// TODO: Implement conversion from Go struct to Protocol Buffer message
	// This is a complex task that requires reflection and field mapping
	return fmt.Errorf("not implemented")
}

// convertFromProtobuf converts a Protocol Buffer message to a Go struct.
func convertFromProtobuf(source proto.Message, target interface{}) error {
	// TODO: Implement conversion from Protocol Buffer message to Go struct
	// This is a complex task that requires reflection and field mapping
	return fmt.Errorf("not implemented")
}
