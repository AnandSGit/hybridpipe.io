package serialization

import (
	"fmt"
	"reflect"
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

// RegisterMessageType registers a Protocol Buffer message type with the registry.
func (r *ProtobufRegistry) RegisterMessageType(message proto.Message, factory func() proto.Message) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Get the message descriptor
	descriptor := message.ProtoReflect().Descriptor()
	fullName := string(descriptor.FullName())

	// Register the message type
	r.messageTypes[fullName] = dynamicpb.NewMessageType(descriptor)
	r.messageFactories[fullName] = factory
}

// GetMessageType returns the message type for the specified name.
func (r *ProtobufRegistry) GetMessageType(name string) (protoreflect.MessageType, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	messageType, exists := r.messageTypes[name]
	return messageType, exists
}

// GetMessageFactory returns the message factory for the specified name.
func (r *ProtobufRegistry) GetMessageFactory(name string) (func() proto.Message, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	factory, exists := r.messageFactories[name]
	return factory, exists
}

// CreateMessage creates a new message of the specified type.
func CreateMessage(typeName string) (proto.Message, error) {
	// Try to get the message factory from the registry
	factory, exists := globalProtobufRegistry.GetMessageFactory(typeName)
	if exists {
		return factory(), nil
	}

	// Try to find the message type in the global registry
	messageType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typeName))
	if err != nil {
		return nil, fmt.Errorf("failed to find message type %s: %w", typeName, err)
	}

	// Create a new message of the specified type
	return dynamicpb.NewMessage(messageType.Descriptor()), nil
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
		// Find the field descriptor
		field := descriptor.Fields().ByName(protoreflect.Name(name))
		if field == nil {
			// Skip fields that don't exist in the message
			continue
		}

		// Convert the value to the appropriate type
		protoValue, err := convertValueToProto(value, field)
		if err != nil {
			return fmt.Errorf("failed to convert field %s: %w", name, err)
		}

		// Set the field value
		message.Set(field, protoValue)
	}

	// Copy the message to the target
	proto.Reset(target)
	proto.Merge(target, message)

	return nil
}

// convertValueToProto converts a Go value to a Protocol Buffer value.
func convertValueToProto(value interface{}, field protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	// Handle nil values
	if value == nil {
		return protoreflect.Value{}, nil
	}

	// Handle different field types
	switch field.Kind() {
	case protoreflect.BoolKind:
		if v, ok := value.(bool); ok {
			return protoreflect.ValueOfBool(v), nil
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		if v, ok := value.(int32); ok {
			return protoreflect.ValueOfInt32(v), nil
		}
		if v, ok := value.(int); ok {
			return protoreflect.ValueOfInt32(int32(v)), nil
		}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		if v, ok := value.(int64); ok {
			return protoreflect.ValueOfInt64(v), nil
		}
		if v, ok := value.(int); ok {
			return protoreflect.ValueOfInt64(int64(v)), nil
		}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if v, ok := value.(uint32); ok {
			return protoreflect.ValueOfUint32(v), nil
		}
		if v, ok := value.(uint); ok {
			return protoreflect.ValueOfUint32(uint32(v)), nil
		}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if v, ok := value.(uint64); ok {
			return protoreflect.ValueOfUint64(v), nil
		}
		if v, ok := value.(uint); ok {
			return protoreflect.ValueOfUint64(uint64(v)), nil
		}
	case protoreflect.FloatKind:
		if v, ok := value.(float32); ok {
			return protoreflect.ValueOfFloat32(v), nil
		}
		if v, ok := value.(float64); ok {
			return protoreflect.ValueOfFloat32(float32(v)), nil
		}
	case protoreflect.DoubleKind:
		if v, ok := value.(float64); ok {
			return protoreflect.ValueOfFloat64(v), nil
		}
	case protoreflect.StringKind:
		if v, ok := value.(string); ok {
			return protoreflect.ValueOfString(v), nil
		}
	case protoreflect.BytesKind:
		if v, ok := value.([]byte); ok {
			return protoreflect.ValueOfBytes(v), nil
		}
	case protoreflect.EnumKind:
		if v, ok := value.(int32); ok {
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(v)), nil
		}
		if v, ok := value.(int); ok {
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(v)), nil
		}
	case protoreflect.MessageKind:
		// Handle nested messages
		if field.IsMap() {
			// TODO: Implement map handling
			return protoreflect.Value{}, fmt.Errorf("map fields are not yet supported")
		} else if field.IsList() {
			// TODO: Implement list handling
			return protoreflect.Value{}, fmt.Errorf("list fields are not yet supported")
		} else {
			// Handle nested message
			if v, ok := value.(map[string]interface{}); ok {
				// Create a new message of the field type
				nestedMsg := dynamicpb.NewMessage(field.Message())
				// Convert the map to the nested message
				if err := mapToProtobuf(v, nestedMsg); err != nil {
					return protoreflect.Value{}, err
				}
				return protoreflect.ValueOfMessage(nestedMsg.ProtoReflect()), nil
			}
		}
	}

	return protoreflect.Value{}, fmt.Errorf("unsupported value type %T for field kind %s", value, field.Kind())
}

