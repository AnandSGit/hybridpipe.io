#[cfg(test)]
mod tests {
    use async_trait::async_trait;
    use serde::{Deserialize, Serialize};
    use std::any::Any;
    use std::sync::{Arc, Mutex};
    use std::time::Duration;
    use tokio::sync::Mutex as TokioMutex;

    use crate::core::{deploy_router, BrokerType, Error, HybridPipe, Process, Result};
    use crate::middleware::{Middleware, MiddlewareStack};
    use crate::protocols::mock::Packet;

    #[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
    struct TestMessage {
        id: u32,
        content: String,
    }

    // A simple test middleware that counts calls
    #[derive(Clone)]
    struct CounterMiddleware {
        before_dispatch_count: Arc<Mutex<u32>>,
        after_dispatch_count: Arc<Mutex<u32>>,
        before_process_count: Arc<Mutex<u32>>,
        after_process_count: Arc<Mutex<u32>>,
        inner: Arc<dyn HybridPipe>,
    }

    impl CounterMiddleware {
        fn new(inner: Arc<dyn HybridPipe>) -> Self {
            Self {
                before_dispatch_count: Arc::new(Mutex::new(0)),
                after_dispatch_count: Arc::new(Mutex::new(0)),
                before_process_count: Arc::new(Mutex::new(0)),
                after_process_count: Arc::new(Mutex::new(0)),
                inner,
            }
        }

        fn get_before_dispatch_count(&self) -> u32 {
            *self.before_dispatch_count.lock().unwrap()
        }

        fn get_after_dispatch_count(&self) -> u32 {
            *self.after_dispatch_count.lock().unwrap()
        }

        fn get_before_process_count(&self) -> u32 {
            *self.before_process_count.lock().unwrap()
        }

        fn get_after_process_count(&self) -> u32 {
            *self.after_process_count.lock().unwrap()
        }
    }

    #[async_trait]
    impl Middleware for CounterMiddleware {
        fn inner(&self) -> &dyn HybridPipe {
            self.inner.as_ref()
        }

        async fn before_dispatch(
            &self,
            _pipe: &str,
            _data: &Box<dyn Any + Send + Sync>,
        ) -> Result<()> {
            let mut count = self.before_dispatch_count.lock().unwrap();
            *count += 1;
            Ok(())
        }

        async fn after_dispatch(
            &self,
            _pipe: &str,
            _data: &Box<dyn Any + Send + Sync>,
            _result: &Result<()>,
        ) {
            let mut count = self.after_dispatch_count.lock().unwrap();
            *count += 1;
        }

        async fn before_process(&self, _pipe: &str, _data: &[u8]) -> Result<()> {
            let mut count = self.before_process_count.lock().unwrap();
            *count += 1;
            Ok(())
        }

        async fn after_process(&self, _pipe: &str, _data: &[u8], _result: &Result<()>) {
            let mut count = self.after_process_count.lock().unwrap();
            *count += 1;
        }
    }

    #[tokio::test]
    async fn test_middleware_stack() {
        // Create a mock router
        let mock = Arc::new(Packet::new());
        mock.connect().await.expect("Failed to connect");

        // Create a middleware
        let counter = CounterMiddleware::new(mock.clone());

        // Create a middleware stack
        let mut stack = MiddlewareStack::new(mock.clone());
        stack.add_middleware(counter.clone());

        // Create a test message
        let test_msg = TestMessage {
            id: 42,
            content: "Hello, HybridPipe!".to_string(),
        };

        // Create a shared variable to store the received message
        let received = Arc::new(TokioMutex::new(None));
        let received_clone = received.clone();

        // Subscribe to a pipe using the ext module
        crate::core::ext::accept::<TestMessage, _>(&stack, "test-pipe", move |data| {
            if let Some(msg) = data.downcast_ref::<TestMessage>() {
                let mut received = received_clone.blocking_lock();
                *received = Some(msg.clone());
            }
        })
        .await
        .expect("Failed to subscribe");

        // Dispatch a message
        stack
            .dispatch("test-pipe", Box::new(test_msg.clone()))
            .await
            .expect("Failed to dispatch");

        // Wait a bit for the message to be processed
        tokio::time::sleep(Duration::from_millis(100)).await;

        // Check if the message was received
        let received_msg = received.lock().await.clone();
        assert!(received_msg.is_some());
        assert_eq!(received_msg.unwrap(), test_msg);

        // Check if the middleware was called
        assert_eq!(counter.get_before_dispatch_count(), 1);
        assert_eq!(counter.get_after_dispatch_count(), 1);
        assert_eq!(counter.get_before_process_count(), 1);
        assert_eq!(counter.get_after_process_count(), 1);

        // Clean up
        stack.disconnect().await.expect("Failed to disconnect");
    }

    #[tokio::test]
    async fn test_middleware_chain() {
        // Create a mock router
        let mock = Arc::new(Packet::new());
        mock.connect().await.expect("Failed to connect");

        // Create multiple middleware instances
        let counter1 = CounterMiddleware::new(mock.clone());
        let counter2 = CounterMiddleware::new(mock.clone());

        // Create a middleware stack with multiple middleware
        let mut stack = MiddlewareStack::new(mock.clone());
        stack.add_middleware(counter1.clone());
        stack.add_middleware(counter2.clone());

        // Create a test message
        let test_msg = TestMessage {
            id: 42,
            content: "Hello, HybridPipe!".to_string(),
        };

        // Dispatch a message
        stack
            .dispatch("test-pipe", Box::new(test_msg.clone()))
            .await
            .expect("Failed to dispatch");

        // Wait a bit for the message to be processed
        tokio::time::sleep(Duration::from_millis(100)).await;

        // Check if both middleware were called
        assert_eq!(counter1.get_before_dispatch_count(), 1);
        assert_eq!(counter1.get_after_dispatch_count(), 1);
        assert_eq!(counter2.get_before_dispatch_count(), 1);
        assert_eq!(counter2.get_after_dispatch_count(), 1);

        // Clean up
        stack.disconnect().await.expect("Failed to disconnect");
    }

    #[tokio::test]
    async fn test_middleware_error_handling() {
        // Create a mock router
        let mock = Arc::new(Packet::new());
        mock.connect().await.expect("Failed to connect");

        // Create a middleware that returns an error
        struct ErrorMiddleware;

        #[async_trait]
        impl Middleware for ErrorMiddleware {
            fn inner(&self) -> &dyn HybridPipe {
                panic!("ErrorMiddleware doesn't wrap a HybridPipe directly");
            }

            async fn before_dispatch(
                &self,
                _pipe: &str,
                _data: &Box<dyn Any + Send + Sync>,
            ) -> Result<()> {
                Err(Error::Other("Test error".to_string()))
            }
        }

        // Create a middleware stack with the error middleware
        let mut stack = MiddlewareStack::new(mock.clone());
        stack.add_middleware(ErrorMiddleware);

        // Create a test message
        let test_msg = TestMessage {
            id: 42,
            content: "Hello, HybridPipe!".to_string(),
        };

        // Dispatch a message, which should fail
        let result = stack
            .dispatch("test-pipe", Box::new(test_msg.clone()))
            .await;
        assert!(result.is_err());

        // Clean up
        stack.disconnect().await.expect("Failed to disconnect");
    }
}
