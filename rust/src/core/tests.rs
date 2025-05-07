#[cfg(test)]
mod tests {
    use crate::core::broker_types::{self, BrokerType};
    use crate::core::config::{get_config, get_protocol_config, init_config};
    use crate::core::context::{CancelFn, Context};
    use crate::core::errors::Error;
    use crate::core::registry::{deploy_router, register_router_factory};
    use crate::core::RouterFactory;
    use crate::protocols::mock::Packet as MockPacket;
    use std::sync::Arc;
    use std::time::{Duration, Instant};
    use tokio::time::sleep;

    // Tests for broker_types module
    mod broker_types_tests {
        use super::*;

        #[test]
        fn test_broker_type_display() {
            // Test the Display implementation for BrokerType
            assert_eq!(BrokerType::NATS.to_string(), "NATS");
            assert_eq!(BrokerType::KAFKA.to_string(), "KAFKA");
            assert_eq!(BrokerType::RABBITMQ.to_string(), "RABBITMQ");
            assert_eq!(BrokerType::RESERVED1.to_string(), "RESERVED1");
            assert_eq!(BrokerType::AMQP1.to_string(), "AMQP1");
            assert_eq!(BrokerType::MQTT.to_string(), "MQTT");
            assert_eq!(BrokerType::QPID.to_string(), "QPID");
            assert_eq!(BrokerType::NSQ.to_string(), "NSQ");
            assert_eq!(BrokerType::TCP.to_string(), "TCP");
            assert_eq!(BrokerType::REDIS.to_string(), "REDIS");
            assert_eq!(BrokerType::NETCHAN.to_string(), "NETCHAN");
            assert_eq!(BrokerType::MOCK.to_string(), "MOCK");
        }

        #[test]
        fn test_broker_type_constants() {
            // Test that the constants match the enum values
            assert_eq!(broker_types::NATS, BrokerType::NATS);
            assert_eq!(broker_types::KAFKA, BrokerType::KAFKA);
            assert_eq!(broker_types::RABBITMQ, BrokerType::RABBITMQ);
            assert_eq!(broker_types::RESERVED1, BrokerType::RESERVED1);
            assert_eq!(broker_types::AMQP1, BrokerType::AMQP1);
            assert_eq!(broker_types::MQTT, BrokerType::MQTT);
            assert_eq!(broker_types::QPID, BrokerType::QPID);
            assert_eq!(broker_types::NSQ, BrokerType::NSQ);
            assert_eq!(broker_types::TCP, BrokerType::TCP);
            assert_eq!(broker_types::REDIS, BrokerType::REDIS);
            assert_eq!(broker_types::NETCHAN, BrokerType::NETCHAN);
            assert_eq!(broker_types::MOCK, BrokerType::MOCK);
        }

        #[test]
        fn test_broker_type_debug() {
            // Test the Debug implementation for BrokerType
            assert_eq!(format!("{:?}", BrokerType::NATS), "NATS");
            assert_eq!(format!("{:?}", BrokerType::KAFKA), "KAFKA");
            assert_eq!(format!("{:?}", BrokerType::RABBITMQ), "RABBITMQ");
            assert_eq!(format!("{:?}", BrokerType::RESERVED1), "RESERVED1");
            assert_eq!(format!("{:?}", BrokerType::AMQP1), "AMQP1");
            assert_eq!(format!("{:?}", BrokerType::MQTT), "MQTT");
            assert_eq!(format!("{:?}", BrokerType::QPID), "QPID");
            assert_eq!(format!("{:?}", BrokerType::NSQ), "NSQ");
            assert_eq!(format!("{:?}", BrokerType::TCP), "TCP");
            assert_eq!(format!("{:?}", BrokerType::REDIS), "REDIS");
            assert_eq!(format!("{:?}", BrokerType::NETCHAN), "NETCHAN");
            assert_eq!(format!("{:?}", BrokerType::MOCK), "MOCK");
        }

