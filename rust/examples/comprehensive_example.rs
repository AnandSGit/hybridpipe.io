use hybridpipe::{
    deploy_router, init, BrokerType, HybridPipe, LoggingMiddleware, MiddlewareStack,
    MonitoringMiddleware, Tracer, TracingMiddleware, new_message_stats,
};
use serde::{Serialize, Deserialize};
use std::sync::Arc;
use tokio::signal;
use tokio::time::{sleep, Duration};
use log::{info, error};

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Person {
    name: String,
    age: u32,
    next_gen: Vec<String>,
    c_age: Vec<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct StatusUpdate {
    status: String,
    timestamp: u64,
    source: String,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logging
    env_logger::init();
    
    // Initialize HybridPipe
    init();
    
    info!("Starting HybridPipe Comprehensive Example");
    
    // Create monitoring and tracing components
    let stats = new_message_stats();
    let tracer = Arc::new(Tracer::new("comprehensive-example"));
    
    // Create middleware
    let monitoring_middleware = MonitoringMiddleware::new(stats.clone());
    let logging_middleware = LoggingMiddleware::new();
    let tracing_middleware = TracingMiddleware::new(tracer.clone());
    
    // Deploy a mock router for testing
    let router = deploy_router(BrokerType::MOCK)?;
    
    // Create a middleware stack
    let mut middleware_stack = MiddlewareStack::new(router);
    middleware_stack.add_middleware(monitoring_middleware);
    middleware_stack.add_middleware(logging_middleware);
    middleware_stack.add_middleware(tracing_middleware);
    
    // Connect to the messaging system
    middleware_stack.connect().await?;
    info!("Connected to mock messaging system");
    
    // Start a trace
    let trace_id = tracer.start_trace("comprehensive-example");
    
    // Subscribe to person updates
    middleware_stack.accept("app.users.events", |data| {
        if let Some(person) = data.downcast_ref::<Person>() {
            info!("Received person: {}, age: {}", person.name, person.age);
            info!("Next generation: {:?}", person.next_gen);
            info!("Next generation ages: {:?}", person.c_age);
        } else {
            error!("Received data of unknown type on app.users.events");
        }
    }).await?;
    info!("Subscribed to pipe: app.users.events");
    
    // Subscribe to status updates
    middleware_stack.accept("app.status.events", |data| {
        if let Some(status) = data.downcast_ref::<StatusUpdate>() {
            info!("Received status update: {} from {} at {}", status.status, status.source, status.timestamp);
        } else {
            error!("Received data of unknown type on app.status.events");
        }
    }).await?;
    info!("Subscribed to pipe: app.status.events");
    
    // Create and send a person message
    let person = Person {
        name: "David Gower".to_string(),
        age: 75,
        next_gen: vec!["Pringle".to_string(), "NH Fairbrother".to_string(), "Wasim".to_string()],
        c_age: vec![45, 37, 39],
    };
    
    middleware_stack.dispatch("app.users.events", Box::new(person)).await?;
    info!("Sent person message to pipe: app.users.events");
    
    // Create and send a status update
    let status = StatusUpdate {
        status: "Online".to_string(),
        timestamp: std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs(),
        source: "HybridPipe Example".to_string(),
    };
    
    middleware_stack.dispatch("app.status.events", Box::new(status)).await?;
    info!("Sent status update to pipe: app.status.events");
    
    // Wait a bit to see the output
    sleep(Duration::from_secs(1)).await;
    
    // Print monitoring stats
    let dispatch_metrics = stats.get_dispatch_metrics();
    info!("Dispatch metrics:");
    for (pipe, metrics) in dispatch_metrics {
        info!("  Pipe: {}", pipe);
        info!("    Total: {}", metrics.total_count);
        info!("    Success: {}", metrics.success_count);
        info!("    Error: {}", metrics.error_count);
        if let Some(min_time) = metrics.min_time {
            info!("    Min time: {:?}", min_time);
        }
        if let Some(max_time) = metrics.max_time {
            info!("    Max time: {:?}", max_time);
        }
    }
    
    let process_metrics = stats.get_process_metrics();
    info!("Process metrics:");
    for (pipe, metrics) in process_metrics {
        info!("  Pipe: {}", pipe);
        info!("    Total: {}", metrics.total_count);
        info!("    Success: {}", metrics.success_count);
        info!("    Error: {}", metrics.error_count);
        info!("    Total size: {} bytes", metrics.total_size);
        if let Some(min_time) = metrics.min_time {
            info!("    Min time: {:?}", min_time);
        }
        if let Some(max_time) = metrics.max_time {
            info!("    Max time: {:?}", max_time);
        }
    }
    
    // Print trace information
    let trace = tracer.get_trace(&trace_id).unwrap();
    info!("Trace: {}", trace.name);
    info!("  ID: {}", trace.id);
    info!("  Start time: {:?}", trace.start_time);
    info!("  Spans: {}", trace.spans.len());
    for span in &trace.spans {
        info!("    Span: {}", span.name);
        info!("      ID: {}", span.id);
        info!("      Start time: {:?}", span.start_time);
        if let Some(end_time) = span.end_time {
            info!("      End time: {:?}", end_time);
        }
        info!("      Success: {}", span.success);
        if !span.attributes.is_empty() {
            info!("      Attributes:");
            for (key, value) in &span.attributes {
                info!("        {}: {}", key, value);
            }
        }
    }
    
    // End the trace
    tracer.end_trace(&trace_id);
    
    // Clean up
    middleware_stack.disconnect().await?;
    info!("Disconnected from mock messaging system");
    
    Ok(())
}
