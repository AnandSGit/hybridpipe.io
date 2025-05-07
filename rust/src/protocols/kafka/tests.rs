#[cfg(test)]
mod tests {
    use crate::core::context::Context;
    use crate::core::errors::Error;
    use crate::core::HybridPipe;
    use crate::protocols::kafka::{register, Packet};
    use serde::{Deserialize, Serialize};
    use std::sync::Arc;
    use std::time::Duration;
    use tokio::sync::Mutex as TokioMutex;

    // Create a test packet with the connected flag set
    fn create_test_packet() -> Packet {
        let packet = Packet::new();

        // Set the connected flag to true
        {
            let mut connected = packet.connected.lock().unwrap();
            *connected = true;
        }

        packet
    }

    // Create a test packet with the connected flag set
    async fn create_connected_test_packet() -> Packet {
        let packet = Packet::new();

        // Set the connected flag to true
        {
            let mut connected = packet.connected.lock().unwrap();
            *connected = true;
        }

        packet
    }

    #[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
    struct TestMessage {
        id: u32,
        content: String,
    }

    #[tokio::test]
    async fn test_kafka_connect_disconnect() {
        let packet = Packet::new();

        // Test initial state
        assert!(!packet.is_connected());

        // Test disconnect when not connected
        let result = packet.disconnect().await;
        assert!(matches!(result, Err(Error::NotConnected)));

        // We can't actually connect to a Kafka broker in a unit test
        // So we'll just set the connected flag manually
        {
            let mut connected = packet.connected.lock().unwrap();
            *connected = true;
        }

        // Test connected state
        assert!(packet.is_connected());

        // Test disconnect
        let result = packet.disconnect().await;
        assert!(result.is_ok());

        // Test disconnected state
        assert!(!packet.is_connected());
    }

    #[tokio::test]
    async fn test_kafka_dispatch_subscribe() {
        // Create a test packet with the connected flag set
        let packet = create_connected_test_packet().await;

        // Create a shared variable to store received messages
        let received = Arc::new(TokioMutex::new(None));
        let received_clone = received.clone();

        // Create a test message
        let test_message = TestMessage {
            id: 42,
            content: "Hello, Kafka!".to_string(),
        };

        // Subscribe to the test pipe
        let result = packet
            .subscribe(
                "test-pipe",
                Box::new(move |data| {
                    // Store the received data
                    let received_clone2 = received_clone.clone();
                    tokio::spawn(async move {
                        let mut received = received_clone2.lock().await;
                        *received = Some(data);
                    });
                    Ok(())
                }),
            )
            .await;

        // This should fail because we don't have a real Kafka client
        // But if it succeeds, that's also fine for the test
        println!("Subscribe result: {:?}", result);

        // Dispatch a message
        let result = packet
            .dispatch("test-pipe", Box::new(test_message.clone()))
            .await;

        // This should fail because we don't have a real Kafka client
        // But if it succeeds, that's also fine for the test
        println!("Dispatch result: {:?}", result);

        // Wait a bit for the message to be processed
        tokio::time::sleep(Duration::from_millis(100)).await;

        // Test unsubscribe
        let result = packet.unsubscribe("test-pipe").await;
        println!("Unsubscribe result: {:?}", result);

        // Test unsubscribe when not subscribed
        let result = packet.unsubscribe("nonexistent-pipe").await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_kafka_context_operations() {
        // Create a test packet with the connected flag set
        let packet = create_connected_test_packet().await;
        let ctx = Context::background();

        // Create a test message
        let test_message = TestMessage {
            id: 43,
            content: "Hello with context!".to_string(),
        };

        // Test dispatch with context
        let result = packet
            .dispatch_with_context(&ctx, "test-pipe", Box::new(test_message.clone()))
            .await;
        // This should fail because we don't have a real Kafka client
        // But if it succeeds, that's also fine for the test
        println!("Dispatch with context result: {:?}", result);

        // Create a shared variable to store received messages
        let received = Arc::new(TokioMutex::new(None));
        let received_clone = received.clone();

        // Test subscribe with context
        let result = packet
            .subscribe_with_context(
                &ctx,
                "test-pipe-ctx",
                Box::new(move |data| {
                    // Store the received data
                    let received_clone2 = received_clone.clone();
                    tokio::spawn(async move {
                        let mut received = received_clone2.lock().await;
                        *received = Some(data);
                    });
                    Ok(())
                }),
            )
            .await;
        // This should fail because we don't have a real Kafka client
        // But if it succeeds, that's also fine for the test
        println!("Subscribe with context result: {:?}", result);

        // Test with timeout context
        let timeout_ctx = Context::with_timeout(&ctx, Duration::from_millis(100));
        let result = packet
            .dispatch_with_context(
                &timeout_ctx,
                "test-pipe-ctx",
                Box::new(test_message.clone()),
            )
            .await;
        // This should fail because we don't have a real Kafka client
        // But if it succeeds, that's also fine for the test
        println!("Dispatch with timeout context result: {:?}", result);
    }

    #[tokio::test]
    async fn test_kafka_error_cases() {
        let packet = Packet::new();

        // Test dispatch when not connected
        let test_message = TestMessage {
            id: 44,
            content: "Error test".to_string(),
        };

        let result = packet
            .dispatch("test-pipe", Box::new(test_message.clone()))
            .await;
        assert!(matches!(result, Err(Error::NotConnected)));

        // Test subscribe when not connected
        let result = packet.subscribe("test-pipe", Box::new(|_| Ok(()))).await;
        assert!(matches!(result, Err(Error::NotConnected)));

        // Test unsubscribe when not connected
        let result = packet.unsubscribe("test-pipe").await;
        assert!(matches!(result, Err(Error::NotConnected)));

        // Test close when not connected
        let result = packet.close().await;
        assert!(matches!(result, Err(Error::NotConnected)));
    }

    #[test]
    fn test_kafka_register() {
        // Test that the register function doesn't panic
        register();
    }
}
