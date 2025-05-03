package hybridpipe

import (
	"fmt"
	"sync"
	"reflect"

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

// GetMessageFactory returns the message factory for the given name.
func (r *ProtobufRegistry) GetMessageFactory(name string) (func() proto.Message, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	factory, exists := r.messageFactories[name]
	return factory, exists
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

	// Convert the source to a map
	sourceMap, err := structToMap(source)
	if err != nil {
		return fmt.Errorf("failed to convert source to map: %w", err)
	}

	// Convert the map to a Protocol Buffer message
	return mapToProtobuf(sourceMap, target)
}

func mapToProtobuf(sourceMap map[string]interface{}, target proto.Message) error {
	// Get the message descriptor
	descriptor := target.ProtoReflect().Descriptor()

	// Create a new message
	message := dynamicpb.NewMessage(descriptor)

	// Set the fields
	for name, value := range sourceMap {
		fieldDescriptor := descriptor.Fields().ByName(protoreflect.Name(name))
		if fieldDescriptor == nil {
			continue
		}
	
		// Set the field value
		switch fieldDescriptor.Kind() {
		case protoreflect.StringKind:
			message.Set(fieldDescriptor, protoreflect.ValueOf(value.(string)))
		case protoreflect.Int64Kind:
			message.Set(fieldDescriptor, protoreflect.ValueOf(value.(int64)))
		case protoreflect.Uint64Kind:
			message.Set(fieldDescriptor, protoreflect.ValueOf(value.(uint64)))
		case protoreflect.BoolKind:
			message.Set(fieldDescriptor, protoreflect.ValueOf(value.(bool)))
		case protoreflect.BytesKind:
			message.Set(fieldDescriptor, protoreflect.ValueOf(value.([]byte)))
		case protoreflect.MessageKind:
			// Recursively convert nested messages
			nestedMap := value.(map[string]interface{})
			nestedMessage := dynamicpb.NewMessage(fieldDescriptor.Message())
			if err := mapToProtobuf(nestedMap, nestedMessage); err != nil {
				return err
			}
			message.Set(fieldDescriptor, protoreflect.ValueOf(nestedMessage.ProtoReflect()))
		}
	}

	// Copy the fields to the target message
	proto.Merge(target, message)

	return nil
}

// structToMap converts a struct to a map using reflection.
func structToMap(source interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	// Get the value of the source
	v := reflect.ValueOf(source)
	
	// If source is a pointer, get the value it points to
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	// Check if source is a struct
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("source must be a struct or pointer to struct, got %s", v.Kind())
	}
	
	// Get the type of the source
	t := v.Type()
	
	// Iterate over all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Get the field value
		fieldValue := v.Field(i).Interface()
		
		// Handle nested structs recursively
		if v.Field(i).Kind() == reflect.Struct {
			nestedMap, err := structToMap(fieldValue)
			if err != nil {
				return nil, err
			}
			result[field.Name] = nestedMap
		} else if v.Field(i).Kind() == reflect.Ptr && v.Field(i).Elem().Kind() == reflect.Struct {
			// Handle pointers to structs
			if !v.Field(i).IsNil() {
				nestedMap, err := structToMap(fieldValue)
				if err != nil {
					return nil, err
				}
				result[field.Name] = nestedMap
			}
		} else {
			// Add the field to the map
			result[field.Name] = fieldValue
		}
	}
	
	return result, nil
}

// convertFromProtobuf converts a Protocol Buffer message to a Go struct.
func convertFromProtobuf(source proto.Message, target interface{}) error {
	// Get the target value
	targetValue := reflect.ValueOf(target)
	
	// Check if target is a pointer
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer, got %s", targetValue.Kind())
	}
	
	// Get the element that the pointer points to
	targetElem := targetValue.Elem()
	
	// Check if target is a struct
	if targetElem.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct, got pointer to %s", targetElem.Kind())
	}
	
	// Convert the Protocol Buffer message to a map
	sourceMap, err := protoToMap(source)
	if err != nil {
		return fmt.Errorf("failed to convert Protocol Buffer message to map: %w", err)
	}
	
	// Convert the map to the target struct
	return mapToStruct(sourceMap, targetElem)
}

