package hybridpipe

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
func (r *TypeRegistry) ConvertForDeserialization(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	valueType := reflect.TypeOf(value)
	r.mutex.RLock()
	converter, exists := r.deserializationConverters[valueType]
	r.mutex.RUnlock()

	if !exists {
		return value, nil
	}

	return converter(value)
}

// RegisterSerializationConverter registers a converter for serialization in the global registry.
func RegisterSerializationConverter(sourceType reflect.Type, converter TypeConverter) {
	globalTypeRegistry.RegisterSerializationConverter(sourceType, converter)
}

// RegisterDeserializationConverter registers a converter for deserialization in the global registry.
func RegisterDeserializationConverter(wireType reflect.Type, converter TypeConverter) {
	globalTypeRegistry.RegisterDeserializationConverter(wireType, converter)
}

// RegisterTypeMapping registers a bidirectional mapping between two types.
// This is a convenience function that registers both serialization and deserialization converters.
func RegisterTypeMapping(sourceType, targetType reflect.Type, toTarget, fromTarget TypeConverter) {
	RegisterSerializationConverter(sourceType, toTarget)
	RegisterDeserializationConverter(targetType, fromTarget)
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
	if t, ok := value.(time.Time); ok {
		return t.UnixNano() / int64(time.Millisecond), nil
	}
	return nil, fmt.Errorf("value is not a time.Time: %T", value)
}

// UnixMillisToTime converts Unix milliseconds (int64) to time.Time.
func UnixMillisToTime(value interface{}) (interface{}, error) {
	if millis, ok := value.(int64); ok {
		return time.Unix(0, millis*int64(time.Millisecond)), nil
	}
	return nil, fmt.Errorf("value is not an int64: %T", value)
}

// DurationToMillis converts a time.Duration to milliseconds (int64).
func DurationToMillis(value interface{}) (interface{}, error) {
	if d, ok := value.(time.Duration); ok {
		return d.Milliseconds(), nil
	}
	return nil, fmt.Errorf("value is not a time.Duration: %T", value)
}

// MillisToDuration converts milliseconds (int64) to time.Duration.
func MillisToDuration(value interface{}) (interface{}, error) {
	if millis, ok := value.(int64); ok {
		return time.Duration(millis) * time.Millisecond, nil
	}
	return nil, fmt.Errorf("value is not an int64: %T", value)
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
