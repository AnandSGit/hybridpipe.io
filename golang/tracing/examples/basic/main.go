package main

import (
	"context"
	"log"
	"time"

	hp "github.com/AnandSGit/hybridpipe.io"
)

func main() {
	// Initialize the tracer
	tracer, err := hp.NewTracer(hp.Config{
		ServiceName:  "example-service",
		SamplingRate: 0.1, // Sample 10% of messages
		Exporter: hp.ExporterConfig{
			Type:     "console", // Use console exporter for demonstration
			Endpoint: "",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	// Initialize the meter
	meter, err := hp.NewMeter(hp.MonitoringConfig{
		ServiceName: "example-service",
		Exporter: hp.MonitoringExporterConfig{
			Type:     "console", // Use console exporter for demonstration
			Endpoint: "",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create meter: %v", err)
	}

	// Create a HybridPipe router
	router, err := hp.DeployRouter(hp.KAFKA)
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	// Wrap the router with tracing and metrics middleware
	tracedRouter := hp.WrapRouterWithMetrics(router, tracer, meter)

	// Define a message handler with tracing
	handleMessage := func(data any) {
		log.Printf("Received message: %v", data)
	}

	// Subscribe to a pipe
	err = tracedRouter.Accept("example-pipe", handleMessage)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Send a message
	_, span := tracer.Start(context.Background(), "send-message")
	message := map[string]any{
		"content":   "Hello, world!",
		"timestamp": time.Now().Unix(),
	}
	err = tracedRouter.Dispatch("example-pipe", message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(2, err.Error()) // 2 is the status code for error
		log.Fatalf("Failed to send message: %v", err)
	}
	span.End()

	// Wait for the message to be processed
	time.Sleep(1 * time.Second)

	// Close the router
	tracedRouter.Close()
}
