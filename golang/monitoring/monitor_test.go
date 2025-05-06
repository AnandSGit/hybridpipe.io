package monitoring

import (
	"sync"
	"testing"
	"time"
)

// MockMetric is a mock implementation of a metric for testing.
type MockMetric struct {
	// Name is the metric name
	Name string
	// Description is the metric description
	Description string
	// Unit is the metric unit
	Unit string
	// Value is the current value
	Value float64
	// Count is the number of updates
	Count int
	// Sum is the sum of all values
	Sum float64
	// Min is the minimum value
	Min float64
	// Max is the maximum value
	Max float64
	// LastUpdate is the time of the last update
	LastUpdate time.Time
	// mutex protects concurrent access
	mutex sync.RWMutex
}

// NewMockMetric creates a new MockMetric.
func NewMockMetric(name, description, unit string) *MockMetric {
	return &MockMetric{
		Name:        name,
		Description: description,
		Unit:        unit,
		Value:       0,
		Count:       0,
		Sum:         0,
		Min:         0,
		Max:         0,
		LastUpdate:  time.Time{},
	}
}

// Update updates the metric value.
func (m *MockMetric) Update(value float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Value = value
	m.Count++
	m.Sum += value
	m.LastUpdate = time.Now()

	if m.Count == 1 {
		m.Min = value
		m.Max = value
	} else {
		if value < m.Min {
			m.Min = value
		}
		if value > m.Max {
			m.Max = value
		}
	}
}

// GetValue returns the current value.
func (m *MockMetric) GetValue() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.Value
}

// GetCount returns the number of updates.
func (m *MockMetric) GetCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.Count
}

// GetSum returns the sum of all values.
func (m *MockMetric) GetSum() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.Sum
}

// GetAverage returns the average value.
func (m *MockMetric) GetAverage() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.Count == 0 {
		return 0
	}
	return m.Sum / float64(m.Count)
}

// GetMin returns the minimum value.
func (m *MockMetric) GetMin() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.Min
}

// GetMax returns the maximum value.
func (m *MockMetric) GetMax() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.Max
}

// MockMonitor is a mock implementation of a monitor for testing.
type MockMonitor struct {
	// Metrics is a map of metric names to metrics
	Metrics map[string]*MockMetric
	// mutex protects concurrent access
	mutex sync.RWMutex
}

// NewMockMonitor creates a new MockMonitor.
func NewMockMonitor() *MockMonitor {
	return &MockMonitor{
		Metrics: make(map[string]*MockMetric),
	}
}

// Counter creates or gets a counter metric.
func (m *MockMonitor) Counter(name, description, unit string) *MockMetric {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if metric, ok := m.Metrics[name]; ok {
		return metric
	}

	metric := NewMockMetric(name, description, unit)
	m.Metrics[name] = metric
	return metric
}

// Gauge creates or gets a gauge metric.
func (m *MockMonitor) Gauge(name, description, unit string) *MockMetric {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if metric, ok := m.Metrics[name]; ok {
		return metric
	}

	metric := NewMockMetric(name, description, unit)
	m.Metrics[name] = metric
	return metric
}

// Histogram creates or gets a histogram metric.
func (m *MockMonitor) Histogram(name, description, unit string) *MockMetric {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if metric, ok := m.Metrics[name]; ok {
		return metric
	}

	metric := NewMockMetric(name, description, unit)
	m.Metrics[name] = metric
	return metric
}

// GetMetric gets a metric by name.
func (m *MockMonitor) GetMetric(name string) (*MockMetric, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	metric, ok := m.Metrics[name]
	return metric, ok
}

// GetMetrics gets all metrics.
func (m *MockMonitor) GetMetrics() map[string]*MockMetric {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Create a copy to avoid concurrent access issues
	metrics := make(map[string]*MockMetric, len(m.Metrics))
	for name, metric := range m.Metrics {
		metrics[name] = metric
	}
	return metrics
}