// convertFromProtobuf converts a Protocol Buffer message to a Go struct.
func convertFromProtobuf(source proto.Message, target interface{}) error {
	// Convert the message to a map
	sourceMap, err := protoToMap(source)
	if err != nil {
		return fmt.Errorf("failed to convert Protocol Buffer message to map: %w", err)
	}

	// Convert the map to the target struct
	return mapToStruct(sourceMap, target)
}

// protoToMap converts a Protocol Buffer message to a map.
func protoToMap(message proto.Message) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get the message reflection
	msg := message.ProtoReflect()

	// Iterate over all fields
	msg.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		// Get the field name
		name := string(field.Name())

		// Convert the value based on the field type
		var convertedValue interface{}
		var err error

		switch field.Kind() {
		case protoreflect.BoolKind:
			convertedValue = value.Bool()
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			convertedValue = int32(value.Int())
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			convertedValue = value.Int()
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			convertedValue = uint32(value.Uint())
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			convertedValue = value.Uint()
		case protoreflect.FloatKind:
			convertedValue = float32(value.Float())
		case protoreflect.DoubleKind:
			convertedValue = value.Float()
		case protoreflect.StringKind:
			convertedValue = value.String()
		case protoreflect.BytesKind:
			convertedValue = value.Bytes()
		case protoreflect.EnumKind:
			convertedValue = int32(value.Enum())
		case protoreflect.MessageKind:
			if field.IsMap() {
				// TODO: Implement map handling
				convertedValue = nil
				err = fmt.Errorf("map fields are not yet supported")
			} else if field.IsList() {
				// TODO: Implement list handling
				convertedValue = nil
				err = fmt.Errorf("list fields are not yet supported")
			} else {
				// Handle nested message
				nestedMsg := value.Message().Interface()
				convertedValue, err = protoToMap(nestedMsg)
			}
		default:
			convertedValue = nil
			err = fmt.Errorf("unsupported field kind %s", field.Kind())
		}

		if err != nil {
			// Log the error but continue with other fields
			fmt.Printf("Error converting field %s: %v\n", name, err)
		} else {
			result[name] = convertedValue
		}

		return true
	})

	return result, nil
}

// structToMap converts a Go struct to a map.
func structToMap(data interface{}) (map[string]interface{}, error) {
	// If data is already a map, return it
	if m, ok := data.(map[string]interface{}); ok {
		return m, nil
	}

	// Convert the struct to a map using reflection
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data is not a struct or map")
	}

	result := make(map[string]interface{})
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			// Skip unexported fields
			continue
		}

		// Get the field name
		name := field.Name
		// Check for json tag
		if tag := field.Tag.Get("json"); tag != "" {
			name = tag
		}

		// Get the field value
		value := v.Field(i).Interface()

		// Add to the map
		result[name] = value
	}

	return result, nil
}

