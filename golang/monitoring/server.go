package monitoring

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// MetricsServer is a server that exposes metrics and traces.
type MetricsServer struct {
	// server is the HTTP server
	server *http.Server
	// metricsRegistry is the metrics registry
	metricsRegistry *MetricsRegistry
	// tracer is the message tracer
	tracer Tracer
	// mutex protects concurrent access to the server
	mutex sync.RWMutex
	// running indicates if the server is running
	running bool
}

// NewMetricsServer creates a new metrics server.
func NewMetricsServer(address string, metricsRegistry *MetricsRegistry, tracer Tracer) *MetricsServer {
	// Create a new server
	server := &http.Server{
		Addr:         address,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Create a new metrics server
	ms := &MetricsServer{
		server:          server,
		metricsRegistry: metricsRegistry,
		tracer:          tracer,
	}

	// Set up the HTTP handlers
	http.HandleFunc("/metrics", ms.handleMetrics)
	http.HandleFunc("/traces", ms.handleTraces)
	http.HandleFunc("/health", ms.handleHealth)
	http.HandleFunc("/", ms.handleRoot)

	return ms
}

// Start starts the metrics server.
func (ms *MetricsServer) Start() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	if ms.running {
		return fmt.Errorf("server is already running")
	}

	// Start the server in a goroutine
	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting monitoring server: %v", err)
		}
	}()

	ms.running = true
	log.Printf("Monitoring server started on %s", ms.server.Addr)
	return nil
}

// Stop stops the metrics server.
func (ms *MetricsServer) Stop() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	if !ms.running {
		return fmt.Errorf("server is not running")
	}

	// Stop the server
	if err := ms.server.Close(); err != nil {
		return fmt.Errorf("failed to stop monitoring server: %w", err)
	}

	ms.running = false
	log.Printf("Monitoring server stopped")
	return nil
}

// IsRunning returns true if the server is running.
func (ms *MetricsServer) IsRunning() bool {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	return ms.running
}

// handleMetrics handles requests to the /metrics endpoint.
func (ms *MetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Get the metrics as JSON
	metrics, err := ms.metricsRegistry.GetMetricsJSON()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting metrics: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Write the metrics
	w.Write(metrics)
}

// handleTraces handles requests to the /traces endpoint.
func (ms *MetricsServer) handleTraces(w http.ResponseWriter, r *http.Request) {
	// Get the traces
	traces := ms.tracer.GetTraces()

	// Convert the traces to JSON
	tracesJSON, err := json.Marshal(traces)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshaling traces: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Write the traces
	w.Write(tracesJSON)
}

// handleHealth handles requests to the /health endpoint.
func (ms *MetricsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Create a health check response
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now(),
	}

	// Convert the health check to JSON
	healthJSON, err := json.Marshal(health)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshaling health check: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Write the health check
	w.Write(healthJSON)
}

// handleRoot handles requests to the root endpoint.
func (ms *MetricsServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	// Create a response
	response := map[string]interface{}{
		"name":        "HybridPipe Monitoring",
		"description": "Monitoring server for the HybridPipe system",
		"endpoints": []string{
			"/metrics",
			"/traces",
			"/health",
		},
	}

	// Convert the response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshaling response: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Write the response
	w.Write(responseJSON)
}

// DefaultMetricsServer is the default metrics server.
var DefaultMetricsServer *MetricsServer

// StartDefaultMonitoringServer starts the default metrics server.
func StartDefaultMonitoringServer(address string) error {
	// Create a new metrics server
	DefaultMetricsServer = NewMetricsServer(address, DefaultRegistry, NewMessageTracer(1000))

	// Start the server
	return DefaultMetricsServer.Start()
}

// StopDefaultMonitoringServer stops the default metrics server.
func StopDefaultMonitoringServer() error {
	if DefaultMetricsServer == nil {
		return fmt.Errorf("default metrics server is not running")
	}

	return DefaultMetricsServer.Stop()
}