// protoToMap converts a Protocol Buffer message to a map.
func protoToMap(message proto.Message) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	// Get the message reflection
	msg := message.ProtoReflect()
	
	// Iterate over all fields
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		// Get the field name
		name := string(fd.Name())
		
		// Convert the value based on the field type
		switch fd.Kind() {
		case protoreflect.StringKind:
			result[name] = v.String()
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			result[name] = v.Int()
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			result[name] = v.Uint()
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			result[name] = int32(v.Int())
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			result[name] = uint32(v.Uint())
		case protoreflect.BoolKind:
			result[name] = v.Bool()
		case protoreflect.BytesKind:
			result[name] = v.Bytes()
		case protoreflect.FloatKind:
			result[name] = float32(v.Float())
		case protoreflect.DoubleKind:
			result[name] = v.Float()
		case protoreflect.MessageKind:
			// Recursively convert nested messages
			nestedMsg := v.Message().Interface()
			nestedMap, err := protoToMap(nestedMsg)
			if err != nil {
				return false
			}
			result[name] = nestedMap
		}
		return true
	})
	
	return result, nil
}

// mapToStruct converts a map to a struct using reflection.
func mapToStruct(sourceMap map[string]interface{}, target reflect.Value) error {
	// Get the type of the target
	targetType := target.Type()
	
	// Iterate over all fields in the target struct
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Get the field name
		fieldName := field.Name
		
		// Check if the field exists in the source map
		value, exists := sourceMap[fieldName]
		if !exists {
			continue
		}
		
		// Get the field value
		fieldValue := target.Field(i)
		
		// Set the field value based on its type
		if err := setFieldValue(fieldValue, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldName, err)
		}
	}
	
	return nil
}

// setFieldValue sets a field value using reflection.
func setFieldValue(field reflect.Value, value interface{}) error {
	// Check if the field is settable
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}
	
	// Handle different field types
	switch field.Kind() {
	case reflect.String:
		if strValue, ok := value.(string); ok {
			field.SetString(strValue)
		} else {
			return fmt.Errorf("expected string, got %T", value)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intValue int64
		switch v := value.(type) {
		case int:
			intValue = int64(v)
		case int8:
			intValue = int64(v)
		case int16:
			intValue = int64(v)
		case int32:
			intValue = int64(v)
		case int64:
			intValue = v
		default:
			return fmt.Errorf("expected integer, got %T", value)
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintValue uint64
		switch v := value.(type) {
		case uint:
			uintValue = uint64(v)
		case uint8:
			uintValue = uint64(v)
		case uint16:
			uintValue = uint64(v)
		case uint32:
			uintValue = uint64(v)
		case uint64:
			uintValue = v
		default:
			return fmt.Errorf("expected unsigned integer, got %T", value)
		}
		field.SetUint(uintValue)
	case reflect.Bool:
		if boolValue, ok := value.(bool); ok {
			field.SetBool(boolValue)
		} else {
			return fmt.Errorf("expected bool, got %T", value)
		}
	case reflect.Float32, reflect.Float64:
		var floatValue float64
		switch v := value.(type) {
		case float32:
			floatValue = float64(v)
		case float64:
			floatValue = v
		default:
			return fmt.Errorf("expected float, got %T", value)
		}
		field.SetFloat(floatValue)
	case reflect.Struct:
		if mapValue, ok := value.(map[string]interface{}); ok {
			return mapToStruct(mapValue, field)
		} else {
			return fmt.Errorf("expected map for struct field, got %T", value)
		}
	case reflect.Ptr:
		if field.IsNil() {
			// Create a new instance of the pointed-to type
			field.Set(reflect.New(field.Type().Elem()))
		}
		// Handle the pointed-to value
		if mapValue, ok := value.(map[string]interface{}); ok && field.Elem().Kind() == reflect.Struct {
			return mapToStruct(mapValue, field.Elem())
		} else {
			return fmt.Errorf("expected map for pointer to struct field, got %T", value)
		}
	}
	
	return nil
}



