package monitoring

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MetricType represents the type of a metric.
type MetricType int

const (
	// MetricTypeCounter is a metric that only increases.
	MetricTypeCounter MetricType = iota
	// MetricTypeGauge is a metric that can increase or decrease.
	MetricTypeGauge
	// MetricTypeHistogram is a metric that tracks the distribution of values.
	MetricTypeHistogram
)

// Metric represents a metric.
type Metric struct {
	// Name is the name of the metric
	Name string
	// Type is the type of the metric
	Type MetricType
	// Value is the current value of the metric
	Value int64
	// Tags are additional tags for the metric
	Tags map[string]string
	// Timestamp is the time the metric was last updated
	Timestamp time.Time
	// Buckets are the histogram buckets (only used for histograms)
	Buckets map[int64]int64
	// mutex protects concurrent access to the metric
	mutex sync.RWMutex
}

// NewCounter creates a new counter metric.
func NewCounter(name string) *Metric {
	return &Metric{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     0,
		Tags:      make(map[string]string),
		Timestamp: time.Now(),
	}
}

// NewGauge creates a new gauge metric.
func NewGauge(name string) *Metric {
	return &Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     0,
		Tags:      make(map[string]string),
		Timestamp: time.Now(),
	}
}

// NewHistogram creates a new histogram metric.
func NewHistogram(name string, buckets []int64) *Metric {
	bucketMap := make(map[int64]int64)
	for _, bucket := range buckets {
		bucketMap[bucket] = 0
	}
	return &Metric{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     0,
		Tags:      make(map[string]string),
		Timestamp: time.Now(),
		Buckets:   bucketMap,
	}
}

// AddTag adds a tag to the metric.
func (m *Metric) AddTag(key, value string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Tags[key] = value
}

// Increment increments the metric by the specified value.
func (m *Metric) Increment(value int64) {
	if m.Type == MetricTypeCounter {
		atomic.AddInt64(&m.Value, value)
	} else if m.Type == MetricTypeGauge {
		atomic.AddInt64(&m.Value, value)
	}
	m.mutex.Lock()
	m.Timestamp = time.Now()
	m.mutex.Unlock()
}

// Decrement decrements the metric by the specified value.
func (m *Metric) Decrement(value int64) {
	if m.Type == MetricTypeGauge {
		atomic.AddInt64(&m.Value, -value)
	}
	m.mutex.Lock()
	m.Timestamp = time.Now()
	m.mutex.Unlock()
}

// Set sets the metric to the specified value.
func (m *Metric) Set(value int64) {
	if m.Type == MetricTypeGauge {
		atomic.StoreInt64(&m.Value, value)
	}
	m.mutex.Lock()
	m.Timestamp = time.Now()
	m.mutex.Unlock()
}

// Observe adds a value to the histogram.
func (m *Metric) Observe(value int64) {
	if m.Type != MetricTypeHistogram {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Find the appropriate bucket
	for bucket := range m.Buckets {
		if value <= bucket {
			m.Buckets[bucket]++
			break
		}
	}

	// Update the total count
	m.Value++
	m.Timestamp = time.Now()
}

// GetValue returns the current value of the metric.
func (m *Metric) GetValue() int64 {
	return atomic.LoadInt64(&m.Value)
}

// GetBuckets returns the histogram buckets.
func (m *Metric) GetBuckets() map[int64]int64 {
	if m.Type != MetricTypeHistogram {
		return nil
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy of the buckets
	buckets := make(map[int64]int64)
	for k, v := range m.Buckets {
		buckets[k] = v
	}
	return buckets
}

// String returns a string representation of the metric.
func (m *Metric) String() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.Type == MetricTypeHistogram {
		return fmt.Sprintf("%s: %d observations (histogram)", m.Name, m.Value)
	}
	return fmt.Sprintf("%s: %d", m.Name, m.Value)
}

// MarshalJSON marshals the metric to JSON.
func (m *Metric) MarshalJSON() ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Create a map for the JSON
	data := map[string]interface{}{
		"name":      m.Name,
		"type":      m.Type,
		"value":     m.Value,
		"tags":      m.Tags,
		"timestamp": m.Timestamp,
	}

	// Add buckets for histograms
	if m.Type == MetricTypeHistogram {
		data["buckets"] = m.Buckets
	}

	return json.Marshal(data)
}

// MetricsRegistry is a registry of metrics.
type MetricsRegistry struct {
	// metrics is the map of metrics
	metrics map[string]*Metric
	// mutex protects concurrent access to the metrics
	mutex sync.RWMutex
}

// NewMetricsRegistry creates a new metrics registry.
func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		metrics: make(map[string]*Metric),
	}
}

// Register registers a metric.
func (mr *MetricsRegistry) Register(metric *Metric) {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	mr.metrics[metric.Name] = metric
}

// GetMetric returns a metric by name.
func (mr *MetricsRegistry) GetMetric(name string) *Metric {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	return mr.metrics[name]
}

// GetMetrics returns all metrics.
func (mr *MetricsRegistry) GetMetrics() []*Metric {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	// Return a copy of the metrics
	metrics := make([]*Metric, 0, len(mr.metrics))
	for _, metric := range mr.metrics {
		metrics = append(metrics, metric)
	}
	return metrics
}

// GetMetricsJSON returns all metrics as JSON.
func (mr *MetricsRegistry) GetMetricsJSON() ([]byte, error) {
	metrics := mr.GetMetrics()
	return json.Marshal(metrics)
}

// DefaultRegistry is the default metrics registry.
var DefaultRegistry = NewMetricsRegistry()

// Register registers a metric with the default registry.
func Register(metric *Metric) {
	DefaultRegistry.Register(metric)
}

// GetMetric returns a metric by name from the default registry.
func GetMetric(name string) *Metric {
	return DefaultRegistry.GetMetric(name)
}

// GetMetrics returns all metrics from the default registry.
func GetMetrics() []*Metric {
	return DefaultRegistry.GetMetrics()
}

// GetMetricsJSON returns all metrics as JSON from the default registry.
func GetMetricsJSON() ([]byte, error) {
	return DefaultRegistry.GetMetricsJSON()
}
