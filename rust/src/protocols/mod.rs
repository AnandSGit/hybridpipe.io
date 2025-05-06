// Protocols module for HybridPipe
//
// This module contains implementations of the HybridPipe interface for various messaging protocols.

pub mod kafka;
pub mod nats;
pub mod mqtt;
pub mod rabbitmq;
pub mod amqp;
pub mod qpid;
pub mod nsq;
pub mod tcp;
pub mod redis;
pub mod netchan;
pub mod mock;

// Register all protocol implementations
pub fn register_all() {
    kafka::register();
    nats::register();
    mqtt::register();
    rabbitmq::register();
    amqp::register();
    qpid::register();
    nsq::register();
    tcp::register();
    redis::register();
    netchan::register();
    mock::register();
}
