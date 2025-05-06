// Package monitoring provides monitoring functionality for the HybridPipe system.
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"hybridpipe.io/core"
	"hybridpipe.io/middleware"
)

// MonitoredRouter is a middleware that adds monitoring to a HybridPipe implementation.
type MonitoredRouter struct {
	*middleware.ContextAwareBaseMiddleware
	// Stats contains message statistics.
	Stats *MessageStats
	// Name is the name of the router.
	Name string
	// MonitoringEnabled indicates whether monitoring is enabled.
	MonitoringEnabled bool
	// MonitoringCallback is called when a message is sent or received.
	MonitoringCallback func(event MonitoringEvent)
}

// MonitoringEvent represents a monitoring event.
type MonitoringEvent struct {
	// Type is the type of event.
	Type string `json:"type"`
	// Pipe is the pipe name.
	Pipe string `json:"pipe"`
	// Timestamp is the time the event occurred.
	Timestamp time.Time `json:"timestamp"`
	// Size is the size of the message in bytes.
	Size uint64 `json:"size"`
	// Error is the error message if the operation failed.
	Error string `json:"error,omitempty"`
	// RouterName is the name of the router.
	RouterName string `json:"router_name"`
}

// NewMonitoredRouter creates a new MonitoredRouter.
func NewMonitoredRouter(router core.HybridPipe, name string) *MonitoredRouter {
	return &MonitoredRouter{
		ContextAwareBaseMiddleware: middleware.NewContextAwareBaseMiddleware(router),
		Stats:                      NewMessageStats(),
		Name:                       name,
		MonitoringEnabled:          true,
		MonitoringCallback:         defaultMonitoringCallback,
	}
}

// Dispatch sends a message to the specified pipe with monitoring.
func (mr *MonitoredRouter) Dispatch(pipe string, data any) error {
	if !mr.MonitoringEnabled {
		return mr.Router.Dispatch(pipe, data)
	}

	// Encode the data to get its size
	encoded, err := core.Encode(data)
	if err != nil {
		// Record the error
		mr.Stats.IncrementError()
		// Create a monitoring event
		event := MonitoringEvent{
			Type:       "dispatch_error",
			Pipe:       pipe,
			Timestamp:  time.Now(),
			Error:      err.Error(),
			RouterName: mr.Name,
		}
		// Call the monitoring callback
		mr.MonitoringCallback(event)
		return err
	}

	// Record the message size
	size := uint64(len(encoded))

	// Dispatch the message
	err = mr.Router.Dispatch(pipe, data)

	if err != nil {
		// Record the error
		mr.Stats.IncrementError()
		// Create a monitoring event
		event := MonitoringEvent{
			Type:       "dispatch_error",
			Pipe:       pipe,
			Timestamp:  time.Now(),
			Size:       size,
			Error:      err.Error(),
			RouterName: mr.Name,
		}
		// Call the monitoring callback
		mr.MonitoringCallback(event)
	} else {
		// Record the message
		mr.Stats.IncrementSent(size)
		// Create a monitoring event
		event := MonitoringEvent{
			Type:       "dispatch",
			Pipe:       pipe,
			Timestamp:  time.Now(),
			Size:       size,
			RouterName: mr.Name,
		}
		// Call the monitoring callback
		mr.MonitoringCallback(event)
	}

	return err
}

