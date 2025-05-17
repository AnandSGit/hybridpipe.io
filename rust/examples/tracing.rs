// Example demonstrating the use of the tracing functionality.

use hybridpipe::{
    deploy_router, BrokerType, InMemoryTracer, Process, Result, Trace, TraceStatus, TracingMiddleware,
};
use log::{error, info};
use std::sync::Arc;
use std::time::Duration;
use tokio::time;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    env_logger::init();

    info!("HybridPipe Tracing Example");

    // Create a tracer
    let tracer = Arc::new(InMemoryTracer::new(100));

    // Deploy a router
    let router = deploy_router(BrokerType::MOCK)?;

    // Create a tracing middleware
    let tracing_router = Arc::new(TracingMiddleware::new_with_broker_type(
        router,
        tracer.clone(),
        BrokerType::MOCK,
    ));

    // Connect to the router
    tracing_router.connect().await?;

    // Subscribe to a pipe
    let callback: Process = Box::new(|data| {
        Box::pin(async move {
            // Simulate processing
            time::sleep(Duration::from_millis(100)).await;
            info!("Received message: {:?}", data);
            Ok(())
        })
    });
    tracing_router.subscribe("test", callback).await?;

    // Dispatch a message
    let message = "Hello, Tracing!".as_bytes().to_vec();
    tracing_router.dispatch("test", Box::new(message)).await?;

    // Wait for the message to be processed
    time::sleep(Duration::from_millis(200)).await;

    // Get the traces
    let traces = tracer.get_traces();
    info!("Collected {} traces", traces.len());

    // Print the traces
    for trace in traces {
        print_trace(&trace);
    }

    // Disconnect
    tracing_router.disconnect().await?;

    Ok(())
}

fn print_trace(trace: &Arc<Trace>) {
    let status = match trace.status {
        TraceStatus::Success => "SUCCESS",
        TraceStatus::Error => "ERROR",
    };
    let duration = trace.duration_ms.unwrap_or(0);
    info!(
        "Trace: {} - {} - {} - {} - {}ms",
        trace.id, trace.protocol, trace.pipe, status, duration
    );
    if let Some(error) = &trace.error {
        error!("  Error: {}", error);
    }
    for (key, value) in &trace.tags {
        info!("  Tag: {} = {}", key, value);
    }
}
