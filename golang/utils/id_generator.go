// Package utils provides utility functions for the HybridPipe system.
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// IDGenerator generates unique IDs.
type IDGenerator struct {
	// prefix is a string prefix for generated IDs.
	prefix string
	// counter is an atomic counter for generating sequential IDs.
	counter uint64
	// mutex protects the counter.
	mutex sync.Mutex
}

// NewIDGenerator creates a new ID generator with the specified prefix.
func NewIDGenerator(prefix string) *IDGenerator {
	return &IDGenerator{
		prefix:  prefix,
		counter: 0,
	}
}

// GenerateID generates a new unique ID.
func (g *IDGenerator) GenerateID() string {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	// Increment the counter
	g.counter++
	
	// Generate the ID
	return fmt.Sprintf("%s-%d-%d", g.prefix, time.Now().UnixNano(), g.counter)
}

// GenerateRandomID generates a new random ID.
func (g *IDGenerator) GenerateRandomID() string {
	// Generate 16 random bytes
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fall back to time-based ID if random generation fails
		return g.GenerateID()
	}
	
	// Convert to hex string
	randomHex := hex.EncodeToString(randomBytes)
	
	// Add prefix
	return fmt.Sprintf("%s-%s", g.prefix, randomHex)
}

// GenerateTimestampedID generates a new ID with a timestamp.
func (g *IDGenerator) GenerateTimestampedID() string {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	// Increment the counter
	g.counter++
	
	// Get the current time
	now := time.Now()
	
	// Generate the ID
	return fmt.Sprintf("%s-%04d%02d%02d-%02d%02d%02d-%d",
		g.prefix,
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(),
		g.counter)
}

// GlobalIDGenerator is a global instance of IDGenerator.
var GlobalIDGenerator = NewIDGenerator("hybridpipe")

// GenerateID generates a new unique ID using the global ID generator.
func GenerateID() string {
	return GlobalIDGenerator.GenerateID()
}

// GenerateRandomID generates a new random ID using the global ID generator.
func GenerateRandomID() string {
	return GlobalIDGenerator.GenerateRandomID()
}

// GenerateTimestampedID generates a new ID with a timestamp using the global ID generator.
func GenerateTimestampedID() string {
	return GlobalIDGenerator.GenerateTimestampedID()
}

// SetGlobalIDGeneratorPrefix sets the prefix for the global ID generator.
func SetGlobalIDGeneratorPrefix(prefix string) {
	GlobalIDGenerator = NewIDGenerator(prefix)
}