// TestMonitorBasic tests basic monitor functionality.
func TestMonitorBasic(t *testing.T) {
	// Create a mock monitor
	monitor := NewMockMonitor()

	// Create a counter
	counter := monitor.Counter("test.counter", "Test counter", "count")

	// Update the counter
	counter.Update(1)
	counter.Update(2)
	counter.Update(3)

	// Check the counter values
	if counter.GetValue() != 3 {
		t.Errorf("Expected counter value = 3, got %f", counter.GetValue())
	}
	if counter.GetCount() != 3 {
		t.Errorf("Expected counter count = 3, got %d", counter.GetCount())
	}
	if counter.GetSum() != 6 {
		t.Errorf("Expected counter sum = 6, got %f", counter.GetSum())
	}
	if counter.GetAverage() != 2 {
		t.Errorf("Expected counter average = 2, got %f", counter.GetAverage())
	}
	if counter.GetMin() != 1 {
		t.Errorf("Expected counter min = 1, got %f", counter.GetMin())
	}
	if counter.GetMax() != 3 {
		t.Errorf("Expected counter max = 3, got %f", counter.GetMax())
	}

	// Create a gauge
	gauge := monitor.Gauge("test.gauge", "Test gauge", "value")

	// Update the gauge
	gauge.Update(10)
	gauge.Update(20)
	gauge.Update(15)

	// Check the gauge values
	if gauge.GetValue() != 15 {
		t.Errorf("Expected gauge value = 15, got %f", gauge.GetValue())
	}
	if gauge.GetCount() != 3 {
		t.Errorf("Expected gauge count = 3, got %d", gauge.GetCount())
	}
	if gauge.GetSum() != 45 {
		t.Errorf("Expected gauge sum = 45, got %f", gauge.GetSum())
	}
	if gauge.GetAverage() != 15 {
		t.Errorf("Expected gauge average = 15, got %f", gauge.GetAverage())
	}
	if gauge.GetMin() != 10 {
		t.Errorf("Expected gauge min = 10, got %f", gauge.GetMin())
	}
	if gauge.GetMax() != 20 {
		t.Errorf("Expected gauge max = 20, got %f", gauge.GetMax())
	}

	// Create a histogram
	histogram := monitor.Histogram("test.histogram", "Test histogram", "ms")

	// Update the histogram
	histogram.Update(100)
	histogram.Update(200)
	histogram.Update(300)
	histogram.Update(400)
	histogram.Update(500)

	// Check the histogram values
	if histogram.GetValue() != 500 {
		t.Errorf("Expected histogram value = 500, got %f", histogram.GetValue())
	}
	if histogram.GetCount() != 5 {
		t.Errorf("Expected histogram count = 5, got %d", histogram.GetCount())
	}
	if histogram.GetSum() != 1500 {
		t.Errorf("Expected histogram sum = 1500, got %f", histogram.GetSum())
	}
	if histogram.GetAverage() != 300 {
		t.Errorf("Expected histogram average = 300, got %f", histogram.GetAverage())
	}
	if histogram.GetMin() != 100 {
		t.Errorf("Expected histogram min = 100, got %f", histogram.GetMin())
	}
	if histogram.GetMax() != 500 {
		t.Errorf("Expected histogram max = 500, got %f", histogram.GetMax())
	}

	// Get metrics by name
	retrievedCounter, ok := monitor.GetMetric("test.counter")
	if !ok {
		t.Fatal("Failed to retrieve counter")
	}
	if retrievedCounter.Name != "test.counter" {
		t.Errorf("Expected counter name = test.counter, got %s", retrievedCounter.Name)
	}

	retrievedGauge, ok := monitor.GetMetric("test.gauge")
	if !ok {
		t.Fatal("Failed to retrieve gauge")
	}
	if retrievedGauge.Name != "test.gauge" {
		t.Errorf("Expected gauge name = test.gauge, got %s", retrievedGauge.Name)
	}

	retrievedHistogram, ok := monitor.GetMetric("test.histogram")
	if !ok {
		t.Fatal("Failed to retrieve histogram")
	}
	if retrievedHistogram.Name != "test.histogram" {
		t.Errorf("Expected histogram name = test.histogram, got %s", retrievedHistogram.Name)
	}

	// Get all metrics
	metrics := monitor.GetMetrics()
	if len(metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(metrics))
	}
}

