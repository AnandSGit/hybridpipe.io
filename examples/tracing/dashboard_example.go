package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	hp "github.com/AnandSGit/hybridpipe.io"
)

// Global variables for the dashboard example
var (
	// tracer is the global tracer
	dashboardTracer hp.Tracer
	// meter is the global meter
	dashboardMeter hp.Meter
	// storage is the global storage
	dashboardStorage hp.Storage
	// tracedRouter is the global traced router
	dashboardRouter hp.HybridPipe
)

// DashboardExample demonstrates a web dashboard for tracing
func DashboardExample() {
	// Initialize the tracer
	var err error
	dashboardTracer, err = hp.NewTracer(hp.Config{
		ServiceName: "dashboard-example",
		SamplingRate: 1.0, // Sample all messages
		Storage: hp.StorageConfig{
			Type: "memory",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create tracer: %v", err)
	}
	defer dashboardTracer.Shutdown(context.Background())

	// Initialize the meter
	dashboardMeter, err = hp.NewMeter(hp.MonitoringConfig{
		ServiceName: "dashboard-example",
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
	dashboardRouter = hp.WrapRouterWithMetrics(router, dashboardTracer, dashboardMeter)

	// Define a message handler with tracing
	handleMessage := func(data interface{}) {
		log.Printf("Received message: %v", data)
	}

	// Subscribe to a pipe
	err = dashboardRouter.Accept("example-pipe", handleMessage)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Start a goroutine to generate sample traces
	go generateSampleTraces()

	// Set up HTTP server for the dashboard
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/traces", handleTraces)
	http.HandleFunc("/trace", handleTrace)
	http.HandleFunc("/metrics", handleMetrics)

	// Start the HTTP server
	log.Println("Starting dashboard server on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// generateSampleTraces generates sample traces for demonstration
func generateSampleTraces() {
	// Create a ticker to generate traces periodically
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Generate traces
	for range ticker.C {
		// Start a new trace
		ctx, span := dashboardTracer.Start(context.Background(), "process-request")

		// Set span attributes
		span.SetAttribute("request_id", fmt.Sprintf("%d", time.Now().UnixNano()))
		span.SetAttribute("user_id", "user-123")

		// Create a child span
		ctx, childSpan := dashboardTracer.Start(ctx, "validate-request")
		childSpan.SetAttribute("validation_result", "success")
		time.Sleep(100 * time.Millisecond)
		childSpan.End()

		// Create another child span
		ctx, childSpan = dashboardTracer.Start(ctx, "process-data")
		childSpan.SetAttribute("data_size", 1024)
		time.Sleep(200 * time.Millisecond)

		// Create a nested child span
		ctx, nestedSpan := dashboardTracer.Start(ctx, "database-query")
		nestedSpan.SetAttribute("query", "SELECT * FROM users")
		time.Sleep(150 * time.Millisecond)
		nestedSpan.End()

		childSpan.End()

		// Send a message
		message := map[string]interface{}{
			"content":   "Hello, world!",
			"timestamp": time.Now().Unix(),
		}

		// Inject trace context into the message
		dashboardTracer.Inject(ctx, message)

		// Dispatch the message
		err := dashboardRouter.Dispatch("example-pipe", message)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(2, err.Error())
			log.Printf("Failed to send message: %v", err)
		}

		// End the root span
		span.End()
	}
}

// handleDashboard handles the dashboard page
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Define the dashboard template
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>HybridPipe Tracing Dashboard</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
        }
        h1 {
            color: #333;
        }
        .container {
            display: flex;
            flex-wrap: wrap;
        }
        .card {
            background-color: #f9f9f9;
            border-radius: 5px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
            margin: 10px;
            padding: 20px;
            width: 300px;
        }
        .card h2 {
            margin-top: 0;
        }
        table {
            border-collapse: collapse;
            width: 100%;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 8px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
    </style>
    <script>
        // Refresh the page every 10 seconds
        setTimeout(function() {
            location.reload();
        }, 10000);

        // Load traces
        function loadTraces() {
            fetch('/traces')
                .then(response => response.json())
                .then(data => {
                    const table = document.getElementById('traces-table');
                    table.innerHTML = '<tr><th>Trace ID</th><th>Name</th><th>Start Time</th><th>Duration</th><th>Status</th></tr>';
                    
                    data.forEach(trace => {
                        const row = document.createElement('tr');
                        row.innerHTML = '<td><a href="/trace?id=' + trace.traceId + '">' + trace.traceId.substring(0, 8) + '...</a></td>' +
                            '<td>' + trace.name + '</td>' +
                            '<td>' + new Date(trace.startTime).toLocaleTimeString() + '</td>' +
                            '<td>' + trace.duration + ' ms</td>' +
                            '<td>' + trace.status + '</td>';
                        table.appendChild(row);
                    });
                });
        }

        // Load metrics
        function loadMetrics() {
            fetch('/metrics')
                .then(response => response.json())
                .then(data => {
                    const container = document.getElementById('metrics-container');
                    container.innerHTML = '';
                    
                    Object.entries(data).forEach(([name, value]) => {
                        const card = document.createElement('div');
                        card.className = 'card';
                        card.innerHTML = '<h2>' + name + '</h2><p>Value: ' + value + '</p>';
                        container.appendChild(card);
                    });
                });
        }

        // Load data when the page loads
        window.onload = function() {
            loadTraces();
            loadMetrics();
        };
    </script>
</head>
<body>
    <h1>HybridPipe Tracing Dashboard</h1>
    
    <h2>Recent Traces</h2>
    <table id="traces-table">
        <tr>
            <th>Trace ID</th>
            <th>Name</th>
            <th>Start Time</th>
            <th>Duration</th>
            <th>Status</th>
        </tr>
    </table>
    
    <h2>Metrics</h2>
    <div id="metrics-container" class="container">
    </div>
</body>
</html>
`

	// Execute the template
	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleTraces handles the traces API endpoint
func handleTraces(w http.ResponseWriter, r *http.Request) {
	// Query for traces
	traces, err := dashboardStorage.Query(hp.Query{
		Limit: 10,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert traces to a format suitable for the dashboard
	type TraceInfo struct {
		TraceID   string `json:"traceId"`
		Name      string `json:"name"`
		StartTime int64  `json:"startTime"`
		Duration  int64  `json:"duration"`
		Status    string `json:"status"`
	}

	traceInfos := make([]TraceInfo, 0, len(traces))
	for _, trace := range traces {
		status := "OK"
		if trace.Status.Code == 2 {
			status = "Error"
		}

		traceInfos = append(traceInfos, TraceInfo{
			TraceID:   trace.TraceID,
			Name:      trace.Name,
			StartTime: trace.StartTime.UnixNano() / 1000000,
			Duration:  trace.EndTime.Sub(trace.StartTime).Milliseconds(),
			Status:    status,
		})
	}

	// Return the trace information as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(traceInfos)
}

// handleTrace handles the trace details API endpoint
func handleTrace(w http.ResponseWriter, r *http.Request) {
	// Get the trace ID from the query parameters
	traceID := r.URL.Query().Get("id")
	if traceID == "" {
		http.Error(w, "Trace ID is required", http.StatusBadRequest)
		return
	}

	// Query for the trace
	spans, err := dashboardStorage.Query(hp.Query{
		TraceID: traceID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the spans as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spans)
}

// handleMetrics handles the metrics API endpoint
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	// This is a simplified implementation
	// In a real implementation, we would query the meter for metrics
	metrics := map[string]interface{}{
		"hybridpipe_messages_sent_total":     42,
		"hybridpipe_messages_received_total": 42,
		"hybridpipe_message_size_bytes":      1024,
		"hybridpipe_message_latency_seconds": 0.123,
		"hybridpipe_errors_total":            0,
	}

	// Return the metrics as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
