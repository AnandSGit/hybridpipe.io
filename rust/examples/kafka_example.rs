use hybridpipe::{deploy_router, BrokerType};
use serde::{Serialize, Deserialize};
use std::sync::Arc;
use tokio::signal;
use log::{info, error};

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Person {
    name: String,
    age: u32,
    next_gen: Vec<String>,
    c_age: Vec<u32>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logging
    env_logger::init();
    
    info!("Starting HybridPipe Kafka Example");
    
    // Connect to Kafka
    let router = deploy_router(BrokerType::KAFKA)?;
    router.connect().await?;
    info!("Connected to Kafka");
    
    // Create a message
    let person = Person {
        name: "David Gower".to_string(),
        age: 75,
        next_gen: vec!["Pringle".to_string(), "NH Fairbrother".to_string(), "Wasim".to_string()],
        c_age: vec![45, 37, 39],
    };
    
    // Subscribe to a pipe
    router.accept("app.users.events", |data| {
        if let Some(person) = data.downcast_ref::<Person>() {
            info!("Received person: {}, age: {}", person.name, person.age);
            info!("Next generation: {:?}", person.next_gen);
            info!("Next generation ages: {:?}", person.c_age);
        } else {
            error!("Received data of unknown type");
        }
    }).await?;
    info!("Subscribed to pipe: app.users.events");
    
    // Send the message
    router.dispatch("app.users.events", Box::new(person)).await?;
    info!("Sent message to pipe: app.users.events");
    
    // Wait for Ctrl+C
    info!("Press Ctrl+C to exit");
    signal::ctrl_c().await?;
    
    // Clean up
    router.disconnect().await?;
    info!("Disconnected from Kafka");
    
    Ok(())
}