// TestMonitorConcurrency tests concurrent use of the monitor.
func TestMonitorConcurrency(t *testing.T) {
	// Create a mock monitor
	monitor := NewMockMonitor()

	// Number of goroutines
	const numGoroutines = 10
	// Number of updates per goroutine
	const numUpdates = 100

	// Create a wait group to wait for all goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Create metrics
			counterName := "counter.goroutine." + string(rune(id+65))
			counter := monitor.Counter(counterName, "Counter for goroutine "+string(rune(id+65)), "count")

			gaugeName := "gauge.goroutine." + string(rune(id+65))
			gauge := monitor.Gauge(gaugeName, "Gauge for goroutine "+string(rune(id+65)), "value")

			histogramName := "histogram.goroutine." + string(rune(id+65))
			histogram := monitor.Histogram(histogramName, "Histogram for goroutine "+string(rune(id+65)), "ms")

			// Update metrics
			for j := 0; j < numUpdates; j++ {
				counter.Update(float64(j))
				gauge.Update(float64(j * 10))
				histogram.Update(float64(j * 100))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check that all metrics were created
	metrics := monitor.GetMetrics()
	if len(metrics) != numGoroutines*3 {
		t.Errorf("Expected %d metrics, got %d", numGoroutines*3, len(metrics))
	}

	// Check that each metric has the correct number of updates
	for name, metric := range metrics {
		if metric.GetCount() != numUpdates {
			t.Errorf("Metric %s: expected count = %d, got %d", name, numUpdates, metric.GetCount())
		}
	}
}

// TestMetricTypes tests different metric types.
func TestMetricTypes(t *testing.T) {
	// Create a mock monitor
	monitor := NewMockMonitor()

	// Test counter behavior
	t.Run("Counter", func(t *testing.T) {
		counter := monitor.Counter("test.counter.behavior", "Test counter behavior", "count")

		// Update the counter with increasing values
		counter.Update(1)
		counter.Update(2)
		counter.Update(3)

		// Check that the counter accumulates
		if counter.GetSum() != 6 {
			t.Errorf("Expected counter sum = 6, got %f", counter.GetSum())
		}

		// Check that the counter keeps the last value
		if counter.GetValue() != 3 {
			t.Errorf("Expected counter value = 3, got %f", counter.GetValue())
		}
	})

	// Test gauge behavior
	t.Run("Gauge", func(t *testing.T) {
		gauge := monitor.Gauge("test.gauge.behavior", "Test gauge behavior", "value")

		// Update the gauge with different values
		gauge.Update(10)
		gauge.Update(5)
		gauge.Update(15)

		// Check that the gauge keeps the last value
		if gauge.GetValue() != 15 {
			t.Errorf("Expected gauge value = 15, got %f", gauge.GetValue())
		}

		// Check that the gauge tracks min and max
		if gauge.GetMin() != 5 {
			t.Errorf("Expected gauge min = 5, got %f", gauge.GetMin())
		}
		if gauge.GetMax() != 15 {
			t.Errorf("Expected gauge max = 15, got %f", gauge.GetMax())
		}
	})

	// Test histogram behavior
	t.Run("Histogram", func(t *testing.T) {
		histogram := monitor.Histogram("test.histogram.behavior", "Test histogram behavior", "ms")

		// Update the histogram with different values
		values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
		for _, value := range values {
			histogram.Update(value)
		}

		// Check that the histogram keeps the last value
		if histogram.GetValue() != 100 {
			t.Errorf("Expected histogram value = 100, got %f", histogram.GetValue())
		}

		// Check that the histogram tracks min and max
		if histogram.GetMin() != 10 {
			t.Errorf("Expected histogram min = 10, got %f", histogram.GetMin())
		}
		if histogram.GetMax() != 100 {
			t.Errorf("Expected histogram max = 100, got %f", histogram.GetMax())
		}

		// Check that the histogram calculates average correctly
		if histogram.GetAverage() != 55 {
			t.Errorf("Expected histogram average = 55, got %f", histogram.GetAverage())
		}
	})
}

// TestMetricReuse tests reusing metrics with the same name.
func TestMetricReuse(t *testing.T) {
	// Create a mock monitor
	monitor := NewMockMonitor()

	// Create a metric
	metric1 := monitor.Counter("test.reuse", "Test reuse", "count")

	// Update the metric
	metric1.Update(1)
	metric1.Update(2)

	// Get the same metric again
	metric2 := monitor.Counter("test.reuse", "Test reuse", "count")

	// Check that it's the same metric
	if metric2.GetCount() != 2 {
		t.Errorf("Expected metric count = 2, got %d", metric2.GetCount())
	}
	if metric2.GetSum() != 3 {
		t.Errorf("Expected metric sum = 3, got %f", metric2.GetSum())
	}

	// Update the metric again
	metric2.Update(3)

	// Check that both references see the update
	if metric1.GetCount() != 3 {
		t.Errorf("Expected metric1 count = 3, got %d", metric1.GetCount())
	}
	if metric1.GetSum() != 6 {
		t.Errorf("Expected metric1 sum = 6, got %f", metric1.GetSum())
	}
	if metric2.GetCount() != 3 {
		t.Errorf("Expected metric2 count = 3, got %d", metric2.GetCount())
	}
	if metric2.GetSum() != 6 {
		t.Errorf("Expected metric2 sum = 6, got %f", metric2.GetSum())
	}
}