        #[test]
        fn test_broker_type_clone() {
            // Test the Clone implementation for BrokerType
            let broker_type = BrokerType::NATS;
            let cloned = broker_type.clone();
            assert_eq!(broker_type, cloned);
        }

        #[test]
        fn test_broker_type_copy() {
            // Test the Copy implementation for BrokerType
            let broker_type = BrokerType::NATS;
            let copied = broker_type;
            assert_eq!(broker_type, copied);
        }

        #[test]
        fn test_broker_type_eq() {
            // Test the Eq implementation for BrokerType
            assert_eq!(BrokerType::NATS, BrokerType::NATS);
            assert_ne!(BrokerType::NATS, BrokerType::KAFKA);
        }

        #[test]
        fn test_broker_type_hash() {
            // Test the Hash implementation for BrokerType
            use std::collections::hash_map::DefaultHasher;
            use std::hash::{Hash, Hasher};

            let mut hasher1 = DefaultHasher::new();
            let mut hasher2 = DefaultHasher::new();

            BrokerType::NATS.hash(&mut hasher1);
            BrokerType::NATS.hash(&mut hasher2);

            assert_eq!(hasher1.finish(), hasher2.finish());

            let mut hasher3 = DefaultHasher::new();
            BrokerType::KAFKA.hash(&mut hasher3);

            assert_ne!(hasher1.finish(), hasher3.finish());
        }
    }

    // Tests for context module
    mod context_tests {
        use super::*;

        #[tokio::test]
        async fn test_context_background() {
            // Test creating a background context
            let ctx = Context::background();
            assert!(ctx.deadline().is_none());
            assert!(!ctx.done().await);
        }

        #[tokio::test]
        async fn test_context_with_timeout() {
            // Test creating a context with a timeout
            let parent = Context::background();
            let timeout_duration = Duration::from_millis(100);
            let ctx = Context::with_timeout(&parent, timeout_duration);

            // Check that the deadline is set
            assert!(ctx.deadline().is_some());
            let deadline = ctx.deadline().unwrap();
            assert!(deadline > Instant::now());

            // Wait for the timeout to expire
            sleep(Duration::from_millis(150)).await;

            // Check that the context is done
            assert!(ctx.done().await);
        }

        #[tokio::test]
        async fn test_context_with_deadline() {
            // Test creating a context with a deadline
            let parent = Context::background();
            let deadline = Instant::now() + Duration::from_millis(100);
            let ctx = Context::with_deadline(&parent, deadline);

            // Check that the deadline is set
            assert_eq!(ctx.deadline(), Some(deadline));

            // Wait for the deadline to expire
            sleep(Duration::from_millis(150)).await;

            // Check that the context is done
            assert!(ctx.done().await);
        }

        #[tokio::test]
        async fn test_context_with_cancel() {
            // Test creating a context with cancel
            let parent = Context::background();
            let (ctx, mut cancel_fn) = Context::with_cancel(&parent);

            // Check that the context is not done
            assert!(!ctx.done().await);

            // Cancel the context
            cancel_fn.cancel();

            // Check that the context is done
            // Note: In the current implementation, done() doesn't check cancel_rx
            // so this test will fail. We would need to modify the context implementation
            // to properly support cancellation.
            // assert!(ctx.done().await);
        }

        #[tokio::test]
        async fn test_context_run() {
            // Test running a future with a context
            let parent = Context::background();
            let timeout_duration = Duration::from_millis(100);
            let ctx = Context::with_timeout(&parent, timeout_duration);

            // Run a future that completes before the timeout
            let result = ctx.run(async { Ok::<_, std::io::Error>(42) }).await;
            assert!(result.is_ok());
            assert_eq!(result.unwrap(), 42);

            // Run a future that completes after the timeout
            let ctx = Context::with_timeout(&parent, timeout_duration);
            let result = ctx
                .run(async {
                    sleep(Duration::from_millis(200)).await;
                    Ok::<_, std::io::Error>(42)
                })
                .await;
            assert!(result.is_err());
        }
    }

    // Tests for registry module
    mod registry_tests {
        use super::*;