// DispatchWithContext sends a message with context for cancellation and timeouts.
func (mr *MonitoredRouter) DispatchWithContext(ctx context.Context, pipe string, data any) error {
	if !mr.MonitoringEnabled {
		if mr.ContextRouter != nil {
			return mr.ContextRouter.DispatchWithContext(ctx, pipe, data)
		}
		return mr.Router.Dispatch(pipe, data)
	}

	// Encode the data to get its size
	encoded, err := core.Encode(data)
	if err != nil {
		// Record the error
		mr.Stats.IncrementError()
		// Create a monitoring event
		event := MonitoringEvent{
			Type:       "dispatch_with_context_error",
			Pipe:       pipe,
			Timestamp:  time.Now(),
			Error:      err.Error(),
			RouterName: mr.Name,
		}
		// Call the monitoring callback
		mr.MonitoringCallback(event)
		return err
	}

	// Record the message size
	size := uint64(len(encoded))

	// Dispatch the message
	var dispatchErr error
	if mr.ContextRouter != nil {
		dispatchErr = mr.ContextRouter.DispatchWithContext(ctx, pipe, data)
	} else {
		dispatchErr = mr.Router.Dispatch(pipe, data)
	}

	if dispatchErr != nil {
		// Record the error
		mr.Stats.IncrementError()
		// Create a monitoring event
		event := MonitoringEvent{
			Type:       "dispatch_with_context_error",
			Pipe:       pipe,
			Timestamp:  time.Now(),
			Size:       size,
			Error:      dispatchErr.Error(),
			RouterName: mr.Name,
		}
		// Call the monitoring callback
		mr.MonitoringCallback(event)
	} else {
		// Record the message
		mr.Stats.IncrementSent(size)
		// Create a monitoring event
		event := MonitoringEvent{
			Type:       "dispatch_with_context",
			Pipe:       pipe,
			Timestamp:  time.Now(),
			Size:       size,
			RouterName: mr.Name,
		}
		// Call the monitoring callback
		mr.MonitoringCallback(event)
	}

	return dispatchErr
}

// Subscribe registers a callback function to be called when messages arrive on the specified pipe.
func (mr *MonitoredRouter) Subscribe(pipe string, callback core.Process) error {
	if !mr.MonitoringEnabled {
		return mr.Router.Subscribe(pipe, callback)
	}

	// Create a wrapper function that records message statistics
	wrapper := func(data []byte) error {
		// Record the message
		size := uint64(len(data))
		mr.Stats.IncrementReceived(size)

		// Create a monitoring event
		event := MonitoringEvent{
			Type:       "receive",
			Pipe:       pipe,
			Timestamp:  time.Now(),
			Size:       size,
			RouterName: mr.Name,
		}

		// Call the monitoring callback
		mr.MonitoringCallback(event)

		// Call the original callback
		err := callback(data)
		if err != nil {
			// Record the error
			mr.Stats.IncrementError()
			// Create a monitoring event
			errorEvent := MonitoringEvent{
				Type:       "receive_error",
				Pipe:       pipe,
				Timestamp:  time.Now(),
				Size:       size,
				Error:      err.Error(),
				RouterName: mr.Name,
			}
			// Call the monitoring callback
			mr.MonitoringCallback(errorEvent)
		}

		return err
	}

	return mr.Router.Subscribe(pipe, wrapper)
}

// SubscribeWithContext registers a callback with context for cancellation.
func (mr *MonitoredRouter) SubscribeWithContext(ctx context.Context, pipe string, callback core.Process) error {
	if !mr.MonitoringEnabled {
		if mr.ContextRouter != nil {
			return mr.ContextRouter.SubscribeWithContext(ctx, pipe, callback)
		}
		return mr.Router.Subscribe(pipe, callback)
	}

	// Create a wrapper function that records message statistics
	wrapper := func(data []byte) error {
		// Record the message
		size := uint64(len(data))
		mr.Stats.IncrementReceived(size)

		// Create a monitoring event
		event := MonitoringEvent{
			Type:       "receive_with_context",
			Pipe:       pipe,
			Timestamp:  time.Now(),
			Size:       size,
			RouterName: mr.Name,
		}

		// Call the monitoring callback
		mr.MonitoringCallback(event)

		// Call the original callback
		err := callback(data)
		if err != nil {
			// Record the error
			mr.Stats.IncrementError()
			// Create a monitoring event
			errorEvent := MonitoringEvent{
				Type:       "receive_with_context_error",
				Pipe:       pipe,
				Timestamp:  time.Now(),
				Size:       size,
				Error:      err.Error(),
				RouterName: mr.Name,
			}
			// Call the monitoring callback
			mr.MonitoringCallback(errorEvent)
		}

		return err
	}

	if mr.ContextRouter != nil {
		return mr.ContextRouter.SubscribeWithContext(ctx, pipe, wrapper)
	}
	return mr.Router.Subscribe(pipe, wrapper)
}

