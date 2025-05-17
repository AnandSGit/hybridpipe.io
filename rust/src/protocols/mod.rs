// Protocols module for HybridPipe
//
// This module contains implementations of the HybridPipe interface for various messaging protocols.

pub mod amqp;
pub mod kafka;
pub mod mock;
#[cfg(test)]
pub mod mocks;
pub mod mqtt;
pub mod nats;
pub mod netchan;
pub mod nsq;
pub mod qpid;
pub mod rabbitmq;
pub mod redis;
pub mod tcp;
pub mod zeromq;

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
    zeromq::register();
    netchan::register();
    mock::register();
}