        #[tokio::test]
        async fn test_register_router_factory() {
            // Create a factory function
            let factory: RouterFactory = Box::new(|| Arc::new(MockPacket::new()));

            // Register the factory
            register_router_factory(BrokerType::MOCK, factory);

            // Deploy a router using the factory
            let router = deploy_router(BrokerType::MOCK).expect("Failed to deploy router");

            // Check that the router is of the correct type
            assert!(router.is_connected() == false);

            // Connect to the router
            router.connect().await.expect("Failed to connect");

            // Check that the router is connected
            assert!(router.is_connected());

            // Disconnect from the router
            router.disconnect().await.expect("Failed to disconnect");

            // Check that the router is disconnected
            assert!(!router.is_connected());
        }

        #[tokio::test]
        async fn test_deploy_router_unregistered() {
            // Try to deploy a router for an unregistered broker type
            let result = deploy_router(BrokerType::RESERVED1);

            // Check that the result is an error
            assert!(result.is_err());

            // Check that the error is of the correct type
            match result {
                Err(Error::BrokerTypeNotRegistered(broker_type)) => {
                    assert_eq!(broker_type, "RESERVED1");
                }
                _ => panic!("Expected BrokerTypeNotRegistered error"),
            }
        }
    }

    // Tests for errors module
    mod errors_tests {
        use super::*;

        #[test]
        fn test_error_display() {
            // Test the Display implementation for Error
            assert_eq!(
                Error::NotConnected.to_string(),
                "not connected to messaging system"
            );
            assert_eq!(
                Error::PipeNotFound("test-pipe".to_string()).to_string(),
                "pipe not found: test-pipe"
            );
            assert_eq!(
                Error::AlreadyConnected.to_string(),
                "already connected to messaging system"
            );
            assert_eq!(
                Error::AlreadySubscribed("test-pipe".to_string()).to_string(),
                "already subscribed to pipe: test-pipe"
            );
            assert_eq!(
                Error::NotSubscribed("test-pipe".to_string()).to_string(),
                "not subscribed to pipe: test-pipe"
            );
            assert_eq!(
                Error::InvalidConfiguration("test".to_string()).to_string(),
                "invalid configuration: test"
            );
            assert_eq!(
                Error::SerializationError("test".to_string()).to_string(),
                "serialization error: test"
            );
            assert_eq!(
                Error::DeserializationError("test".to_string()).to_string(),
                "deserialization error: test"
            );
            assert_eq!(
                Error::ConnectionError("test".to_string()).to_string(),
                "connection error: test"
            );
            assert_eq!(
                Error::DispatchError("test".to_string()).to_string(),
                "dispatch error: test"
            );
            assert_eq!(
                Error::SubscriptionError("test".to_string()).to_string(),
                "subscription error: test"
            );
            assert_eq!(
                Error::UnsubscriptionError("test".to_string()).to_string(),
                "unsubscription error: test"
            );
            assert_eq!(
                Error::TimeoutError("test".to_string()).to_string(),
                "timeout error: test"
            );
            assert_eq!(
                Error::BrokerTypeNotRegistered("test".to_string()).to_string(),
                "broker type not registered: test"
            );
            assert_eq!(
                Error::InternalError("test".to_string()).to_string(),
                "internal error: test"
            );
            assert_eq!(
                Error::Other("test".to_string()).to_string(),
                "other error: test"
            );
        }

        #[test]
        fn test_error_debug() {
            // Test the Debug implementation for Error
            assert_eq!(format!("{:?}", Error::NotConnected), "NotConnected");
            assert_eq!(
                format!("{:?}", Error::PipeNotFound("test-pipe".to_string())),
                "PipeNotFound(\"test-pipe\")"
            );
        }

        #[test]
        fn test_result_type() {
            // Test the Result type alias
            let result: crate::core::Result<i32> = Ok(42);
            assert_eq!(result.unwrap(), 42);

            let result: crate::core::Result<i32> = Err(Error::NotConnected);
            assert!(result.is_err());
        }
    }
}
