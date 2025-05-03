package serialization

import (
	"encoding/gob"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// TypeConverter defines a function that converts between Go types during serialization/deserialization.
type TypeConverter func(value interface{}) (interface{}, error)

// TypeRegistry manages custom type mappings for serialization.
type TypeRegistry struct {
	// serializationConverters maps from source type to target type for serialization
	serializationConverters map[reflect.Type]TypeConverter
	// deserializationConverters maps from wire type to target type for deserialization
	deserializationConverters map[reflect.Type]TypeConverter
	// mutex protects concurrent access to the maps
	mutex sync.RWMutex
}

// globalTypeRegistry is the global instance of TypeRegistry
var globalTypeRegistry = NewTypeRegistry()

// NewTypeRegistry creates a new type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		serializationConverters:   make(map[reflect.Type]TypeConverter),
		deserializationConverters: make(map[reflect.Type]TypeConverter),
	}
}

// RegisterSerializationConverter registers a converter for serialization.
// The converter will be called when a value of sourceType is being serialized.
func (r *TypeRegistry) RegisterSerializationConverter(sourceType reflect.Type, converter TypeConverter) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.serializationConverters[sourceType] = converter
}

// RegisterDeserializationConverter registers a converter for deserialization.
// The converter will be called when a value of wireType is being deserialized.
func (r *TypeRegistry) RegisterDeserializationConverter(wireType reflect.Type, converter TypeConverter) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.deserializationConverters[wireType] = converter
}

// ConvertForSerialization converts a value for serialization if a converter is registered.
func (r *TypeRegistry) ConvertForSerialization(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	valueType := reflect.TypeOf(value)
	r.mutex.RLock()
	converter, exists := r.serializationConverters[valueType]
	r.mutex.RUnlock()

	if !exists {
		return value, nil
	}

	return converter(value)
}

// ConvertForDeserialization converts a value for deserialization if a converter is registered.
func (r *TypeRegistry) ConvertForDeserialization(value interface{}) error {
	if value == nil {
		return nil
	}

	// Get the pointer to the value
	valuePtr := reflect.ValueOf(value)
	if valuePtr.Kind() != reflect.Ptr {
		return fmt.Errorf("value must be a pointer")
	}

	// Get the actual value
	valueElem := valuePtr.Elem()
	if !valueElem.IsValid() {
		return fmt.Errorf("invalid value")
	}

	// Get the type of the value
	valueType := valueElem.Type()

	// Check if a converter is registered for this type
	r.mutex.RLock()
	converter, exists := r.deserializationConverters[valueType]
	r.mutex.RUnlock()

	if !exists {
		return nil
	}

	// Convert the value
	converted, err := converter(valueElem.Interface())
	if err != nil {
		return err
	}

	// Set the converted value
	convertedValue := reflect.ValueOf(converted)
	if !convertedValue.Type().AssignableTo(valueType) {
		return fmt.Errorf("converted value type %s is not assignable to %s", convertedValue.Type(), valueType)
	}

	valueElem.Set(convertedValue)
	return nil
}

// RegisterTypeMapping registers a bidirectional type mapping for serialization and deserialization.
func RegisterTypeMapping(sourceType, wireType reflect.Type, toWire, fromWire TypeConverter) {
	globalTypeRegistry.RegisterSerializationConverter(sourceType, toWire)
	globalTypeRegistry.RegisterDeserializationConverter(sourceType, fromWire)
}

// RegisterTypeAlias registers a type alias for serialization.
// This is useful for cross-language compatibility where types may have different names.
func RegisterTypeAlias(sourceType reflect.Type, alias string) error {
	// Get the package path and type name
	// pkgPath := sourceType.PkgPath()
	// typeName := sourceType.Name()

	// Create a unique identifier for the type (will be used in future implementation)
	// typeID := fmt.Sprintf("%s.%s", pkgPath, typeName)

	// Register the type with gob
	gob.Register(reflect.New(sourceType).Elem().Interface())

	// TODO: Store the alias mapping for cross-language compatibility

	return nil
}

// Common type converters

// TimeToUnixMillis converts a time.Time to Unix milliseconds (int64).
func TimeToUnixMillis(value interface{}) (interface{}, error) {
	t, ok := value.(time.Time)
	if !ok {
		return nil, fmt.Errorf("value is not a time.Time")
	}
	return t.UnixNano() / int64(time.Millisecond), nil
}

// UnixMillisToTime converts Unix milliseconds (int64) to time.Time.
func UnixMillisToTime(value interface{}) (interface{}, error) {
	millis, ok := value.(int64)
	if !ok {
		return nil, fmt.Errorf("value is not an int64")
	}
	return time.Unix(0, millis*int64(time.Millisecond)), nil
}

// DurationToMillis converts a time.Duration to milliseconds (int64).
func DurationToMillis(value interface{}) (interface{}, error) {
	d, ok := value.(time.Duration)
	if !ok {
		return nil, fmt.Errorf("value is not a time.Duration")
	}
	return d.Milliseconds(), nil
}

// MillisToDuration converts milliseconds (int64) to time.Duration.
func MillisToDuration(value interface{}) (interface{}, error) {
	millis, ok := value.(int64)
	if !ok {
		return nil, fmt.Errorf("value is not an int64")
	}
	return time.Duration(millis) * time.Millisecond, nil
}

// Register common type converters
func init() {
	// Register time.Time <-> int64 (Unix milliseconds)
	RegisterTypeMapping(
		reflect.TypeOf(time.Time{}),
		reflect.TypeOf(int64(0)),
		TimeToUnixMillis,
		UnixMillisToTime,
	)

	// Register time.Duration <-> int64 (milliseconds)
	RegisterTypeMapping(
		reflect.TypeOf(time.Duration(0)),
		reflect.TypeOf(int64(0)),
		DurationToMillis,
		MillisToDuration,
	)
}
