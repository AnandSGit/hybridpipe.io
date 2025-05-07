#[cfg(test)]
mod tests {
    use serde::{Deserialize, Serialize};
    use std::sync::Arc;
    use std::time::Duration;
    use tokio::sync::Mutex;

    use crate::core::{deploy_router, ext, BrokerType, HybridPipe};
    use crate::protocols::mock::Packet;
    use crate::serialization;

    #[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
    struct TestMessage {
        id: u32,
        content: String,
    }

    #[tokio::test]
    async fn test_mock_connect_disconnect() {
        let mock = Arc::new(Packet::new());

        // Test connect
        assert!(!mock.is_connected());
        mock.connect().await.expect("Failed to connect");
        assert!(mock.is_connected());

        // Test disconnect
        mock.disconnect().await.expect("Failed to disconnect");
        assert!(!mock.is_connected());
    }

    #[tokio::test]
    async fn test_mock_dispatch_subscribe() {
        let mock = Arc::new(Packet::new());
        mock.connect().await.expect("Failed to connect");

        // Create a test message
        let test_msg = TestMessage {
            id: 42,
            content: "Hello, HybridPipe!".to_string(),
        };

        // Create a shared variable to store the received message
        let received = Arc::new(Mutex::new(None));
        let received_clone = received.clone();

        // Subscribe to a pipe
        let received_clone2 = received_clone.clone();
        ext::accept::<TestMessage, _>(&*mock, "test-pipe", move |data| {
            if let Some(msg) = data.downcast_ref::<TestMessage>() {
                let msg_clone = msg.clone();
                let received_clone3 = received_clone2.clone();
                // Use a non-blocking approach
                tokio::spawn(async move {
                    let mut received = received_clone3.lock().await;
                    *received = Some(msg_clone);
                });
            }
        })
        .await
        .expect("Failed to subscribe");

        // Dispatch a message
        let serialized = serialization::encode(&test_msg).unwrap();
        mock.dispatch("test-pipe", Box::new(serialized.to_vec()))
            .await
            .expect("Failed to dispatch");

        // Wait a bit for the message to be processed
        tokio::time::sleep(Duration::from_millis(100)).await;

        // Check if the message was received
        let received_msg = received.lock().await.clone();
        assert!(received_msg.is_some());
        assert_eq!(received_msg.unwrap(), test_msg);

        // Unsubscribe
        mock.unsubscribe("test-pipe")
            .await
            .expect("Failed to unsubscribe");

        // Clean up
        mock.disconnect().await.expect("Failed to disconnect");
    }

    #[tokio::test]
    async fn test_mock_multiple_subscribers() {
        let mock = Arc::new(Packet::new());
        mock.connect().await.expect("Failed to connect");

        // Create test messages
        let test_msg1 = TestMessage {
            id: 1,
            content: "Message 1".to_string(),
        };

        let test_msg2 = TestMessage {
            id: 2,
            content: "Message 2".to_string(),
        };

        // Create shared variables to store the received messages
        let received1 = Arc::new(Mutex::new(None));
        let received1_clone = received1.clone();

        let received2 = Arc::new(Mutex::new(None));
        let received2_clone = received2.clone();

        // Subscribe to pipes
        let received1_clone2 = received1_clone.clone();
        ext::accept::<TestMessage, _>(&*mock, "pipe1", move |data| {
            if let Some(msg) = data.downcast_ref::<TestMessage>() {
                let msg_clone = msg.clone();
                let received1_clone3 = received1_clone2.clone();
                // Use a non-blocking approach
                tokio::spawn(async move {
                    let mut received = received1_clone3.lock().await;
                    *received = Some(msg_clone);
                });
            }
        })
        .await
        .expect("Failed to subscribe to pipe1");

        let received2_clone2 = received2_clone.clone();
        ext::accept::<TestMessage, _>(&*mock, "pipe2", move |data| {
            if let Some(msg) = data.downcast_ref::<TestMessage>() {
                let msg_clone = msg.clone();
                let received2_clone3 = received2_clone2.clone();
                // Use a non-blocking approach
                tokio::spawn(async move {
                    let mut received = received2_clone3.lock().await;
                    *received = Some(msg_clone);
                });
            }
        })
        .await
        .expect("Failed to subscribe to pipe2");

        // Dispatch messages
        let serialized1 = serialization::encode(&test_msg1).unwrap();
        mock.dispatch("pipe1", Box::new(serialized1.to_vec()))
            .await
            .expect("Failed to dispatch to pipe1");

        let serialized2 = serialization::encode(&test_msg2).unwrap();
        mock.dispatch("pipe2", Box::new(serialized2.to_vec()))
            .await
            .expect("Failed to dispatch to pipe2");

        // Wait a bit for the messages to be processed
        tokio::time::sleep(Duration::from_millis(100)).await;

        // Check if the messages were received
        let received_msg1 = received1.lock().await.clone();
        assert!(received_msg1.is_some());
        assert_eq!(received_msg1.unwrap(), test_msg1);

        let received_msg2 = received2.lock().await.clone();
        assert!(received_msg2.is_some());
        assert_eq!(received_msg2.unwrap(), test_msg2);

        // Clean up
        mock.disconnect().await.expect("Failed to disconnect");
    }

    #[tokio::test]
    async fn test_mock_error_cases() {
        let mock = Arc::new(Packet::new());

        // Test dispatch before connect
        let test_msg = TestMessage {
            id: 42,
            content: "Hello, HybridPipe!".to_string(),
        };

        let result = mock.dispatch("test-pipe", Box::new(test_msg.clone())).await;
        assert!(result.is_err());

        // Connect
        mock.connect().await.expect("Failed to connect");

        // Test double connect
        let result = mock.connect().await;
        assert!(result.is_err());

        // Test unsubscribe from non-existent pipe
        let result = mock.unsubscribe("non-existent-pipe").await;
        assert!(result.is_err());

        // Subscribe to a pipe
        ext::accept::<TestMessage, _>(&*mock, "test-pipe", |_| {})
            .await
            .expect("Failed to subscribe");

        // Test double subscribe
        let result = ext::accept::<TestMessage, _>(&*mock, "test-pipe", |_| {}).await;
        assert!(result.is_err());

        // Clean up
        mock.disconnect().await.expect("Failed to disconnect");

        // Test disconnect when already disconnected
        let result = mock.disconnect().await;
        assert!(result.is_err());
    }
}
