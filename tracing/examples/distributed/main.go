package main

import (
	"context"
	"log"
	"sync"
	"time"

	hp "github.com/AnandSGit/hybridpipe.io"
)

func main() {
	// Start the services
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		serviceA()
	}()

	// Wait for service A to start
	time.Sleep(500 * time.Millisecond)

	go func() {
		defer wg.Done()
		serviceB()
	}()

	// Wait for service B to start
	time.Sleep(500 * time.Millisecond)

	go func() {
		defer wg.Done()
		serviceC()
	}()

	// Wait for all services to complete
	wg.Wait()
}

// serviceA is the first service in the chain
func serviceA() {
	// Initialize the tracer
	tracer, err := hp.NewTracer(hp.Config{
		ServiceName:  "service-a",
		SamplingRate: 1.0, // Sample all messages
		Exporter: hp.ExporterConfig{
			Type:     "console", // Use console exporter for demonstration
			Endpoint: "",
		},
	})
	if err != nil {
		log.Fatalf("Service A: Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	// Create a HybridPipe router
	router, err := hp.DeployRouter(hp.KAFKA)
	if err != nil {
		log.Fatalf("Service A: Failed to create router: %v", err)
	}

	// Wrap the router with tracing middleware
	tracedRouter := hp.WrapRouter(router, tracer)

	// Start a new trace
	ctx, span := tracer.Start(context.Background(), "process-request")
	defer span.End()

	// Create a message with trace context
	message := map[string]any{
		"content":   "Hello from Service A!",
		"timestamp": time.Now().Unix(),
	}

	// Inject trace context into the message
	tracer.Inject(ctx, message)

	// Send the message to Service B
	log.Println("Service A: Sending message to Service B")
	err = tracedRouter.Dispatch("service-b", message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(2, err.Error()) // 2 is the status code for error
		log.Fatalf("Service A: Failed to send message: %v", err)
	}

	// Wait for the entire trace to complete
	time.Sleep(3 * time.Second)

	// Close the router
	tracedRouter.Close()
}

// serviceB is the second service in the chain
func serviceB() {
	// Initialize the tracer
	tracer, err := hp.NewTracer(hp.Config{
		ServiceName:  "service-b",
		SamplingRate: 1.0, // Sample all messages
		Exporter: hp.ExporterConfig{
			Type:     "console", // Use console exporter for demonstration
			Endpoint: "",
		},
	})
	if err != nil {
		log.Fatalf("Service B: Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	// Create a HybridPipe router
	router, err := hp.DeployRouter(hp.KAFKA)
	if err != nil {
		log.Fatalf("Service B: Failed to create router: %v", err)
	}

	// Wrap the router with tracing middleware
	tracedRouter := hp.WrapRouter(router, tracer)

	// Define a message handler with tracing
	handleMessage := func(data any) {
		// Extract trace context from the message
		ctx := tracer.Extract(context.Background(), data)

		// Start a new span as part of the same trace
		ctx, span := tracer.Start(ctx, "process-message")
		defer span.End()

		// Process the message
		message := data.(map[string]any)
		log.Printf("Service B: Received message: %v", message["content"])

		// Add some processing time
		time.Sleep(500 * time.Millisecond)

		// Modify the message
		message["content"] = "Hello from Service B!"
		message["timestamp"] = time.Now().Unix()

		// Inject trace context into the message
		tracer.Inject(ctx, message)

		// Forward the message to Service C
		log.Println("Service B: Forwarding message to Service C")
		err := tracedRouter.Dispatch("service-c", message)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(2, err.Error()) // 2 is the status code for error
			log.Printf("Service B: Failed to forward message: %v", err)
		}
	}

	// Subscribe to messages from Service A
	err = tracedRouter.Accept("service-b", handleMessage)
	if err != nil {
		log.Fatalf("Service B: Failed to subscribe: %v", err)
	}

	// Keep the service running
	select {}
}

// serviceC is the third service in the chain
func serviceC() {
	// Initialize the tracer
	tracer, err := hp.NewTracer(hp.Config{
		ServiceName:  "service-c",
		SamplingRate: 1.0, // Sample all messages
		Exporter: hp.ExporterConfig{
			Type:     "console", // Use console exporter for demonstration
			Endpoint: "",
		},
	})
	if err != nil {
		log.Fatalf("Service C: Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	// Create a HybridPipe router
	router, err := hp.DeployRouter(hp.KAFKA)
	if err != nil {
		log.Fatalf("Service C: Failed to create router: %v", err)
	}

	// Wrap the router with tracing middleware
	tracedRouter := hp.WrapRouter(router, tracer)

	// Define a message handler with tracing
	handleMessage := func(data any) {
		// Extract trace context from the message
		ctx := tracer.Extract(context.Background(), data)

		// Start a new span as part of the same trace
		ctx, span := tracer.Start(ctx, "process-message")
		defer span.End()

		// Process the message
		message := data.(map[string]any)
		log.Printf("Service C: Received message: %v", message["content"])

		// Add some processing time
		time.Sleep(500 * time.Millisecond)

		// Add an event to the span
		span.AddEvent("message-processed", map[string]any{
			"timestamp": time.Now().Unix(),
		})
	}

	// Subscribe to messages from Service B
	err = tracedRouter.Accept("service-c", handleMessage)
	if err != nil {
		log.Fatalf("Service C: Failed to subscribe: %v", err)
	}

	// Keep the service running
	select {}
}