// mapToStruct converts a map to a Go struct.
func mapToStruct(data map[string]interface{}, target interface{}) error {
	// Get the target value
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	v = v.Elem()

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct")
	}

	// Get the target type
	t := v.Type()

	// Iterate over the struct fields
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			// Skip unexported fields
			continue
		}

		// Get the field name
		name := field.Name
		// Check for json tag
		if tag := field.Tag.Get("json"); tag != "" {
			name = tag
		}

		// Get the value from the map
		value, exists := data[name]
		if !exists {
			continue
		}

		// Set the field value
		fieldValue := v.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		// Convert the value to the field type
		if err := setFieldValue(fieldValue, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", name, err)
		}
	}

	return nil
}

// setFieldValue sets a field value with appropriate type conversion.
func setFieldValue(field reflect.Value, value interface{}) error {
	// Handle nil values
	if value == nil {
		return nil
	}

	// Get the value type
	valueType := reflect.TypeOf(value)
	fieldType := field.Type()

	// Check if the value can be directly assigned to the field
	if valueType.AssignableTo(fieldType) {
		field.Set(reflect.ValueOf(value))
		return nil
	}

	// Handle type conversions
	switch field.Kind() {
	case reflect.Bool:
		if v, ok := value.(bool); ok {
			field.SetBool(v)
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int:
			field.SetInt(int64(v))
			return nil
		case int32:
			field.SetInt(int64(v))
			return nil
		case int64:
			field.SetInt(v)
			return nil
		case float64:
			field.SetInt(int64(v))
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case uint:
			field.SetUint(uint64(v))
			return nil
		case uint32:
			field.SetUint(uint64(v))
			return nil
		case uint64:
			field.SetUint(v)
			return nil
		case int:
			if v >= 0 {
				field.SetUint(uint64(v))
				return nil
			}
		case float64:
			if v >= 0 {
				field.SetUint(uint64(v))
				return nil
			}
		}
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float32:
			field.SetFloat(float64(v))
			return nil
		case float64:
			field.SetFloat(v)
			return nil
		case int:
			field.SetFloat(float64(v))
			return nil
		case int64:
			field.SetFloat(float64(v))
			return nil
		}
	case reflect.String:
		if v, ok := value.(string); ok {
			field.SetString(v)
			return nil
		}
	case reflect.Struct:
		// Handle nested structs
		if v, ok := value.(map[string]interface{}); ok {
			// Create a new instance of the struct
			newStruct := reflect.New(fieldType).Elem()
			// Convert the map to the struct
			if err := mapToStruct(v, newStruct.Addr().Interface()); err != nil {
				return err
			}
			// Set the field value
			field.Set(newStruct)
			return nil
		}
	case reflect.Map:
		// Handle maps
		if v, ok := value.(map[string]interface{}); ok {
			// Create a new map
			mapType := field.Type()
			mapValue := reflect.MakeMap(mapType)
			// Get the map key and value types
			keyType := mapType.Key()
			valueType := mapType.Elem()
			// Convert the map
			for k, v := range v {
				// Convert the key
				keyValue := reflect.ValueOf(k)
				if keyType.Kind() == reflect.String {
					keyValue = reflect.ValueOf(k)
				} else {
					// TODO: Handle other key types
					return fmt.Errorf("unsupported map key type %s", keyType)
				}
				// Convert the value
				elemValue := reflect.New(valueType).Elem()
				if err := setFieldValue(elemValue, v); err != nil {
					return err
				}
				// Set the map entry
				mapValue.SetMapIndex(keyValue, elemValue)
			}
			// Set the field value
			field.Set(mapValue)
			return nil
		}
	case reflect.Slice, reflect.Array:
		// Handle slices and arrays
		if v, ok := value.([]interface{}); ok {
			// Create a new slice
			sliceType := field.Type()
			sliceValue := reflect.MakeSlice(sliceType, len(v), len(v))
			// Get the slice element type
			elemType := sliceType.Elem()
			// Convert the slice
			for i, v := range v {
				// Convert the element
				elemValue := reflect.New(elemType).Elem()
				if err := setFieldValue(elemValue, v); err != nil {
					return err
				}
				// Set the slice element
				sliceValue.Index(i).Set(elemValue)
			}
			// Set the field value
			field.Set(sliceValue)
			return nil
		}
	}

	return fmt.Errorf("cannot convert %T to %s", value, fieldType)
}