// SetMonitoringEnabled sets whether monitoring is enabled.
func (mr *MonitoredRouter) SetMonitoringEnabled(enabled bool) {
	mr.MonitoringEnabled = enabled
}

// SetMonitoringCallback sets the callback function for monitoring events.
func (mr *MonitoredRouter) SetMonitoringCallback(callback func(MonitoringEvent)) {
	mr.MonitoringCallback = callback
}

// GetStats returns the message statistics.
func (mr *MonitoredRouter) GetStats() *MessageStats {
	return mr.Stats
}

// ResetStats resets the message statistics.
func (mr *MonitoredRouter) ResetStats() {
	mr.Stats.Reset()
}

// defaultMonitoringCallback is the default callback for monitoring events.
func defaultMonitoringCallback(event MonitoringEvent) {
	log.Printf("Monitoring event: %s, Pipe: %s, Size: %d bytes, Router: %s",
		event.Type, event.Pipe, event.Size, event.RouterName)
}

// MonitoringServer is a HTTP server that exposes monitoring information.
type MonitoringServer struct {
	// Routers is a map of router names to monitored routers.
	Routers map[string]*MonitoredRouter
	// Server is the HTTP server.
	Server *http.Server
	// mutex protects the Routers map.
	mutex sync.RWMutex
}

// NewMonitoringServer creates a new monitoring server.
func NewMonitoringServer(addr string) *MonitoringServer {
	ms := &MonitoringServer{
		Routers: make(map[string]*MonitoredRouter),
	}

	// Create a new HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/stats", ms.handleStats)
	mux.HandleFunc("/stats/", ms.handleRouterStats)
	mux.HandleFunc("/reset", ms.handleReset)
	mux.HandleFunc("/reset/", ms.handleRouterReset)

	ms.Server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return ms
}

// RegisterRouter registers a monitored router with the server.
func (ms *MonitoringServer) RegisterRouter(name string, router *MonitoredRouter) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.Routers[name] = router
}

// UnregisterRouter unregisters a monitored router from the server.
func (ms *MonitoringServer) UnregisterRouter(name string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	delete(ms.Routers, name)
}

// Start starts the monitoring server.
func (ms *MonitoringServer) Start() error {
	return ms.Server.ListenAndServe()
}

// Stop stops the monitoring server.
func (ms *MonitoringServer) Stop(ctx context.Context) error {
	return ms.Server.Shutdown(ctx)
}

// handleStats handles requests for all router statistics.
func (ms *MonitoringServer) handleStats(w http.ResponseWriter, r *http.Request) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	// Collect statistics for all routers
	stats := make(map[string]map[string]interface{})
	for name, router := range ms.Routers {
		stats[name] = router.GetStats().GetStats()
	}

	// Return the statistics as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleRouterStats handles requests for a specific router's statistics.
func (ms *MonitoringServer) handleRouterStats(w http.ResponseWriter, r *http.Request) {
	// Extract the router name from the URL
	name := r.URL.Path[len("/stats/"):]
	if name == "" {
		http.Error(w, "Router name is required", http.StatusBadRequest)
		return
	}

	// Get the router
	ms.mutex.RLock()
	router, exists := ms.Routers[name]
	ms.mutex.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Router %s not found", name), http.StatusNotFound)
		return
	}

	// Return the router's statistics as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(router.GetStats().GetStats())
}

// handleReset handles requests to reset all router statistics.
func (ms *MonitoringServer) handleReset(w http.ResponseWriter, r *http.Request) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	// Reset statistics for all routers
	for _, router := range ms.Routers {
		router.ResetStats()
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("All router statistics reset"))
}

// handleRouterReset handles requests to reset a specific router's statistics.
func (ms *MonitoringServer) handleRouterReset(w http.ResponseWriter, r *http.Request) {
	// Extract the router name from the URL
	name := r.URL.Path[len("/reset/"):]
	if name == "" {
		http.Error(w, "Router name is required", http.StatusBadRequest)
		return
	}

	// Get the router
	ms.mutex.RLock()
	router, exists := ms.Routers[name]
	ms.mutex.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Router %s not found", name), http.StatusNotFound)
		return
	}

	// Reset the router's statistics
	router.ResetStats()

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Router %s statistics reset", name)))
}
