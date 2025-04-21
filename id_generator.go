package hybridpipe

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// IDGenerator is the interface for generating trace and span IDs
type IDGenerator interface {
	// GenerateTraceID generates a new trace ID
	GenerateTraceID() string
	// GenerateSpanID generates a new span ID
	GenerateSpanID() string
}

// randomIDGenerator generates random trace and span IDs
type randomIDGenerator struct{}

// GenerateTraceID generates a new trace ID
func (g *randomIDGenerator) GenerateTraceID() string {
	return generateRandomID(16)
}

// GenerateSpanID generates a new span ID
func (g *randomIDGenerator) GenerateSpanID() string {
	return generateRandomID(8)
}

// generateRandomID generates a random ID of the given length
func generateRandomID(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		// This should never happen, but if it does, we'll use a fallback
		// that's not as secure but still works
		for i := 0; i < length; i++ {
			bytes[i] = byte(i)
		}
	}
	return hex.EncodeToString(bytes)
}

// Sampler is the interface for sampling traces
type Sampler interface {
	// ShouldSample determines whether a trace should be sampled
	ShouldSample(traceID string, name string, attributes map[string]interface{}) bool
}

// probabilitySampler samples traces based on a probability
type probabilitySampler struct {
	// probability is the probability that a trace will be sampled (0.0-1.0)
	probability float64
}

// ShouldSample determines whether a trace should be sampled
func (s *probabilitySampler) ShouldSample(traceID string, name string, attributes map[string]interface{}) bool {
	// If probability is 0, never sample
	if s.probability <= 0.0 {
		return false
	}

	// If probability is 1, always sample
	if s.probability >= 1.0 {
		return true
	}

	// Use the first 8 bytes of the trace ID as a random value
	if len(traceID) < 2 {
		return false
	}

	// Convert the first two characters of the trace ID to a byte
	b, err := hex.DecodeString(traceID[:2])
	if err != nil || len(b) == 0 {
		return false
	}

	// Use the byte as a random value between 0 and 255
	randomValue := float64(b[0]) / 255.0

	// Sample if the random value is less than the probability
	return randomValue < s.probability
}

// alwaysOnSampler always samples traces
type alwaysOnSampler struct{}

// ShouldSample always returns true
func (s *alwaysOnSampler) ShouldSample(traceID string, name string, attributes map[string]interface{}) bool {
	return true
}

// alwaysOffSampler never samples traces
type alwaysOffSampler struct{}

// ShouldSample always returns false
func (s *alwaysOffSampler) ShouldSample(traceID string, name string, attributes map[string]interface{}) bool {
	return false
}

// Propagator is the interface for propagating span contexts
type Propagator interface {
	// Extract extracts a span context from a carrier
	Extract(carrier interface{}) SpanContext
	// Inject injects a span context into a carrier
	Inject(spanContext SpanContext, carrier interface{})
}

// w3cPropagator propagates span contexts using the W3C Trace Context format
type w3cPropagator struct{}

// Extract extracts a span context from a carrier
func (p *w3cPropagator) Extract(carrier interface{}) SpanContext {
	if carrier == nil {
		return SpanContext{}
	}

	// Handle different carrier types
	switch c := carrier.(type) {
	case map[string]interface{}:
		// Extract from map
		return p.extractFromMap(c)
	case map[string]string:
		// Extract from string map
		return p.extractFromStringMap(c)
	case map[string][]string:
		// Extract from multi-value map
		return p.extractFromMultiValueMap(c)
	default:
		// Unsupported carrier type
		return SpanContext{}
	}
}

// extractFromMap extracts a span context from a map
func (p *w3cPropagator) extractFromMap(carrier map[string]interface{}) SpanContext {
	// Look for traceparent header
	traceparentValue, ok := carrier["traceparent"]
	if !ok {
		return SpanContext{}
	}

	// Convert to string
	traceparent, ok := traceparentValue.(string)
	if !ok {
		return SpanContext{}
	}

	// Parse traceparent
	return p.parseTraceparent(traceparent)
}

// extractFromStringMap extracts a span context from a string map
func (p *w3cPropagator) extractFromStringMap(carrier map[string]string) SpanContext {
	// Look for traceparent header
	traceparent, ok := carrier["traceparent"]
	if !ok {
		return SpanContext{}
	}

	// Parse traceparent
	return p.parseTraceparent(traceparent)
}

// extractFromMultiValueMap extracts a span context from a multi-value map
func (p *w3cPropagator) extractFromMultiValueMap(carrier map[string][]string) SpanContext {
	// Look for traceparent header
	traceparentValues, ok := carrier["traceparent"]
	if !ok || len(traceparentValues) == 0 {
		return SpanContext{}
	}

	// Parse traceparent
	return p.parseTraceparent(traceparentValues[0])
}

// parseTraceparent parses a traceparent header
func (p *w3cPropagator) parseTraceparent(traceparent string) SpanContext {
	// Traceparent format: 00-traceID-spanID-flags
	// Example: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01

	// Check length
	if len(traceparent) < 55 {
		return SpanContext{}
	}

	// Check version
	if traceparent[:2] != "00" {
		return SpanContext{}
	}

	// Check separators
	if traceparent[2:3] != "-" || traceparent[35:36] != "-" || traceparent[52:53] != "-" {
		return SpanContext{}
	}

	// Extract trace ID
	traceID := traceparent[3:35]

	// Extract span ID
	spanID := traceparent[36:52]

	// Extract flags
	flags := traceparent[53:55]

	// Convert flags to byte
	flagsByte, err := hex.DecodeString(flags)
	if err != nil || len(flagsByte) == 0 {
		return SpanContext{}
	}

	return SpanContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flagsByte[0],
		Remote:     true,
	}
}

// Inject injects a span context into a carrier
func (p *w3cPropagator) Inject(spanContext SpanContext, carrier interface{}) {
	if spanContext.TraceID == "" || spanContext.SpanID == "" {
		return
	}

	// Create traceparent header
	traceparent := fmt.Sprintf("00-%s-%s-%02x", spanContext.TraceID, spanContext.SpanID, spanContext.TraceFlags)

	// Handle different carrier types
	switch c := carrier.(type) {
	case map[string]interface{}:
		// Inject into map
		c["traceparent"] = traceparent
	case map[string]string:
		// Inject into string map
		c["traceparent"] = traceparent
	case map[string][]string:
		// Inject into multi-value map
		c["traceparent"] = []string{traceparent}
	}
}
