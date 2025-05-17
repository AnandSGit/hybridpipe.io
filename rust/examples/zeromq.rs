// Example demonstrating the use of the ZeroMQ functionality.

use hybridpipe::{deploy_router, BrokerType, Process, Result};
use log::{error, info};
use std::sync::Arc;
use std::time::Duration;
use tokio::time;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    env_logger::init();

    info!("HybridPipe ZeroMQ Example");

    // Deploy a router
    let router = deploy_router(BrokerType::ZEROMQ)?;

    // Connect to the router
    router.connect().await?;

    // Subscribe to a pipe
    let callback: Process = Box::new(|data| {
        Box::pin(async move {
            // Simulate processing
            time::sleep(Duration::from_millis(100)).await;
            info!("Received message: {:?}", data);
            Ok(())
        })
    });
    router.subscribe("test", callback).await?;

    // Dispatch a message
    let message = "Hello, ZeroMQ!".as_bytes().to_vec();
    router.dispatch("test", Box::new(message)).await?;

    // Wait for the message to be processed
    time::sleep(Duration::from_millis(200)).await;

    // Disconnect
    router.disconnect().await?;

    Ok(())
}
